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

type scopeCdxGroup struct {
	combainCdx cdx
	data       []*scopeWheredata
	sub        []*scopeCdxGroup

	orderField []string
	page       int
	size       int
}

func Where(i interface{}, count ...*int64) func(d *gorm.DB) *gorm.DB {
	return func(d *gorm.DB) *gorm.DB {
		nameFormat := d.Config.NamingStrategy
		dialector := d.Dialector
		dataMap := toScopeWhereMap(i, nameFormat, dialector)
		d.Scopes(dataMap.scopes())
		if len(count) == 1 {
			t := d.Session(&gorm.Session{})
			t.Count(count[0])
		}
		if dataMap.size != 0 {
			d.Limit(dataMap.size)
		}
		if dataMap.page != 0 {
			d.Offset((dataMap.page - 1) * dataMap.size)
		}
		return d
	}
}

func (s *scopeCdxGroup) scopes() func(d *gorm.DB) *gorm.DB {
	return func(d *gorm.DB) *gorm.DB {
		switch s.combainCdx {
		case and:
			for _, v := range s.data {
				d.Where(v.wherestr, v.val...)
			}
			for _, v := range s.sub {
				d.Where(v.scopesSub(d, s))
			}
		case or:
			for _, v := range s.data {
				d.Or(v.wherestr, v.val...)
			}
			for _, v := range s.sub {
				d.Or(v.scopesSub(d, s))
			}
		}
		d.Order(strings.Join(s.orderField, ","))
		return d
	}
}

func (s *scopeCdxGroup) scopesSub(db *gorm.DB, p *scopeCdxGroup) *gorm.DB {
	db = db.Session(&gorm.Session{
		NewDB: true,
	})
	switch s.combainCdx {
	case and:
		for _, v := range s.data {
			db = db.Where(v.wherestr, v.val...)
		}
		for _, v := range s.sub {
			db = db.Where(v.scopesSub(db, v))
		}
	case or:
		for _, v := range s.data {
			db = db.Or(v.wherestr, v.val...)
		}
		for _, v := range s.sub {
			db = db.Or(v.scopesSub(db, v))
		}
	}
	if len(s.orderField) != 0 {
		p.orderField = append(p.orderField, s.orderField...)
	}

	if p.size == 0 && s.size != 0 {
		p.size = s.size
	}
	if p.page == 0 && s.page != 0 {
		p.page = s.page
	}
	return db
}

func toScopeWhereMap(i interface{}, nameFormat schema.Namer, dialector gorm.Dialector) *scopeCdxGroup {
	ret := &scopeCdxGroup{
		combainCdx: and,
	}
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
		if fieldValue.Kind() == reflect.Ptr {
			fieldValue = fieldValue.Elem()
		}
		if fieldValue.Kind() == reflect.Struct {
			m := toScopeWhereMap(fieldValue.Interface(), nameFormat, dialector)
			if fieldType.Anonymous {
				m.combainCdx = or
			}
			ret.sub = append(ret.sub, m)
			continue
		}
		field := refv.Type().Field(i)
		fieldName := nameFormat.ColumnName("", field.Name)
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
		switch fieldTag {
		case "page":
			if checkInt(field) {
				ret.page = int(fieldValue.Int())
			}
			if checkUint(field) {
				ret.page = int(fieldValue.Uint())
			}
		case "size":
			if checkInt(field) {
				ret.size = int(fieldValue.Int())
			}
			if checkUint(field) {
				ret.size = int(fieldValue.Uint())
			}
		case "asc":
			fv := fieldValue.Interface()
			asc, ok := fv.(bool)
			if ok {
				ret.orderField = append(ret.orderField, parseOrderField(fieldName, dialector, asc)...)
			}
		case "desc":
			fv := fieldValue.Interface()
			desc, ok := fv.(bool)
			if ok {
				ret.orderField = append(ret.orderField, parseOrderField(fieldName, dialector, !desc)...)
			}
		default:
			whereStr, count := paresWhere(fieldName, fieldTag, dialector)
			val := make([]interface{}, 0)
			for i := 0; i < count; i++ {
				//between 需要将切片拆开
				switch fieldTag {
				case "between", "not between":
					if fieldValue.Kind() == reflect.Slice && fieldValue.Len() == 2 {
						val = append(val, fieldValue.Index(0).Interface(), fieldValue.Index(1).Interface())
					}
				//like
				case "like":
					str := fmt.Sprint(fieldValue.Interface())
					if (!strings.Contains(str, "%")) || (strings.Contains(str, "%") && strings.Count(str, "%") == strings.Count(str, `\%`)) {
						str = "%" + str + "%"
					}
					val = append(val, str)
				case "null":
					fv := fieldValue.Interface()
					isnull, ok := fv.(bool)
					if ok && (!isnull) {
						whereStr = strings.ReplaceAll(whereStr, "IS NULL", "IS NOT NULL")
					}
				case "not null":
					fv := fieldValue.Interface()
					notnull, ok := fv.(bool)
					if ok && (!notnull) {
						whereStr = strings.ReplaceAll(whereStr, "IS NOT NULL", "IS NULL")
					}

				default:
					val = append(val, fieldValue.Interface())
				}
			}
			if whereStr != "" {
				ret.data = append(ret.data, &scopeWheredata{
					wherestr: whereStr,
					val:      val,
				})
			}
		}

	}
	return ret
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

func parseOrderField(fieldName string, dialector gorm.Dialector, asc bool) []string {
	sb := &strings.Builder{}
	orderway := " DESC"
	if asc {
		orderway = " ASC"
	}
	var ret []string
	for _, v := range strings.Split(fieldName, ",") {
		dialector.QuoteTo(sb, v)
		sb.WriteString(orderway)
		ret = append(ret, sb.String())
		sb.Reset()
	}
	return ret
}

func paresWhere(fieldName, fieldTag string, dialector gorm.Dialector) (whereStr string, count int) {
	count++
	start := 0
	privOp := ""
	leftstr := ""
	rightstr := ""
	sb := &strings.Builder{}
	for i, v := range fieldName {
		if v == '&' || v == '|' {
			count++
			fnstr := fieldName[start:i]
			leftstr, rightstr, fnstr = splitBrackets(fnstr)
			dialector.QuoteTo(sb, fnstr)
			fn := sb.String()
			sb.Reset()
			whereStr += privOp + leftstr + parseWherefield(fn, fieldTag) + rightstr
			start = i + 1
			privOp = symbolToText(v)
			leftstr = ""
			rightstr = ""
		}
		if len(fieldName)-1 == i {
			fnstr := fieldName[start : i+1]
			leftstr, rightstr, fnstr = splitBrackets(fnstr)
			dialector.QuoteTo(sb, fnstr)
			fn := sb.String()
			sb.Reset()
			whereStr += (privOp + leftstr + parseWherefield(fn, fieldTag)) + rightstr
		}
	}
	return
}

func splitBrackets(str string) (left, right, center string) {
	str = strings.TrimLeftFunc(str, func(r rune) bool {
		if r == '(' {
			left += "("
			return true
		}
		return false
	})
	center = strings.TrimRightFunc(str, func(r rune) bool {
		if r == ')' {
			right += ")"
			return true
		}
		return false
	})
	return
}

func parseWherefield(fieldName, fieldTag string) (wherestr string) {
	switch fieldTag {
	case "=", "!=", ">", "<", ">=", "<=", "like":
		wherestr = fieldName + " " + strings.ToUpper(fieldTag) + " ?"
	case "in", "not in":
		wherestr = fieldName + " " + strings.ToUpper(fieldTag) + " (?)"
	case "between", "not between":
		wherestr = fieldName + " " + strings.ToUpper(fieldTag) + " ? AND ?"
	case "null", "not null":
		wherestr = fieldName + " IS " + strings.ToUpper(fieldTag)
	default:
		if strings.Contains(fieldTag, "?") {
			wherestr = fieldTag
		}
	}
	return
}

var includeIntType = []reflect.Kind{
	reflect.Int,
	reflect.Int8,
	reflect.Int16,
	reflect.Int32,
	reflect.Int64,
}

var includeUintType = []reflect.Kind{
	reflect.Uint,
	reflect.Uint8,
	reflect.Uint16,
	reflect.Uint32,
	reflect.Uint64,
}

func checkInt(f reflect.StructField) bool {
	for _, v := range includeIntType {
		if f.Type.Kind() == v {
			return true
		}
	}
	return false
}

func checkUint(f reflect.StructField) bool {
	for _, v := range includeUintType {
		if f.Type.Kind() == v {
			return true
		}
	}
	return false
}
