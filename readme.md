
# 结构体转GORM where 条件

## 字段where条件筛选

字段之间为and关系

```go
type UserWhere struct{
    Name string
    Age int
}

u := UserWhere{
    Name: "foo",
    Age: 12,
}
var user []User

//SELECT * FROM `user` WHERE name = "foo" AND age = 12
DB.Scopes(structquery.Where(u)).Find(&user)


```

组合的匿名字段为or关系

```go

type UserOr struct {
	Name string
	Age  int
}

type UserWhere struct {
	UserOr
}

u := UserWhere{
	UserOr: UserOr{
		Name: "foo",
		Age:  12,
	},
}
var user []User

//SELECT * FROM `user` WHERE (name = "foo" OR age = 12)
DB.Scopes(structquery.Where(u)).Find(&user)

```

多种组合

```go
type UserAnd struct {
	Father  string
	Mother string
}
type UserOr struct {
	Name     string
	Age      int
	AndField UserAnd
}
type UserWhere struct {
	Addr string
	UserOr
}
u := UserWhere{
	UserOr: UserOr{
		Name: "foo",
		Age:  12,
		AndField: UserAnd{
			Father:  "foo",
			Mother: "bar",
		},
	},
	Addr: "bar",
}

var user []User

//SELECT * FROM `user` WHERE addr = "bar" AND (name = "foo" OR age = 12 OR (father = "foo" AND Mother = "bar"))
DB.Scopes(structquery.Where(u)).Find(&user)
```

## tag

1. op

op 表示操作符 不写默认等号 = 可选值：=,!=,>,<,>=,<=,like,in,not in,between,not between,null 或者含问号符号的

```go
type UserWhere struct {
	Name string
	Age  int `op:">"`
}
u := UserWhere{
	Name: "foo",
	Age:  12,
}

var user []User

//SELECT * FROM `user` WHERE name = \"foo\" AND age > 12
DB.Scopes(structquery.Where(u)).Find(&user)
type UserWhere2 struct {
	Name string `op:"like"`
}
u2 := UserWhere2{
	Name: "tom", //like 若不包含%则会在值开头结尾加上%
}

//SELECT * FROM `user` WHERE name like "%tom%"
DB.Scopes(structquery.Where(u2)).Find(&user)
u3 := UserWhere2{
	Name: "tom%",
}

//SELECT * FROM `user` WHERE name like "tom%"
DB.Scopes(structquery.Where(u3)).Find(&user)
type UesrWhere5 struct {
	TypeID []int32 `op:"type_id in ?"`
}

u6 := UesrWhere5{
	TypeID: []int32{1, 2, 3},
}

//SELECT * FROM `user` WHERE type_id in (1,2,3)
DB.Scopes(structquery.Where(u6)).Find(&user)
```

当字段为切片时，op默认为 in 可选值：in,not in

```go
type UserWhere3 struct {
	ID []int
}
u4 := UserWhere3{
	ID: []int{1, 2, 3},
}

var user []User

//SELECT * FROM `user` WHERE id in (1,2,3)
DB.Scopes(structquery.Where(u4)).Find(&user)

type UserWhere4 struct {
	ID []int `op:"not in"`
}

u5 := UserWhere4{
	ID: []int{1, 2, 3},
}

//SELECT * FROM `user` WHERE id not in (1,2,3)
DB.Scopes(structquery.Where(u5)).Find(&user)
```

当字段为时间切片时，op默认为 between 可选值：between,not between

```go
type UserWhere6 struct {
	Birth []time.Time
}
u7 := UserWhere6{
	Birth: []time.Time{
		time.Date(2001, 01, 02, 0, 0, 0, 0, time.Local),
		time.Date(2002, 01, 02, 0, 0, 0, 0, time.Local),
	},
}

var user []User
//SELECT * FROM `user` WHERE birth between "2001-01-02 00:00:00" and "2002-01-02 00:00:00"
DB.Scopes(structquery.Where(u7)).Find(&user)
```

当值为0值或空值或者op tag 值为"-"会跳过

```go
type UserWhere7 struct {
	Name   string
	Age    int `op:"-"`
	Father *string
	Status *int
}
var i int = 0
u8 := UserWhere7{
	Name:   "",
	Age:    12,
	Father: nil,
	Status: &i
}

var user []User
//SELECT * FROM `user where status = 0
DB.Scopes(structquery.Where(u8)).Find(&user)

```

当需要查询零值时，可以将字段类型改成对应的指针类型，当需要查询是否为空时，可以将op tag设置为null,字段类型为bool或*bool类型：
```go
type UserWhere8 struct {
	DeletedAt bool `op:"null"` //为true时查询 IS NULL 为false时跳过条件
	Status    *int
	CreatedAt *bool `op:"null"` //为true时查询 IS NULL，为false时查询 IS NOT NULL
}
var status int
var created bool
u9 := UserWhere8{
	DeletedAt: true,
	Status:    &status,
	CreatedAt: &created,
}

var user []User
//SELECT * FROM `user` WHERE deleted_at IS NULL AND status = 0 AND created_at IS NOT NULL
DB.Scopes(structquery.Where(u9)).Find(&user)
```



2. field

field 字段名称，可配置多个字段名称，字段名称之间用&分割时，字段名称之间的关系为and 用|分割时，表示or

```go
type UserWhere1 struct {
	Name string `op:"like" field:"name|father&Mother"`

}
u1 := UserWhere1{
	Name: "tommy",
}

var user []User
//SELECT * FROM `user` WHER1E name LIKE "%tommy%" OR father LIKE "%tommy%" AND Mother LIKE "%tommy%"
DB.Scopes(structquery.Where(u1)).Find(&user)
```

## 特殊字段

当字段名为page或则size时，表示分页属性

```go
type UserWhere1 struct {
	Page int
	Size int
}
u1 := UserWhere1{
	Page: 2,
	Size: 10,
}
var user []User


//SELECT * FROM `user` LIMIT 10 OFFSET 10
DB.Scopes(structquery.Where(u1)).Find(&user)
```

可以用以下方法来获取总行数

```go
type UserWhere1 struct {
	Page int
	Size int
	Name string `op:"like" field:"name"`
}
u1 := UserWhere1{
	Page: 1,
	Size: 10,
	Name: "To",
}
//添加了Where函数中添加count字段即可
//SELECT count(*) FROM `user` WHERE name LIKE "%To%"
//SELECT * FROM `user` WHERE name LIKE "%To%" LIMIT 10
DB.Scopes(structquery.Where(u1, &count)).Find(&user)
```

