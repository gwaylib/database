package database

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/gwaylib/errors"
	"github.com/jmoiron/sqlx/reflectx"
)

// Bool type
type Bool bool

func (v *Bool) Scan(i interface{}) error {
	b := sql.NullBool{}
	if err := b.Scan(i); err != nil {
		return err
	}
	*v = Bool(b.Bool)
	return nil
}
func (v *Bool) Value() (driver.Value, error) {
	return v, nil
}

// Int64 type
type Int64 int64

func (v *Int64) Scan(i interface{}) error {
	b := sql.NullInt64{}
	if err := b.Scan(i); err != nil {
		return err
	}
	*v = Int64(b.Int64)
	return nil
}
func (v Int64) Value() (driver.Value, error) {
	return int64(v), nil
}

// Float64 type
type Float64 float64

func (v *Float64) Scan(i interface{}) error {
	b := sql.NullFloat64{}
	if err := b.Scan(i); err != nil {
		return err
	}
	*v = Float64(b.Float64)
	return nil
}
func (v Float64) Value() (driver.Value, error) {
	return float64(v), nil
}

// String type
type String string

func (v *String) Scan(i interface{}) error {
	b := sql.NullString{}
	if err := b.Scan(i); err != nil {
		return err
	}
	*v = String(b.String)
	return nil
}
func (v String) Value() (driver.Value, error) {
	return string(v), nil
}
func (v *String) String() string {
	return string(*v)
}

// 通用的字符串查询
type DBData string

func (d *DBData) Scan(i interface{}) error {
	if i == nil {
		*d = ""
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
		*d = DBData(i.(time.Time).Format(time.RFC3339))
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

// return is it a auto_increment field
func travelStructField(f *reflectx.FieldInfo, v *reflect.Value, order *int, drvName *string, outputNames *[]byte, outputInputs *[]byte, outputVals *[]interface{}) *reflect.Value {
	*order += 1
	switch v.Kind() {
	case reflect.Invalid:
		// nil value
		return nil
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
		// continue
		break
	case reflect.Struct, reflect.Ptr:
		if _, ok := v.Interface().(driver.Valuer); ok {
			break
		}
		switch v.Type().String() {
		case "time.Time":
			break
		default:
			var autoIncrement *reflect.Value
			childrenLen := len(f.Children)
			for i := 0; i < childrenLen; i++ {
				child := f.Children[i]
				if child == nil {
					// found ignore tag, do next.
					continue
				}
				fieldVal := reflect.Indirect(*v).Field(i)
				autoFiled := travelStructField(
					child,
					&fieldVal,
					order, drvName,
					outputNames, outputInputs, outputVals,
				)
				if autoFiled != nil {
					autoIncrement = autoFiled
				}
			}
			return autoIncrement
		}
	default:
		// unsupport
		switch v.Type().String() {
		case "[]uint8":
			break
		default:
			return nil
		}
	}

	//
	// decode fileds
	//

	_, ok := f.Options["autoincrement"]
	if ok {
		// ignore 'autoincrement' for insert data
		return v
	}
	_, ok = f.Options["auto_increment"]
	if ok {
		// ignore 'auto_increment' for insert data
		return v
	}

	*outputVals = append(*outputVals, v.Interface())
	switch {
	case strings.Index(*drvName, "oracle") > -1, strings.Index(*drvName, "oci8") > -1:
		*order += 1
		*outputNames = append(*outputNames, []byte("\""+f.Name+"\",")...)
		*outputInputs = append(*outputInputs, []byte(fmt.Sprintf(":%s,", f.Name))...)
	case strings.Index(*drvName, "postgres") > -1:
		*outputNames = append(*outputNames, []byte("\""+f.Name+"\",")...)
		*outputInputs = append(*outputInputs, []byte(fmt.Sprintf(":%d,", *order))...)
		*order += 1
	case strings.Index(*drvName, "sqlserver") > -1, strings.Index(*drvName, "mssql") > -1:
		*outputNames = append(*outputNames, []byte("["+f.Name+"],")...)
		*outputInputs = append(*outputInputs, []byte(fmt.Sprintf("@p%d,", *order))...)
		*order += 1
	case strings.Index(*drvName, "mysql") > -1:
		*order += 1
		*outputNames = append(*outputNames, []byte("`"+f.Name+"`,")...)
		*outputInputs = append(*outputInputs, []byte("?,")...)
	default:
		*outputNames = append(*outputNames, []byte("\""+f.Name+"\",")...)
		*outputInputs = append(*outputInputs, []byte("?,")...)
	}

	return nil
}

type reflectInsertField struct {
	Names  string
	Stmts  string
	Values []interface{}

	AutoIncrement *reflect.Value
}

func (r *reflectInsertField) SetAutoIncrement(v reflect.Value) {
	if r.AutoIncrement == nil {
		return
	}
	r.AutoIncrement.Set(v)
}

func reflectInsertStruct(i interface{}, drvName string) (*reflectInsertField, error) {
	v := reflect.ValueOf(i)
	k := v.Kind()
	switch k {
	case reflect.Ptr:
	default:
		return nil, errors.New("Unsupport reflect type").As(k.String())
	}
	v = reflect.Indirect(v)

	tm := refxM.TypeMap(v.Type())

	names := []byte{}
	inputs := []byte{}
	vals := []interface{}{}
	var autoIncrement *reflect.Value

	childrenLen := len(tm.Tree.Children)
	order := 0
	for i := 0; i < childrenLen; i++ {
		field := tm.Tree.Children[i]
		if field == nil {
			// found ignore tag, do next.
			continue
		}

		fieldVal := v.Field(i)
		autoField := travelStructField(field, &fieldVal, &order, &drvName, &names, &inputs, &vals)
		if autoField != nil {
			autoIncrement = autoField
		}
	}

	if len(names) == 0 {
		panic("No public field in struct")
	}
	return &reflectInsertField{
		Names:         string(names[:len(names)-1]),
		Stmts:         string(inputs[:len(inputs)-1]),
		Values:        vals,
		AutoIncrement: autoIncrement,
	}, nil
}
