# 说明

我很喜欢使用标准库对标准SQL的调用，因此本项目参考了其他库仅实现补充了标准库的一些不方便的地方，比如通用的新增与查询功能、配置文件使用等，以便提调编码效率，参考资料：

标准库

    database/sql

sqlx框架

    https://github.com/jmoiron/sqlx


# 使用例子：

## Cache使用
### 配置文件

配置文件(假定为:"/etc/db.cfg")中配置如下格式:

``` text
# 主库
[master]
driver: mysql
dsn: username:passwd@tcp(127.0.0.1:3306)/center?timeout=30s&strict=true&loc=Local&parseTime=true&allowOldPasswords=1
life_time:7200 # 选填，单位秒，未配置时默认值为系统默认值；用于设置sql.SetConnMaxLifetime(time.Duration)值

# 日志库
[log]
driver: mysql
dsn: username:passwd@tcp(127.0.0.1:3306)/log?timeout=30s&strict=true&loc=Local&parseTime=true&allowOldPasswords=1
life_time:7200 # 选填，单位秒，未配置时默认值为系统默认值；用于设置sql.SetConnMaxLifetime(time.Duration)值
```

### 重写Cache接口

``` text
package db

// 导入驱动库
import (
	"github.com/gwaylib/conf"
	"github.com/gwaylib/database"
	_ "github.com/go-sql-driver/mysql"
)

var dbFile = conf.RootDir() + "/etc/db.cfg"

func init() {
   database.REFLECT_DRV_NAME = database.DRV_NAME_MYSQL // 修改反射工具的默认数据库驱动类型, 不执行此操作时，默认是MYSQL
}

func GetCache(section string) *database.DB {
	return database.GetCache(dbFile, section)
}

func HasCache(section string) (*database.DB, error) {
	return database.HasCache(dbFile, section)
}

// 当使用了Cache，在程序退出时可调用database.CloseCache进行正常关闭数据库连接
func CloseCache() {
	database.CloseCache()
}

```

### Cache调用
``` text
mdb := db.GetCache("master")
```



## 性能级别建议使用标准库以便可灵活运用

### 执行标准查询
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

### 快速新增数据
``` text
// 定义表结构体
type User struct{
    Id     int64  `db:"id,auto_increment"` // 带有autoincrement、auto_increment等标签的字段在插入时会触发SetLastInsertId接口
    Name   string `db:"name"`
    Ignore string `db:"-"` // 带"-"标签时，该字段将不会被执行
}

// 实现自增回调接口
// AutoIncrAble接口应配合auto_increment标签使用
func (u *User)SetLastInsertId(id int64, err error){
    if err != nil{
        panic(err)
    }
    u.Id = id
}

var u = &User{
    Name:"testing",
}

// 新增例子一：
// 若mdb是非本接口实现的DB时，需要设置默认驱动名
// database.DEFAULT_DRV_NAME = database.DRV_NAME_MYSQL
if _, err := database.InsertStruct(mdb, u, "testing"); err != nil{
    // ... 
}
// ...

// 新增例子二(使用指定的反射数据库驱动)：
if _, err := database.InsertStruct(mdb, u, "testing", database.DRV_NAME_MYSQL); err != nil{
    // ... 
}
// ...
```

### 批量操纵数据
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

## 快速查询, 用于通用性的查询，例如js页面返回
### 查询结果到结构体
``` text

// 定义表结构体
type User struct{
    Id   int64 `db:"id"`
    Name string `db:"name"`
}

// 方法一
mdb := db.GetCache("master") 
// or mdb = <sql.Tx>
var u = *User{}
if err := database.QueryStruct(mdb, u, "SELECT id, name FROM a WHERE id = ?", id)
if err != nil{
    // ...
}
// ..

// 或者
// 方法二
mdb := db.GetCache("master") 
// or mdb = <sql.Tx>
var u = *User{}
if err := database.ScanStruct(database.QueryRow(mdb, "SELECT id, name FROM a WHERE id = ?", id), u); err != nil {
    // ...
}

// 或者
// 方法三
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

// 或者
// 方法四
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

### 查询单个元素结果
```text
mdb := db.GetCache("master") 
// or mdb = <sql.Tx>
count := 0
if err := database.QueryElem(mdb, &count, "SELECT count(*) FROM a WHERE id = ?", id); err != nil{
    // ...
}
```

### 批量查询
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
    mobile "手机号"
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

// 查询总数量
count := 0
if err := database.QueryElem(
    mdb,
    &count, 
    userInfoQsql.Sprintf("user_info_200601").CountSql,
    "13800138000",
); err != nil{
    // ...
}

// 表格方式查询结果集
result, err := database.QueryTable(
    mdb,
    userInfoQsql.Sprintf("user_info_200601").DataSql,
    "13800138000", currPage*10, 10)
if err != nil {
    // ...
}

// 或者对象方式查询结果集
result, err := database.QueryMap(
    mdb,
    userInfoQsql.Sprintf("user_info_200601").DataSql,
    "13800130000",
    currPage*10, 10) 
if err != nil {
    // ...
}
```
