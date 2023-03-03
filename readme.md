
# 结构体转GORM where 条件

字段where条件筛选

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

## tag 解释:

op 表示操作符 默认等号 = 可选值：=,!=,>,<,>=,<=,like,in,not in,between,not between 或者含问号符号的

当字段为数组时，op可选值：in,not in

当字段为时间数组时，op可选值：between,not between

当值为0值或空值会跳过

field 字段名称，可配置多个字段名称，字段名称之间用&分割时，字段名称之间的关系为and

用|分割时，表示or

当字段名为page或则size时，表示分页属性
