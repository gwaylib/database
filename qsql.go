package database

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/gwaylib/errors"
	"github.com/jmoiron/sqlx/reflectx"
)

type MultiTx struct {
	Query string
	Args  []interface{}
}

func NewMultiTx(query string, args ...interface{}) *MultiTx {
	return &MultiTx{query, args}
}

const (
	addObjSql = "INSERT INTO %s (%s) VALUES (%s);"
)

// field flag like: `db:"name"`
// more: github.com/jmoiron/sqlx
func insertStruct(exec Execer, ctx context.Context, obj interface{}, tbName string, drvNames ...string) (sql.Result, error) {
	drvName := REFLECT_DRV_NAME
	db, ok := exec.(*DB)
	if ok {
		drvName = db.DriverName()
	} else {
		drvNamesLen := len(drvNames)
		if drvNamesLen > 0 {
			if drvNamesLen != 0 {
				panic(errors.New("'drvNames' expect only one argument").As(drvNames))
			}
			drvName = drvNames[0]
		}
	}

	fields, err := reflectInsertStruct(obj, drvName)
	if err != nil {
		return nil, errors.As(err)
	}
	execSql := fmt.Sprintf(addObjSql, tbName, fields.Names, fields.Stmts)
	// log.Debugf("%s%+v", execSql, vals)
	result, err := exec.ExecContext(ctx, execSql, fields.Values...)
	if err != nil {
		return nil, errors.As(err, execSql)
	}
	if fields.AutoIncrement != nil {
		id, _ := result.LastInsertId()
		var val reflect.Value
		kind := fields.AutoIncrement.Kind()
		switch kind {
		case reflect.Int:
			val = reflect.ValueOf(int(id))
		case reflect.Int8:
			val = reflect.ValueOf(int8(id))
		case reflect.Int16:
			val = reflect.ValueOf(int16(id))
		case reflect.Int32:
			val = reflect.ValueOf(int32(id))
		case reflect.Int64:
			val = reflect.ValueOf(int64(id))
		case reflect.Uint: // Warnning: this maybe out of int64
			val = reflect.ValueOf(uint(id))
		case reflect.Uint8:
			val = reflect.ValueOf(uint8(id))
		case reflect.Uint16:
			val = reflect.ValueOf(uint16(id))
		case reflect.Uint32:
			val = reflect.ValueOf(uint32(id))
		case reflect.Uint64: // Warnning: this maybe out of int64
			val = reflect.ValueOf(uint64(id))
		default:
			// unsupport other kind here
			panic("unsupport auto increment kind: " + kind.String())
		}

		fields.AutoIncrement.Set(val)
	}
	return result, nil
}

func execMultiTx(tx *sql.Tx, ctx context.Context, mTx []*MultiTx) error {
	for _, mt := range mTx {
		if _, err := tx.ExecContext(ctx, mt.Query, mt.Args...); err != nil {
			return errors.As(err)
		}
	}
	return nil
}

// fieldsByName fills a values interface with fields from the passed value based
// on the traversals in int.  If ptrs is true, return addresses instead of values.
// We write this instead of using FieldsByName to save allocations and map lookups
// when iterating over many rows.  Empty traversals will get an interface pointer.
// Because of the necessity of requesting ptrs or values, it's considered a bit too
// specialized for inclusion in reflectx itself.
func fieldsByTraversal(v reflect.Value, traversals [][]int, values []interface{}, ptrs bool) error {
	for i, traversal := range traversals {
		if len(traversal) == 0 {
			values[i] = new(interface{})
			continue
		}
		f := reflectx.FieldByIndexes(v, traversal)
		if ptrs {
			values[i] = f.Addr().Interface()
		} else {
			values[i] = f.Interface()
		}
	}
	return nil
}

func scanStruct(rows Rows, obj interface{}) error {
	if obj == nil {
		return errors.New("nil pointer passed to StructScan destination")
	}

	value := reflect.ValueOf(obj)
	if value.Kind() != reflect.Ptr {
		return errors.New("must pass a pointer, not a value, to StructScan destination")
	}

	base := reflectx.Deref(value.Type())
	if base.Kind() != reflect.Struct {
		return errors.As(fmt.Errorf("expected struct pointer but got %s", value.Kind()))
	}

	columns, err := rows.Columns()
	if err != nil {
		return errors.As(err)
	}

	fields := refxM.TraversalsByName(base, columns)
	values := make([]interface{}, len(columns))

	direct := reflect.Indirect(value)

	vp := reflect.New(base)
	v := reflect.Indirect(vp)
	if err := fieldsByTraversal(v, fields, values, true); err != nil {
		return errors.As(err)
	}
	if !rows.Next() {
		return sql.ErrNoRows
	}
	if err := rows.Scan(values...); err != nil {
		return errors.As(err)
	}
	direct.Set(v)
	return nil
}
func scanStructs(rows Rows, obj interface{}) error {
	if obj == nil {
		return errors.New("nil pointer passed to StructScan destination")
	}
	value := reflect.ValueOf(obj)
	if value.Kind() != reflect.Ptr {
		return errors.New("must pass a pointer, not a value, to StructScan destination")
	}
	slice := reflectx.Deref(value.Type())
	if slice.Kind() != reflect.Slice {
		return errors.As(fmt.Errorf("expected slice but got %s", value.Kind()))
	}
	base := reflectx.Deref(slice.Elem())

	columns, err := rows.Columns()
	if err != nil {
		return errors.As(err)
	}
	fields := refxM.TraversalsByName(base, columns)
	direct := reflect.Indirect(value)
	isPtr := slice.Elem().Kind() == reflect.Ptr
	values := make([]interface{}, len(columns))
	var v, vp reflect.Value
	for rows.Next() {
		vp = reflect.New(base)
		v = reflect.Indirect(vp)
		if err := fieldsByTraversal(v, fields, values, true); err != nil {
			return errors.As(err)
		}

		if err := rows.Scan(values...); err != nil {
			return errors.As(err)
		}
		if isPtr {
			direct.Set(reflect.Append(direct, vp))
		} else {
			direct.Set(reflect.Append(direct, v))
		}
	}

	return nil
}

func queryStruct(db Queryer, ctx context.Context, obj interface{}, querySql string, args ...interface{}) error {
	rows, err := db.QueryContext(ctx, querySql, args...)
	if err != nil {
		return errors.As(err, args)
	}
	defer Close(rows)

	if err := scanStruct(rows, obj); err != nil {
		return errors.As(err, args)
	}
	return nil
}

func queryStructs(db Queryer, ctx context.Context, obj interface{}, querySql string, args ...interface{}) error {
	rows, err := db.QueryContext(ctx, querySql, args...)
	if err != nil {
		return errors.As(err, args)
	}
	defer Close(rows)

	if err := scanStructs(rows, obj); err != nil {
		return errors.As(err, args)
	}

	return nil
}

func queryElem(db Queryer, ctx context.Context, result interface{}, querySql string, args ...interface{}) error {
	if err := db.QueryRowContext(ctx, querySql, args...).Scan(result); err != nil {
		if sql.ErrNoRows != err {
			return errors.As(err, querySql, args)
		}
		return err
	}
	return nil
}

func queryElems(db Queryer, ctx context.Context, arr interface{}, querySql string, args ...interface{}) error {
	if arr == nil {
		return errors.New("nil pointer passed to StructScan destination")
	}
	value := reflect.ValueOf(arr)
	if value.Kind() != reflect.Ptr {
		return errors.New("must pass a pointer, not a value, to StructScan destination")
	}
	slice := reflectx.Deref(value.Type())
	if slice.Kind() != reflect.Slice {
		return errors.As(fmt.Errorf("expected slice but got %s", value.Kind()))
	}
	base := reflectx.Deref(slice.Elem())

	rows, err := db.QueryContext(ctx, querySql, args...)
	if err != nil {
		return errors.As(err, querySql, args)
	}
	defer Close(rows)

	isPtr := slice.Elem().Kind() == reflect.Ptr
	direct := reflect.Indirect(value)
	var vp reflect.Value
	for rows.Next() {
		vp = reflect.New(base)
		if err := rows.Scan(vp.Interface()); err != nil {
			return errors.As(err)
		}
		if isPtr {
			direct.Set(reflect.Append(direct, vp))
		} else {
			direct.Set(reflect.Append(direct, reflect.Indirect(vp)))
		}
	}
	return nil
}

// 执行一个通用的查询
// 因需要查标题，相对标准sql会慢一些，适用于偷懒查询的方式
// 即使发生错误返回至少是零长度的值
func queryPageArr(db Queryer, ctx context.Context, querySql string, args ...interface{}) (titles []string, result [][]interface{}, err error) {
	titles = []string{}
	result = [][]interface{}{}
	rows, err := db.QueryContext(ctx, querySql, args...)
	if err != nil {
		return titles, result, errors.As(err, args)
	}
	defer Close(rows)

	titles, err = rows.Columns()
	if err != nil {
		return titles, result, errors.As(err, args)
	}

	for rows.Next() {
		r := makeDBData(len(titles))
		if err := rows.Scan(r...); err != nil {
			return titles, result, errors.As(err, args)
		}
		result = append(result, r)
	}

	return titles, result, nil
}

// 查询一条数据，并发map结构返回，以便页面可以直接调用
// 因需要查标题，相对标准sql会慢一些，适用于偷懒查询的方式
// 即使发生错误返回至少是零长度的值
func queryPageMap(db Queryer, ctx context.Context, querySql string, args ...interface{}) ([]string, []map[string]interface{}, error) {
	rows, err := db.QueryContext(ctx, querySql, args...)
	if err != nil {
		return nil, []map[string]interface{}{}, errors.As(err, args)
	}
	defer Close(rows)

	titles, err := rows.Columns()
	if err != nil {
		return titles, []map[string]interface{}{}, errors.As(err, args)
	}

	result := []map[string]interface{}{}
	for rows.Next() {
		r := makeDBData(len(titles))
		if err := rows.Scan(r...); err != nil {
			return titles, []map[string]interface{}{}, errors.As(err, args)
		}
		mData := map[string]interface{}{}
		for i, name := range titles {
			_, ok := mData[name]
			if ok {
				return titles, result, errors.New("Already exist column name").As(name)
			}
			mData[name] = r[i]
		}
		result = append(result, mData)
	}
	return titles, result, nil
}
