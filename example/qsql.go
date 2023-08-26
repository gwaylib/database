package main

import (
	"fmt"

	"github.com/gwaylib/database"
	"github.com/gwaylib/errors"
	_ "github.com/mattn/go-sqlite3"
)

type TestingUser struct {
	ID       int64  `db:"id,auto_increment"` // auto_increment or autoincrement
	UserName string `db:"username"`
	Passwd   string `db:"passwd"`
}

func main() {
	mdb, _ := database.Open("sqlite3", ":memory:")
	defer database.Close(mdb)

	// create table
	if _, err := database.Exec(mdb,
		`CREATE TABLE user (
		  "id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
		  "created_at" datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
		  "username" VARCHAR(32) NOT NULL UNIQUE,
		  "passwd" VARCHAR(128) NOT NULL
		);`); err != nil {
		panic(err)
	}

	// std insert
	if _, err := database.Exec(mdb, "INSERT INTO user(username,passwd)VALUES(?,?)", "t1", "t1"); err != nil {
		panic(err)
	}

	// reflect insert
	newUser := &TestingUser{UserName: "t2", Passwd: "t2"}
	if _, err := database.InsertStruct(mdb, newUser, "user"); err != nil {
		panic(err)
	}
	if newUser.ID == 0 {
		panic("expect newUser.ID > 0")
	}

	// std query
	var id int64
	var username, passwd string
	if err := database.QueryRow(mdb, "SELECT id, username, passwd FROM user WHERE username=?", "t1").Scan(&id, &username, &passwd); err != nil {
		panic(err)
	}
	if username != "t1" && passwd != "t1" {
		panic(username + "," + passwd)
	}
	if id == 0 {
		panic(id)
	}

	// reflect query
	// query struct data
	expectUser := &TestingUser{}
	if err := database.QueryStruct(mdb, expectUser, "SELECT * FROM user WHERE username=?", "t1"); err != nil {
		panic(err)
	}
	if expectUser.UserName != "t1" && expectUser.Passwd != "t1" {
		panic("data not match")
	}
	users := []*TestingUser{}
	if err := database.QueryStructs(mdb, &users, "SELECT * FROM user LIMIT 2"); err != nil {
		panic(err)
	}
	if len(users) != 2 {
		panic("expect len==2")
	}
	// query elememt data
	pwd := ""
	if err := database.QueryElem(mdb, &pwd, "SELECT passwd FROM user WHERE username=?", "t1"); err != nil {
		panic(err)
	}
	if pwd != "t1" {
		panic(pwd)
	}
	ids := []int64{}
	if err := database.QueryElems(mdb, &ids, "SELECT id FROM user LIMIT 2"); err != nil {
		panic(err)
	}
	if len(ids) != 2 {
		panic("expect len==2")
	}
	fmt.Printf("ids:%+v\n", ids)

	// query data in string
	// table type
	titles, data, err := database.QueryPageArr(mdb, "SELECT * FROM user LIMIT 10")
	if err != nil {
		panic(err)
	}
	fmt.Printf("PageArr title:%+v\n", titles)
	fmt.Printf("PageArr data: %+v\n", data)
	// map type
	titles, mData, err := database.QueryPageMap(mdb, "SELECT * FROM user LIMIT 10")
	if err != nil {
		panic(err)
	}
	fmt.Printf("PageMap title:%+v\n", titles)
	fmt.Printf("PageMap data: %+v\n", mData)

	// executer for tx
	tx, err := mdb.Begin()
	if err != nil {
		panic(err)
	}
	txUsers := []TestingUser{
		{UserName: "t3", Passwd: "t3"},
		{UserName: "t4", Passwd: "t4"},
	}
	for _, u := range txUsers {
		if _, err := database.InsertStruct(tx, &u, "user"); err != nil {
			println(errors.As(err))
			database.Rollback(tx)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		println(errors.As(err))
		database.Rollback(tx)
		return
	}

	// excute for stmt
	stmt, err := mdb.Prepare("SELECT COUNT(*) FROM user WHERE username=?")
	count := 0
	if err := stmt.QueryRow("t3").Scan(&count); err != nil {
		panic(err)
	}
	if count != 1 {
		panic(errors.New("need count==1").As(count))
	}
}
