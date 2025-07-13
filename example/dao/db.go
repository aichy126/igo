package dao

import (
	"time"

	"github.com/aichy126/igo"
	"github.com/aichy126/igo/db"
	"github.com/aichy126/igo/ictx"
)

type Test0 struct {
	ID        int64     `xorm:"not null pk autoincr BIGINT(20) id"`
	CreatedAt time.Time `xorm:"created not null default CURRENT_TIMESTAMP TIMESTAMP created_at"`
	UpdatedAt time.Time `xorm:"updated_at TIMESTAMP"`
	UserID    string    `xorm:"not null default 0 BIGINT(20) INDEX user_id"`
}

type Test1 struct {
	ID     string `xorm:"not null pk autoincr BIGINT(20) id"`
	UserID string `xorm:"not null default 0 BIGINT(20) INDEX user_id"`
}

type Test2 struct {
	ID        int64     `xorm:"not null pk autoincr BIGINT(20) id"`
	CreatedAt time.Time `xorm:"created not null default CURRENT_TIMESTAMP TIMESTAMP created_at"`
	UpdatedAt time.Time `xorm:"updated_at TIMESTAMP"`
}

type TestDbDao struct {
	db *db.Repo
}

// NewTestDbDao
func NewTestDbDao() *TestDbDao {
	return &TestDbDao{
		db: igo.App.DB.NewDBTable("test", "test0"),
	}
}

func (m *TestDbDao) Info(ctx ictx.Context, ID int64) (*Test0, bool, error) {
	Data := new(Test0)
	has, err := m.db.Where("id =? ", ID).Get(Data)
	if err != nil {
		return Data, false, err
	}
	return Data, has, nil
}

func (m *TestDbDao) Sync(ctx ictx.Context) error {
	m.db.Engine.Sync(new(Test0), new(Test1), new(Test2))
	return nil
}
