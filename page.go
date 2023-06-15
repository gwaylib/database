//
// Example:
//
// qSql = &database.Page{
//      CountSql:`SELECT count(1) FROM user_info WHERE create_time >= ? AND create_time <= ?`,
//      DataSql:`SELECT mobile, balance FROM user_info WHERE create_time >= ? AND create_time <= ?`
// }
// count, err := qSql.QueryCount(
// if err != nil{
//     return errors.As(err)
// }
// if count==0 {
//     return errors.ErrNoData
// }
// titles, result, err := qSql.QueryArray(db, condition, 0, 10)
// ...
// Or
// count, titles, result, err := qSql.QueryMap(db, condtion, 0, 10)
// ...

//
package database

import (
	"fmt"

	"github.com/gwaylib/errors"
)

type Page struct {
	CountSql string
	DataSql  string
}

// fill the page sql with fmt arg, and return a new page
// Typically used for table name formatting
func (p Page) FmtPage(args ...interface{}) *Page {
	countSql := p.CountSql
	if len(countSql) > 0 {
		countSql = fmt.Sprintf(p.CountSql, args...)
	}
	dataSql := p.DataSql
	if len(dataSql) > 0 {
		dataSql = fmt.Sprintf(p.DataSql, args...)
	}

	return &Page{
		CountSql: countSql,
		DataSql:  dataSql,
	}
}

func (p *Page) QueryCount(db *DB, args ...interface{}) (int64, error) {
	count := int64(0)
	if err := QueryElem(db, &count, p.CountSql, args...); err != nil {
		return 0, errors.As(err)
	}
	return count, nil
}

func (p *Page) QueryMatrixArr(db *DB, args ...interface{}) ([]string, [][]interface{}, error) {
	titles, data, err := QueryMatrixArr(db, p.DataSql, args...)
	if err != nil {
		return nil, nil, errors.As(err)
	}
	return titles, data, nil
}

func (p *Page) QueryMatrixMap(db *DB, args ...interface{}) ([]string, []map[string]interface{}, error) {
	title, data, err := QueryMatrixMap(db, p.DataSql, args...)
	if err != nil {
		return nil, nil, errors.As(err)
	}
	return title, data, nil
}
