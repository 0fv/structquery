package structquery

import (
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type cdx uint8

const (
	or cdx = iota + 1
	and
)

type scopeWheredata struct {
	wherestr string
	val      []interface{}
}

type scopeWhereGroup struct {
	combainCdx cdx
	data       []scopeWheredata
	sub        []scopeWhereGroup
}

func Where(i interface{}, count ...*int64) func(d *gorm.DB) *gorm.DB {
	return func(d *gorm.DB) *gorm.DB {
		nameFormat := d.Config.NamingStrategy
		dataMap, page, limit := toScopeWhereMap(i, nameFormat)
		d.Scopes(dataMap.scopes())
		if len(count) == 1 {
			t := d.Session(&gorm.Session{})
			t.Count(count[0])
		}
		if page > 0 && limit > 0 {
			d.Limit(limit).Offset((page - 1) * limit)
		}
		return d
	}
}

func (s *scopeWhereGroup) scopes() func(d *gorm.DB) *gorm.DB {
	return func(d *gorm.DB) *gorm.DB {
		switch s.combainCdx {
		case and:
			for _, v := range s.data {
				d.Where(v.wherestr, v.val...)
			}
			for _, v := range s.sub {
				d.Where(v.scopesSub(d))
			}
		case or:
			for _, v := range s.data {
				d.Or(v.wherestr, v.val...)
			}
			for _, v := range s.sub {
				d.Or(v.scopesSub(d))
			}
		}
		return d
	}
}

func (s *scopeWhereGroup) scopesSub(db *gorm.DB) *gorm.DB {
	db = db.Session(&gorm.Session{
		NewDB: true,
	})
	switch s.combainCdx {
	case and:
		for _, v := range s.data {
			db = db.Where(v.wherestr, v.val...)
		}
		for _, v := range s.sub {
			db = db.Where(v.scopesSub(db))
		}
	case or:
		for _, v := range s.data {
			db = db.Or(v.wherestr, v.val...)
		}
		for _, v := range s.sub {
			db = db.Or(v.scopesSub(db))
		}
	}
	return db
}

func toScopeWhereMap(i interface{}, nameFormat schema.Namer) (scopeWhereGroup, int, int) {
	ret := scopeWhereGroup{
		combainCdx: and,
	}
	var page int
	var limit int
	refv := reflect.ValueOf(i)
	reft := reflect.TypeOf(i)
	if refv.Kind() == reflect.Ptr {
		refv = refv.Elem()
		reft = reft.Elem()
	}
	fieldlength := refv.NumField()
	for i := 0; i < fieldlength; i++ {
		fieldValue := refv.Field(i)
		fieldType := reft.Field(i)
		//0 值跳过
		if fieldValue.IsZero() ||
			(fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil()) ||
			(fieldValue.Kind() == reflect.Slice && fieldValue.Len() == 0) {
			continue
		}
		if fieldValue.Kind() == reflect.Struct {
			m, _, _ := toScopeWhereMap(fieldValue.Interface(), nameFormat)
			if fieldType.Anonymous {
				m.combainCdx = or
			}
			ret.sub = append(ret.sub, m)
			continue
		}
		field := refv.Type().Field(i)
		fieldName := nameFormat.ColumnName("", field.Name)
		//分页
		if fieldName == "page" {
			isNum := checkInt(field)
			if isNum {
				page = int(fieldValue.Int())
				continue
			}
		}
		if fieldName == "size" {
			isNum := checkInt(field)
			if isNum {
				limit = int(fieldValue.Int())
				continue
			}
		}
		fieldTag := "="
		if fieldValue.Kind() == reflect.Slice {
			n := fieldValue.Type().Elem().Name()
			p := fieldValue.Type().Elem().PkgPath()
			if n == "Time" && p == "time" {
				fieldTag = "between"
			} else {
				fieldTag = "in"
			}
		}
		if v := field.Tag.Get("op"); v != "-" {
			if v != "" {
				fieldTag = v
			}
		} else {
			continue
		}
		if v, ok := field.Tag.Lookup("field"); ok {
			if v != "" {
				fieldName = v
			}
		}
		whereStr, count := paresWhere(fieldName, fieldTag)
		val := make([]interface{}, 0)
		for i := 0; i < count; i++ {
			//between 需要将切片拆开
			switch {
			case strings.Contains(fieldTag, "between"):
				if fieldValue.Kind() == reflect.Slice && fieldValue.Len() == 2 {
					val = append(val, fieldValue.Index(0).Interface(), fieldValue.Index(1).Interface())
				}
			//like
			case fieldTag == "like":
				str := fmt.Sprint(fieldValue.Interface())
				if (!strings.Contains(str, "%")) || (strings.Contains(str, "%") && strings.Count(str, "%") == strings.Count(str, `\%`)) {
					str = "%" + str + "%"
				}
				val = append(val, str)
			default:
				val = append(val, fieldValue.Interface())
			}
		}
		ret.data = append(ret.data, scopeWheredata{
			wherestr: whereStr,
			val:      val,
		})
	}
	return ret, page, limit
}

func symbolToText(symbol rune) string {
	switch symbol {
	case '&':
		return " AND "
	case '|':
		return " OR "
	}
	return ""
}

func paresWhere(fieldName, fieldTag string) (whereStr string, count int) {
	count++
	start := 0
	privOp := ""
	for i, v := range fieldName {
		if v == '&' || v == '|' {
			count++
			whereStr += (privOp + parseWherefield(fieldName[start:i], fieldTag))
			start = i + 1
			privOp = symbolToText(v)
		}
		if len(fieldName)-1 == i {
			whereStr += (privOp + parseWherefield(fieldName[start:i+1], fieldTag))
		}
	}
	return
}

func parseWherefield(fieldName, fieldTag string) (wherestr string) {
	switch fieldTag {
	case "=", "!=", ">", "<", ">=", "<=", "like":
		wherestr = fieldName + " " + strings.ToUpper(fieldTag) + " ?"
		return
	case "in", "not in":
		wherestr = fieldName + " " + strings.ToUpper(fieldTag) + " (?)"
		return
	case "between", "not between":
		wherestr = fieldName + " " + strings.ToUpper(fieldTag) + " ? AND ?"
		return
	}
	if strings.Contains(fieldTag, "?") {
		wherestr = fieldTag
	}
	return
}

var includeType = []reflect.Kind{
	reflect.Int,
	reflect.Int8,
	reflect.Int16,
	reflect.Int32,
	reflect.Int64,
	reflect.Uint,
	reflect.Uint8,
	reflect.Uint16,
	reflect.Uint32,
	reflect.Uint64,
}

func checkInt(f reflect.StructField) bool {

	for _, v := range includeType {
		if f.Type.Kind() == v {
			return true
		}
	}
	return false
}
