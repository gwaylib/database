package main

import (
	"github.com/gwaylib/database"
	_ "github.com/mattn/go-sqlite3"
)

type TestingUser struct {
	ID       int64  `db:"id,auto_increment"`
	UserName string `db:"username"`
	Passwd   string `db:"passwd"`
}

// implement LastInsertId callback
func (u *TestingUser) SetLastInsertId(id int64, err error) {
	if err != nil {
		panic(err)
	}
	u.ID = id
}

func main() {
	mdb, _ := database.Open("sqlite3", ":memory:")

	// create table
	if _, err := database.Exec(mdb, `
CREATE TABLE user (
  id INTEGER AUTOINCREMENT,
  username VARCHAR(32) NOT NULL UNIQUE,
  passwd VARCHAR(128) NOT NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY(id)
);`); err != nil {
		panic(err)
	}

	// insert data
	if _, err := database.Exec(mdb, "INSERT INTO user(username,passwd)VALUES(?,?)", "t1", "t1"); err != nil {
		panic(err)
	}
	newUser := &TestingUser{UserName: "t2", Passwd: "t2"}
	if _, err := database.InsertStruct(mdb, "user", newUser); err != nil {
		panic(err)
	}
	if newUser.ID == 0 {
		panic("expect newUser.ID > 0")
	}

	// query data
	expectUser := &TestingUser{}
	if err := database.QueryStruct(mdb, expectUser, "SELECT * FROM user WHERE id WHERE username=?", "t1"); err != nil {
		panic(err)
	}
	if expectUser.UserName != "t1" && expectUser.Passwd != "t1" {
		panic("data not match")
	}
	users := []*TestingUser{}
	if err := database.QueryStruct(mdb, &users, "SELECT * FROM user LIMIT 2"); err != nil {
		panic(err)
	}
	if len(users) != 2 {
		panic("expect len==2")
	}

	pwds := []string{}
	if err := database.QueryStruct(mdb, &users, "SELECT passwd FROM user LIMIT 2"); err != nil {
		panic(err)
	}
	if len(pwds) != 2 {
		panic("expect len==2")
	}
}
