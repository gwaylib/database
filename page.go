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

type PageArgs struct {
	args   []interface{}
	offset int64
	limit  int64
}

func NewPageArgs(args ...interface{}) *PageArgs {
	return &PageArgs{
		args: args,
	}
}

// using offset and limit when limit is set.
func (p *PageArgs) Limit(offset, limit int64) *PageArgs {
	p.offset = offset
	p.limit = limit
	return p
}

type PageSql struct {
	countSql string
	dataSql  string
}

func NewPageSql(countSql, dataSql string) *PageSql {
	if len(countSql) == 0 {
		panic("countSql not set")
	}
	if len(dataSql) == 0 {
		panic("dataSql not set")
	}
	return &PageSql{
		countSql: countSql,
		dataSql:  dataSql,
	}
}

func (p PageSql) CountSql() string {
	return p.countSql
}
func (p PageSql) DataSql() string {
	return p.dataSql
}

// fill the page sql with fmt arg, and return a new page
// Typically used for table name formatting
func (p PageSql) FmtPage(args ...interface{}) PageSql {
	countSql := p.countSql
	if len(countSql) > 0 {
		countSql = fmt.Sprintf(p.countSql, args...)
	}
	dataSql := p.dataSql
	if len(dataSql) > 0 {
		dataSql = fmt.Sprintf(p.dataSql, args...)
	}

	return PageSql{
		countSql: countSql,
		dataSql:  dataSql,
	}
}

func (p *PageSql) QueryCount(db *DB, args ...interface{}) (int64, error) {
	count := int64(0)
	if err := QueryElem(db, &count, p.countSql, args...); err != nil {
		return 0, errors.As(err)
	}
	return count, nil
}

func (p *PageSql) QueryPageArr(db *DB, doCount bool, args *PageArgs) (int64, []string, [][]interface{}, error) {
	total := int64(0)
	dataArgs := args.args
	if args.limit > 0 {
		dataArgs = append(dataArgs, []interface{}{args.offset, args.limit}...)
	}
	titles, data, err := QueryPageArr(db, p.dataSql, dataArgs...)
	if err != nil {
		return total, nil, nil, errors.As(err)
	} else if doCount {
		count, err := p.QueryCount(db, args.args)
		if err != nil {
			return total, nil, nil, errors.As(err)
		}
		total = count
	}
	return total, titles, data, nil
}

func (p *PageSql) QueryPageMap(db *DB, doCount bool, args *PageArgs) (int64, []string, []map[string]interface{}, error) {
	total := int64(0)
	dataArgs := args.args
	if args.limit > 0 {
		dataArgs = append(dataArgs, []interface{}{args.offset, args.limit}...)
	}
	title, data, err := QueryPageMap(db, p.dataSql, dataArgs...)
	if err != nil {
		return total, nil, nil, errors.As(err)
	} else if doCount {
		count, err := p.QueryCount(db, args.args)
		if err != nil {
			return total, nil, nil, errors.As(err)
		}
		total = count
	}
	return total, title, data, nil
}
