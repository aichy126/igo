package db

import (
	"database/sql"

	"github.com/aichy126/igo/config"
	"github.com/aichy126/igo/log"
	_ "github.com/go-sql-driver/mysql"
	"xorm.io/xorm"
)

// Db
type DB struct {
	*DBResourceManager
}

// NewDb
func NewDb(conf *config.Config) (*DB, error) {
	db := new(DB)
	manager := New(conf.Viper)
	db.DBResourceManager = manager
	return db, nil
}

// Repo Reference xorm
type Repo struct {
	Engine    *xorm.Engine
	tableName string
}

func (s *DB) NewDBTable(dbname string, tableName string) *Repo {
	return &Repo{s.Get(dbname).WriteDB, tableName}
}
func (repo *Repo) SetTableName(name string) {
	if len(name) == 0 {
		return
	}
	repo.tableName = name
}

func (repo *Repo) InsertOne(beans interface{}) (int64, error) {
	sess := repo.Engine.NewSession()
	defer sess.Close()
	r, err := sess.InsertOne(beans)
	if err != nil {
		log.Error("Mysql", log.Any("error", err.Error()))
	}
	return r, err
}

func (repo *Repo) Insert(beans ...interface{}) (int64, error) {
	sess := repo.Engine.NewSession()
	defer sess.Close()
	r, err := sess.Insert(beans...)
	if err != nil {
		log.Error("Mysql", log.Any("error", err.Error()))
	}
	return r, err
}

func (repo *Repo) Get(bean interface{}) (bool, error) {
	sess := repo.Engine.NewSession()
	defer sess.Close()
	r, err := sess.Get(bean)
	if err != nil {
		log.Error("Mysql", log.Any("error", err.Error()))
	}
	return r, err
}

func (repo *Repo) Where(query interface{}, args ...interface{}) *xorm.Session {
	return repo.Engine.Table(repo.tableName).Where(query, args...)
}

func (repo *Repo) Select(query string) *xorm.Session {
	return repo.Engine.Table(repo.tableName).Select(query)
}

func (repo *Repo) In(column string, args ...interface{}) *xorm.Session {
	return repo.Engine.Table(repo.tableName).In(column, args...)
}

func (repo *Repo) Query(sql string, paramStr ...interface{}) (resultsSlice []map[string][]byte, err error) {
	sess := repo.Engine.NewSession()
	args := make([]interface{}, 0)
	args = append(args, sql)
	args = append(args, paramStr...)
	defer sess.Close()
	r, err := sess.Query(args...)
	if err != nil {
		log.Error("Mysql", log.Any("error", err.Error()))
	}
	return r, err
}

func (repo *Repo) Find(bean interface{}, condiBeans ...interface{}) error {
	sess := repo.Engine.NewSession()
	defer sess.Close()
	err := sess.Find(bean, condiBeans)
	if err != nil {
		log.Error("Mysql", log.Any("error", err.Error()))
	}
	return err
}

func (repo *Repo) Exec(sql string, args ...interface{}) (sql.Result, error) {
	sess := repo.Engine.NewSession()
	defer sess.Close()
	params := make([]interface{}, 0, len(args)+1)
	params = append(params, sql)
	for _, arg := range args {
		params = append(params, arg)
	}
	r, err := repo.Engine.Exec(params...)
	if err != nil {
		log.Error("Mysql", log.Any("error", err.Error()))
	}
	return r, err
}

func (repo *Repo) ShowSQL(b bool) {
	repo.Engine.ShowSQL(b)
}
