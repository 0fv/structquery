package structquery

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

func TestMain(m *testing.M) {
	var err error
	DB, err = gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{
		DryRun: false,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	DB.AutoMigrate(&User{})
	if err != nil {
		panic(err)
	}
	m.Run()
	os.Remove("gorm.db")
}

type User struct {
	ID     int `gorm:"primaryKey"`
	Name   string
	Age    int
	Father string
	Mother string
}

func TestAll(t *testing.T){
	TestAnd(t)
	TestOrType1(t)
	TestOrType2(t)
	TestOp(t)
	TestField(t)
	TestPageSize(t)
	TestOrder(t)
	TestCount(t)
}

func TestAnd(t *testing.T) {
	type UserWhere struct {
		Name string
		Age  int
	}
	u := UserWhere{
		Name: "foo",
		Age:  12,
	}
	assert.Equal(
		t,
		"SELECT * FROM `user` WHERE `name` = \"foo\" AND `age` = 12",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u)).Find(&User{})
		}), "should be equal",
	)
}

func TestOrType1(t *testing.T) {
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
	assert.Equal(
		t,
		"SELECT * FROM `user` WHERE (`name` = \"foo\" OR `age` = 12)",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u)).Find(&User{})
		}), "should be equal",
	)
}

func TestOrType2(t *testing.T) {
	type UserAnd struct {
		Father string
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
				Father: "foo",
				Mother: "bar",
			},
		},
		Addr: "bar",
	}
	assert.Equal(
		t,
		"SELECT * FROM `user` WHERE `addr` = \"bar\" AND (`name` = \"foo\" OR `age` = 12 OR (`father` = \"foo\" AND `mother` = \"bar\"))",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u)).Find(&User{})
		}), "should be equal",
	)
}

func TestOp(t *testing.T) {
	type UserWhere struct {
		Name string
		Age  int `op:">"`
	}
	u := UserWhere{
		Name: "foo",
		Age:  12,
	}
	assert.Equal(
		t,
		"SELECT * FROM `user` WHERE `name` = \"foo\" AND `age` > 12",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u)).Find(&User{})
		}), "should be equal",
	)
	type UserWhere2 struct {
		Name string `op:"like"`
	}

	u2 := UserWhere2{
		Name: "tom",
	}
	assert.Equal(
		t,
		"SELECT * FROM `user` WHERE `name` LIKE \"%tom%\"",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u2)).Find(&User{})
		}), "should be equal",
	)
	u3 := UserWhere2{
		Name: "tom%",
	}
	assert.Equal(
		t,
		"SELECT * FROM `user` WHERE `name` LIKE \"tom%\"",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u3)).Find(&User{})
		}), "should be equal",
	)
	type UserWhere3 struct {
		ID []int
	}
	u4 := UserWhere3{
		ID: []int{1, 2, 3},
	}
	assert.Equal(
		t,
		"SELECT * FROM `user` WHERE `id` IN (1,2,3)",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u4)).Find(&User{})
		}), "should be equal",
	)
	type UserWhere4 struct {
		ID []int `op:"not in"`
	}
	u5 := UserWhere4{
		ID: []int{1, 2, 3},
	}
	assert.Equal(
		t,
		"SELECT * FROM `user` WHERE `id` NOT IN (1,2,3)",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u5)).Find(&User{})
		}), "should be equal",
	)
	type UserWhere5 struct {
		TypeID []int32 `op:"type_id in ?"`
	}
	u6 := UserWhere5{
		TypeID: []int32{1, 2, 3},
	}
	assert.Equal(
		t,
		"SELECT * FROM `user` WHERE type_id in (1,2,3)",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u6)).Find(&User{})
		}), "should be equal",
	)
	type UserWhere6 struct {
		Birth []time.Time
	}
	u7 := UserWhere6{
		Birth: []time.Time{
			time.Date(2001, 01, 02, 0, 0, 0, 0, time.Local),
			time.Date(2002, 01, 02, 0, 0, 0, 0, time.Local),
		},
	}
	assert.Equal(
		t,
		"SELECT * FROM `user` WHERE `birth` BETWEEN \"2001-01-02 00:00:00\" AND \"2002-01-02 00:00:00\"",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u7)).Find(&User{})
		}), "should be equal",
	)
	type UserWhere7 struct {
		Name   string
		Age    int `op:"-"`
		Father *string
	}
	u8 := UserWhere7{
		Name:   "",
		Age:    12,
		Father: nil,
	}
	assert.Equal(
		t,
		"SELECT * FROM `user`",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u8)).Find(&User{})
		}), "should be equal",
	)

	type UserWhere8 struct {
		DeletedAt bool `op:"null"`
		Status    *int
		CreatedAt *bool `op:"null"`
		UpdatedAt bool  `op:"not null"`
	}
	var status int
	var created bool
	u9 := UserWhere8{
		DeletedAt: true,
		Status:    &status,
		CreatedAt: &created,
		UpdatedAt: true,
	}
	assert.Equal(
		t,
		"SELECT * FROM `user` WHERE `deleted_at` IS NULL AND `status` = 0 AND `created_at` IS NOT NULL AND `updated_at` IS NOT NULL",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u9)).Find(&User{})
		}), "should be equal",
	)
}

func TestField(t *testing.T) {
	type UserWhere1 struct {
		Name string `op:"like" field:"name|father&mother"`
	}
	u1 := UserWhere1{
		Name: "tommy",
	}
	assert.Equal(
		t,
		"SELECT * FROM `user` WHERE `name` LIKE \"%tommy%\" OR `father` LIKE \"%tommy%\" AND `mother` LIKE \"%tommy%\"",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u1)).Find(&User{})
		}), "should be equal",
	)
	type UserWhere2 struct {
		Name string `op:"like" field:"name&(father|mother)"`
	}
	u2 := UserWhere2{
		Name: "tommy",
	}
	assert.Equal(
		t,
		"SELECT * FROM `user` WHERE `name` LIKE \"%tommy%\" AND (`father` LIKE \"%tommy%\" OR `mother` LIKE \"%tommy%\")",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u2)).Find(&User{})
		}), "should be equal",
	)
}

func TestPageSize(t *testing.T) {
	type UserWhere1 struct {
		Page int `op:"page"`
		Size int `op:"size"`
	}
	u1 := UserWhere1{
		Page: 2,
		Size: 10,
	}
	assert.Equal(
		t,
		"SELECT * FROM `user` LIMIT 10 OFFSET 10",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u1)).Find(&User{})
		}), "should be equal",
	)
	type UserWhere2 struct {
		UserWhere1
	}
	u2 := UserWhere2{
		UserWhere1: UserWhere1{
			Page: 2,
			Size: 10,
		},
	}
	assert.Equal(
		t,
		"SELECT * FROM `user` LIMIT 10 OFFSET 10",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u2)).Find(&User{})
		}), "should be equal",
	)
}

func TestOrder(t *testing.T) {
	var b bool
	type UserOrder struct {
		NameOrder bool  `op:"asc" field:"name"`
		Age       bool  `op:"desc"`
		Birth     *bool `op:"asc"`
		Names     bool  `op:"desc" field:"father,mother"`
	}
	u1 := UserOrder{
		NameOrder: true,
		Age:       true,
		Birth:     &b,
		Names:     true,
	}
	assert.Equal(
		t,
		"SELECT * FROM `user` ORDER BY `name` ASC,`age` DESC,`birth` DESC,`father` DESC,`mother` DESC",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u1)).Find(&User{})
		}), "should be equal",
	)

}

func TestCount(t *testing.T) {
	type UserWhere1 struct {
		Page int    `op:"page"`
		Size int    `op:"size"`
		Name string `op:"like" field:"name"`
	}
	u1 := UserWhere1{
		Page: 1,
		Size: 10,
		Name: "To",
	}
	users := User{
		ID:   1,
		Name: "Tony",
	}

	DB.Create(&users)
	var count int64
	var user []User
	DB.Scopes(Where(u1, &count)).Find(&user)
	assert.Equal(
		t,
		int64(1),
		count, "should be equal",
	)
}
