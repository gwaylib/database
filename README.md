# Refere to:
```
database/sql
https://github.com/jmoiron/sqlx
```

# Examle:
More examle see the examle directory.

## Using etc cache
Assume that the configuration file path is: './etc/db.cfg'

The etc content
```
[master]
driver: mysql
dsn: username:passwd@tcp(127.0.0.1:3306)/center?timeout=30s&strict=true&loc=Local&parseTime=true&allowOldPasswords=1
life_time:7200

[log]
driver: mysql
dsn: username:passwd@tcp(127.0.0.1:3306)/log?timeout=30s&strict=true&loc=Local&parseTime=true&allowOldPasswords=1
life_time:7200
``

Make a cache
``` text
package db

import (
	"github.com/gwaylib/conf"
	"github.com/gwaylib/database"
	_ "github.com/go-sql-driver/mysql"
)

var dbFile = conf.RootDir() + "/etc/db.cfg"

func init() {
   database.REFLECT_DRV_NAME = database.DRV_NAME_MYSQL 
}

func GetCache(section string) *database.DB {
	return database.GetCache(dbFile, section)
}

func HasCache(section string) (*database.DB, error) {
	return database.HasCache(dbFile, section)
}

func CloseCache() {
	database.CloseCache()
}
```

Call a cache
``` text
mdb := db.GetCache("master")
```

## Standar query 
``` text
mdb := db.GetCache("master") 
// or mdb = <sql.Tx>

// row := mdb.QueryRow("SELECT * ...")
row := database.QueryRow(mdb, "SELECT * ...")
// ...

// rows, err := mdb.Query("SELECT * ...")
rows, err := database.Query(mdb, "SELECT * ...")
// ...

// result, err := mdb.Exec("UPDATE ...")
result, err := database.Exec(mdb, "UPDATE ...")
// ...
```

## Insert a struct to db(using reflect)
``` text
type User struct{
    Id     int64  `db:"id,auto_increment"` // flag "autoincrement", "auto_increment" will call "SetLastInsertId" method if you implement the database.AutoIncrAble interface.
    Name   string `db:"name"`
    Ignore string `db:"-"` // ignore flag: "-"
}

func (u *User)SetLastInsertId(id int64, err error){
    if err != nil{
        panic(err)
    }
    u.Id = id
}

var u = &User{
    Name:"testing",
}

// Insert data with default driver.
if _, err := database.InsertStruct(mdb, u, "testing"); err != nil{
    // ... 
}
// ...

// Or Insert data with designated driver.
if _, err := database.InsertStruct(mdb, u, "testing", database.DRV_NAME_MYSQL); err != nil{
    // ... 
}
// ...
```

## MultiTx
``` text
multiTx := []*database.MultiTx{}
multiTx = append(multiTx, database.NewMultiTx(
    "UPDATE testing SET name = ? WHERE id = ?",
    id,
))
multiTx = append(multiTx, database.NewMultiTx(
    "UPDATE testing SET name = ? WHERE id = ?",
    id,
))

// do exec multi tx
mdb := db.GetCache("master") 
tx, err := mdb.Begin()
if err != nil{
    // ...
}
if err := database.ExecMutlTx(tx, multiTx); err != nil {
    database.Rollback(tx)
    // ...
}
if err := tx.Commit(); err != nil {
    database.Rollback(tx)
    // ...
}
```

## Quick query way
``` text

// Way 1: query result to a struct.
type User struct{
    Id   int64 `db:"id"`
    Name string `db:"name"`
}

mdb := db.GetCache("master") 
// or mdb = <sql.Tx>
var u = *User{}
if err := database.QueryStruct(mdb, u, "SELECT id, name FROM a WHERE id = ?", id)
if err != nil{
    // ...
}
// ..

// Way 2: query row to struct
mdb := db.GetCache("master") 
// or mdb = <sql.Tx>
var u = *User{}
if err := database.ScanStruct(database.QueryRow(mdb, "SELECT id, name FROM a WHERE id = ?", id), u); err != nil {
    // ...
}

// Way 3: query result to structs
mdb := db.GetCache("master") 
// or mdb = <sql.Tx>
var u = []*User{}
if err := database.QueryStructs(mdb, &u, "SELECT id, name FROM a WHERE id = ?", id); err != nil {
    // ...
}
if len(u) == 0{
    // data not found
    // ...
}
// .. 

// Way 4: query rows to structs
mdb := db.GetCache("master") 
// or mdb = <sql.Tx>
rows, err := database.Query(mdb, "SELECT id, name FROM a WHERE id = ?", id)
if err != nil {
    // ...
}
defer database.Close(rows)
var u = []*User{}
if err := database.ScanStructs(rows, &u); err != nil{
    // ...
}
if len(u) == 0{
    // data not found
    // ...
}

```

## Query an element which implementd sql.Scanner

```text
mdb := db.GetCache("master") 
// or mdb = <sql.Tx>
count := 0
if err := database.QueryElem(mdb, &count, "SELECT count(*) FROM a WHERE id = ?", id); err != nil{
    // ...
}
```

## Mass query.
```text
mdb := db.GetCache("master") 

var (
	userInfoQsql = &qsql.Template{
		CountSql: `
SELECT 
    count(1) 
FROM 
    %s
WHERE
    mobile = ?
`,
		DataSql: `
SELECT 
    mobile "phone"
FROM 
    %s
WHERE
    mobile = ?
ORDER BY
    mobile
LIMIT ?, ?
`,
	}
)

// Count the rows
count := 0
if err := database.QueryElem(
    mdb,
    &count, 
    userInfoQsql.Sprintf("user_info_200601").CountSql,
    "13800138000",
); err != nil{
    // ...
}

// Query the result to a string table
title, result, err := database.QueryTable(
    mdb,
    userInfoQsql.Sprintf("user_info_200601").DataSql,
    "13800138000", currPage*10, 10)
if err != nil {
    // ...
}

// Query the result to a string map
result, err := database.QueryMap(
    mdb,
    userInfoQsql.Sprintf("user_info_200601").DataSql,
    "13800130000",
    currPage*10, 10) 
if err != nil {
    // ...
}
```
