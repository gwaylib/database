package database

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/gwaylib/errors"
	"github.com/jmoiron/sqlx/reflectx"
)

// 通用的字符串查询
type DBData string

func (d *DBData) Scan(i interface{}) error {
	if i == nil {
		return nil
	}
	switch i.(type) {
	case int64:
		*d = DBData(fmt.Sprintf("%d", i))
	case float64:
		*d = DBData(fmt.Sprint(i))
	case []byte:
		*d = DBData(string(i.([]byte)))
	case string:
		*d = DBData(i.(string))
	case bool:
		*d = DBData(fmt.Sprintf("%t", i))
	case time.Time:
		*d = DBData(i.(time.Time).Format("2006-01-02 15:04:05"))
	default:
		*d = DBData(fmt.Sprint(i))
	}
	return nil
}
func (d *DBData) String() string {
	return string(*d)
}

func makeDBData(l int) []interface{} {
	r := make([]interface{}, l)
	for i := 0; i < l; i++ {
		d := DBData("")
		r[i] = &d
	}
	return r
}

type Template struct {
	CountSql string // 读取数据总行数
	DataSql  string // 读取数据细节
}

// 返回一个fmt.Sprintf()格式化Sql后的Template，
// 主要用于分表的读取
func (t Template) FmtTemplate(args ...interface{}) *Template {
	countSql := t.CountSql
	if len(countSql) > 0 {
		countSql = fmt.Sprintf(t.CountSql, args...)
	}
	dataSql := t.DataSql
	if len(dataSql) > 0 {
		dataSql = fmt.Sprintf(t.DataSql, args...)
	}

	return &Template{
		CountSql: countSql,
		DataSql:  dataSql,
	}
}

var refxM = reflectx.NewMapperTagFunc("db", func(in string) string {
	// for tag name
	return in
}, func(in string) string {
	// for options
	trims := []string{}
	options := strings.Split(in, ",")
	for _, op := range options {
		trims = append(trims, strings.TrimSpace(op))
	}
	return strings.Join(trims, ",")
})

func travelChild(f *reflectx.FieldInfo, v reflect.Value, order *int, drvName *string, outputNames *[]byte, outputInputs *[]byte, outputVals *[]interface{}) {
	*order += 1
	childrenLen := len(f.Children)
	isDriverValue := false
	switch v.Kind() {
	case reflect.Invalid:
		// nil value
		return
	case
		reflect.Bool,
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
		reflect.Float32,
		reflect.Float64,
		reflect.String:
		isDriverValue = true
	default:
		switch v.Type().String() {
		case "[]uint8", "time.Time":
			isDriverValue = true
		}

	}
	if isDriverValue {
		_, ok := f.Options["autoincrement"]
		if ok {
			// ignore 'autoincrement' for insert data
			return
		}

		*outputNames = append(*outputNames, []byte(f.Name+",")...)
		*outputVals = append(*outputVals, v.Interface())
		switch {
		case strings.Index(*drvName, "oracle") > -1, strings.Index(*drvName, "oci8") > -1:
			*outputInputs = append(*outputInputs, []byte(fmt.Sprintf(":%s,", f.Name))...)
		case strings.Index(*drvName, "postgres") > -1:
			*outputInputs = append(*outputInputs, []byte(fmt.Sprintf(":%d,", *order-1))...)
		case strings.Index(*drvName, "sqlserver") > -1, strings.Index(*drvName, "mssql") > -1:
			*outputInputs = append(*outputInputs, []byte(fmt.Sprintf("@p%d,", *order-1))...)
		default:
			*outputInputs = append(*outputInputs, []byte("?,")...)
		}

		return
	}

	for i := 0; i < childrenLen; i++ {
		child := f.Children[i]
		if child == nil {
			// found ignore tag, do next.
			continue
		}
		travelChild(
			child,
			reflect.Indirect(v.Field(i)),
			order, drvName,
			outputNames, outputInputs, outputVals,
		)
	}
}

func reflectInsertStruct(i interface{}, drvName string) (string, string, []interface{}, error) {
	v := reflect.ValueOf(i)
	k := v.Kind()
	switch k {
	case reflect.Ptr:
	default:
		return "", "", nil, errors.New("Unsupport reflect type").As(k.String())
	}
	v = reflect.Indirect(v)

	tm := refxM.TypeMap(v.Type())

	names := []byte{}
	inputs := []byte{}
	vals := []interface{}{}
	order := 0
	travelChild(tm.Tree, v, &order, &drvName, &names, &inputs, &vals)

	if len(names) == 0 {
		panic("No public field in struct")
	}
	return string(names[:len(names)-1]), string(inputs[:len(inputs)-1]), vals, nil
}
