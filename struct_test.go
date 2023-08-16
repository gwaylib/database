package database

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"time"
)

// for id insert
type ReflectTestStruct1 struct {
	Id         int64          `db:"id,auto_increment"`
	A          int            `db:"a"`
	B          int            `db:"-"`
	T          time.Time      `db:"time"`
	Slice      []byte         `db:"data"`
	Byte       byte           `db:"byte"`
	DBStr      DBData         `db:"dbdata"`
	NullString sql.NullString `db:"null_string"`
	C          string
}

// for autoincrement
type ReflectTestStruct2 struct {
	Id int64 `db:"id"`
	A  int   `db:"a"`
	B  int   `db:"-"`
	C  string
}

type ReflectTestStruct3 struct {
	ReflectTestStruct1
	D string `db:"d"`
}
type ReflectTestStruct4 struct {
	ReflectTestStruct3
	*ReflectTestStruct2
	E string `db:"e"`
}

func TestReflect(t *testing.T) {
	s1 := &ReflectTestStruct1{
		A:     100,
		B:     200,
		C:     "testing",
		Slice: []byte("abc"),
	}
	refVal, err := reflectInsertStruct(s1, "mysql")
	if err != nil {
		t.Fatal(err)
	}
	if refVal.Names != "`a`,`time`,`data`,`byte`,`dbdata`,`null_string`,`C`" {
		t.Fatal(refVal.Names)
	}
	if refVal.Stmts != "?,?,?,?,?,?,?" {
		t.Fatal(refVal.Stmts)
	}
	if len(refVal.Values) != 7 {
		t.Fatalf("%+v\n", refVal.Values)
	}
	refVal.SetAutoIncrement(reflect.ValueOf(int64(2)))
	if refVal.AutoIncrement.Int() != 2 {
		t.Fatalf("expect 2, but:%v", refVal.AutoIncrement.Int())
	}

	s2 := &ReflectTestStruct2{
		Id: 1,
		A:  101,
		B:  201,
		C:  "testing1",
	}
	refVal, err = reflectInsertStruct(s2, "mysql")
	if err != nil {
		t.Fatal(err)
	}
	if refVal.Names != "`id`,`a`,`C`" {
		t.Fatal(refVal.Names)
	}
	if refVal.Stmts != "?,?,?" {
		t.Fatal(refVal.Stmts)
	}
	if len(refVal.Values) != 3 {
		t.Fatalf("%+v\n", refVal.Values)
	}

	s4 := &ReflectTestStruct4{
		ReflectTestStruct2: s2,
		ReflectTestStruct3: ReflectTestStruct3{
			ReflectTestStruct1: *s1,
			D:                  "d",
		},
		E: "e",
	}
	refVal, err = reflectInsertStruct(s4, "oracle")
	if err != nil {
		t.Fatal(err)
	}
	if refVal.Names != `"a","time","data","byte","dbdata","null_string","C","d","id","a","C","e"` {
		t.Fatal(refVal.Names)
	}
	if refVal.Stmts != ":a,:time,:data,:byte,:dbdata,:null_string,:C,:d,:id,:a,:C,:e" {
		t.Fatal(refVal.Stmts)
	}
	if fmt.Sprintf("%+v", refVal.Values) != `[100 0001-01-01 00:00:00 +0000 UTC [97 98 99] 0  {String: Valid:false} testing d 1 101 testing1 e]` {
		t.Fatal(refVal.Values)
	}
}
