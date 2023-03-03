package structquery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

func TestMain(m *testing.M) {
	var err error
	DB, err = gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{
		DryRun: true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		panic(err)
	}
	m.Run()
}

type User struct {
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
		"SELECT * FROM `user` WHERE name = \"foo\" AND age = 12",
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
		"SELECT * FROM `user` WHERE (name = \"foo\" OR age = 12)",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u)).Find(&User{})
		}), "should be equal",
	)
}

func TestOrType2(t *testing.T) {
	type UserAnd struct {
		Addr  string
		Addr2 string
	}

	type UserOr struct {
		Name string
		Age  int
		And  UserAnd
	}
	type UserWhere struct {
		UserOr
		Addr string
	}

	u := UserWhere{
		UserOr: UserOr{
			Name: "foo",
			Age:  12,
			And: UserAnd{
				Addr:  "foo",
				Addr2: "bar",
			},
		},
		Addr: "bar",
	}
	assert.Equal(
		t,
		"SELECT * FROM `user` WHERE (name = \"foo\" OR age = 12)",
		DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Scopes(Where(u)).Find(&User{})
		}), "should be equal",
	)
}
