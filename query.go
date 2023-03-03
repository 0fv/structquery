package structquery

import (
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

/* */
func Where(i interface{}, count ...*int64) func(d *gorm.DB) *gorm.DB {
	return func(d *gorm.DB) *gorm.DB {
		nameFormat := d.Config.NamingStrategy
		dataMap, order, page, limit := toScopeWhereMap(i, nameFormat)
		for _, v := range order {
			if data, ok := dataMap[v]; ok {
				if mp, ok := data[0].(map[string][]interface{}); ok {
					d = d.Scopes(subScopes(mp))
				} else {
					d = d.Where(v, dataMap[v]...)
				}
			}
		}
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

func subScopes(m map[string][]interface{}) func(d *gorm.DB) *gorm.DB {
	return func(d *gorm.DB) *gorm.DB {
		t := d.Session(&gorm.Session{
			NewDB: true,
		})

		set := false
		for k, v := range m {
			if mp, ok := v[0].(map[string][]interface{}); ok {
				d = d.Scopes(subScopes(mp))
			} else {
				t = t.Or(k, v...)
				set = true
			}
		}
		if set {
			d = d.Where(t)
		}
		return d
	}
}

func toScopeWhereMap(i interface{}, nameFormat schema.Namer) (map[string][]interface{}, []string, int, int) {
	ret := make(map[string][]interface{})
	order := make([]string, 0)
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
			if fieldType.Anonymous {
				m, _, p, lm := toScopeWhereMap(fieldValue.Interface(), nameFormat)
				for k, v := range m {
					ret[k] = v
				}

				if p == 0 || lm == 0 {
					ret[fieldType.Name] = append(ret[fieldType.Name], m)
					order = append(order, fieldType.Name)
				}
			} else {
				m, _, _, _ := toScopeWhereMap(fieldValue.Interface(), nameFormat)
				ret[fieldType.Name] = append(ret[fieldType.Name], m)
				order = append(order, fieldType.Name)
			}
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
		for i := 0; i < count; i++ {
			//between 需要将数组拆开
			switch {
			case strings.Contains(fieldTag, "between"):
				if fieldValue.Kind() == reflect.Slice && fieldValue.Len() == 2 {
					ret[whereStr] = append(ret[whereStr], fieldValue.Index(0).Interface(), fieldValue.Index(1).Interface())
				}
			//like
			case fieldTag == "like":
				ret[whereStr] = append(ret[whereStr], "%"+fmt.Sprint(fieldValue.Interface())+"%")
			default:
				ret[whereStr] = append(ret[whereStr], fieldValue.Interface())
			}
		}
		order = append(order, whereStr)
	}
	return ret, order, page, limit
}

func symbolToText(symbol rune) string {
	switch symbol {
	case '&':
		return " and "
	case '|':
		return " or "
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
		wherestr = fieldName + " " + fieldTag + " ?"
		return
	case "in", "not in":
		wherestr = fieldName + " " + fieldTag + " (?)"
		return
	case "between", "not between":
		wherestr = fieldName + " " + fieldTag + " ? and ?"
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
}

func checkInt(f reflect.StructField) bool {

	for _, v := range includeType {
		if f.Type.Kind() == v {
			return true
		}
	}
	return false
}
