package db

import (
	"database/sql"
	"fmt"

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
	repo.Engine.Table(name)
}

func (repo *Repo) NewSession() *xorm.Session {
	var sess *xorm.Session
	newSession := repo.Engine.NewSession()
	sess = newSession.Table(repo.tableName)
	return sess
}

func (repo *Repo) InsertOne(beans interface{}) (int64, error) {
	sess := repo.NewSession()
	defer sess.Close()
	r, err := sess.InsertOne(beans)
	if err != nil {
		log.Error("Mysql", log.Any("error", err.Error()))
	}
	return r, err
}

func (repo *Repo) Insert(beans ...interface{}) (int64, error) {
	sess := repo.NewSession()
	defer sess.Close()
	r, err := sess.Insert(beans...)
	if err != nil {
		log.Error("Mysql", log.Any("error", err.Error()))
	}
	return r, err
}

func (repo *Repo) Get(bean interface{}) (bool, error) {
	sess := repo.NewSession()
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
	sess := repo.NewSession()
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
	sess := repo.NewSession()
	defer sess.Close()
	err := sess.Find(bean, condiBeans)
	if err != nil {
		log.Error("Mysql", log.Any("error", err.Error()))
	}
	return err
}

func (repo *Repo) Exec(sql string, args ...interface{}) (sql.Result, error) {
	sess := repo.NewSession()
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

// Close 关闭所有数据库连接
func (s *DB) Close() error {
	if s.DBResourceManager != nil {
		return s.DBResourceManager.Close()
	}
	return nil
}

// NewSession 获取指定数据库的原始Session（未绑定表，可用于跨表操作）
// 使用示例：
//
//	sess := igo.App.DB.NewSession("test")
//	defer sess.Close()
//	sess.Table("users").Insert(&user)
//	sess.Table("orders").Insert(&order)
func (s *DB) NewSession(dbname string) *xorm.Session {
	dm := s.Get(dbname)
	if dm == nil || dm.WriteDB == nil {
		log.Error("数据库不存在或未初始化", log.Any("dbname", dbname))
		return nil
	}
	return dm.WriteDB.NewSession()
}

// BeginTx 启动事务（支持跨表操作）
// 使用示例：
//
//	sess, err := igo.App.DB.BeginTx("test")
//	if err != nil {
//	    return err
//	}
//	defer sess.Close()
//
//	sess.Table("users").Insert(&user)
//	sess.Table("orders").Insert(&order)
//	err = sess.Commit()
func (s *DB) BeginTx(dbname string) (*xorm.Session, error) {
	sess := s.NewSession(dbname)
	if sess == nil {
		return nil, fmt.Errorf("无法创建Session，数据库: %s", dbname)
	}
	if err := sess.Begin(); err != nil {
		sess.Close()
		return nil, fmt.Errorf("启动事务失败: %w", err)
	}
	return sess, nil
}
