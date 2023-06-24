//
// Example:
//
// qSql = &database.Page{
//      CountSql:`SELECT count(1) FROM user_info WHERE create_time >= ? AND create_time <= ?`,
//      DataSql:`SELECT mobile, balance FROM user_info WHERE create_time >= ? AND create_time <= ?`
// }
// count, titles, result, err := qSql.QueryPageArray(db, true, condition, 0, 10)
// ...
// Or
// count, titles, result, err := qSql.QueryPageMap(db, true, condtion, 0, 10)
// ...
// if err != nil {
//	   if !errors.ErrNoData.Equal(err) {
//         return errors.As(err)
//     }
//     // no data
// }
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

func (p *Page) QueryPageArr(db *DB, doCount bool, offset, limit int64, countArgs ...interface{}) (int64, []string, [][]interface{}, error) {
	total := int64(0)
	dataArgs := append(countArgs, []interface{}{offset, limit}...)
	titles, data, err := QueryPageArr(db, p.DataSql, dataArgs...)
	if err != nil {
		return total, nil, nil, errors.As(err)
	} else if doCount {
		count, err := p.QueryCount(db, countArgs)
		if err != nil {
			return total, nil, nil, errors.As(err)
		}
		total = count
	}
	return total, titles, data, nil
}

func (p *Page) QueryPageMap(db *DB, doCount bool, offset, limit int64, countArgs ...interface{}) (int64, []string, []map[string]interface{}, error) {
	total := int64(0)
	dataArgs := append(countArgs, []interface{}{offset, limit}...)
	title, data, err := QueryPageMap(db, p.DataSql, dataArgs...)
	if err != nil {
		return total, nil, nil, errors.As(err)
	} else if doCount {
		count, err := p.QueryCount(db, countArgs)
		if err != nil {
			return total, nil, nil, errors.As(err)
		}
		total = count
	}
	return total, title, data, nil
}
