package db

import (
	"context"
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

// NewDb 从配置初始化数据库
// 配置了的数据库连接失败会返回错误(fail-fast);完全没配置数据库时返回可用的空实例
func NewDb(conf *config.Config) (*DB, error) {
	db := new(DB)
	manager, err := New(conf.Viper)
	if err != nil {
		return nil, err
	}
	db.DBResourceManager = manager
	return db, nil
}

// Repo Reference xorm
type Repo struct {
	Engine    *xorm.Engine
	tableName string
}

// NewDBTable 获取绑定到指定库和表的操作对象
// dbname 必须是配置文件中 [mysql.xxx]/[sqlite.xxx] 里的配置名,不存在时 panic 并给出明确提示
// (数据库连接在 NewApp 阶段已保证可用,走到这里失败只可能是配置名写错,尽早暴露)
func (s *DB) NewDBTable(dbname string, tableName string) *Repo {
	dm := s.Get(dbname)
	if dm == nil || dm.WriteDB == nil {
		panic(fmt.Sprintf("数据库 [%s] 不存在,请检查配置文件中的 [mysql.%s] 或 [sqlite.%s] 配置", dbname, dbname, dbname))
	}
	return &Repo{dm.WriteDB, tableName}
}

// SetTableName 修改 Repo 绑定的表名
func (repo *Repo) SetTableName(name string) {
	if len(name) == 0 {
		return
	}
	repo.tableName = name
}

func (repo *Repo) NewSession() *xorm.Session {
	var sess *xorm.Session
	newSession := repo.Engine.NewSession()
	sess = newSession.Table(repo.tableName)
	return sess
}

// WithCtx 返回绑定了 context 的查询 session:请求取消/超时后查询会被中断,
// 慢查询不会在客户端断开后继续占用数据库。igo 的 context.IContext 可直接传入。
// 使用示例：
//
//	err := repo.WithCtx(ctx).Where("uid = ?", uid).Find(&rows)
func (repo *Repo) WithCtx(ctx context.Context) *xorm.Session {
	return repo.Engine.Table(repo.tableName).Context(ctx)
}

func (repo *Repo) InsertOne(beans any) (int64, error) {
	sess := repo.NewSession()
	defer sess.Close()
	r, err := sess.InsertOne(beans)
	if err != nil {
		log.Error("Mysql", log.Any("error", err.Error()))
	}
	return r, err
}

func (repo *Repo) Insert(beans ...any) (int64, error) {
	sess := repo.NewSession()
	defer sess.Close()
	r, err := sess.Insert(beans...)
	if err != nil {
		log.Error("Mysql", log.Any("error", err.Error()))
	}
	return r, err
}

func (repo *Repo) Get(bean any) (bool, error) {
	sess := repo.NewSession()
	defer sess.Close()
	r, err := sess.Get(bean)
	if err != nil {
		log.Error("Mysql", log.Any("error", err.Error()))
	}
	return r, err
}

func (repo *Repo) Where(query any, args ...any) *xorm.Session {
	return repo.Engine.Table(repo.tableName).Where(query, args...)
}

func (repo *Repo) Select(query string) *xorm.Session {
	return repo.Engine.Table(repo.tableName).Select(query)
}

func (repo *Repo) In(column string, args ...any) *xorm.Session {
	return repo.Engine.Table(repo.tableName).In(column, args...)
}

func (repo *Repo) Query(sql string, paramStr ...any) (resultsSlice []map[string][]byte, err error) {
	sess := repo.NewSession()
	args := make([]any, 0)
	args = append(args, sql)
	args = append(args, paramStr...)
	defer sess.Close()
	r, err := sess.Query(args...)
	if err != nil {
		log.Error("Mysql", log.Any("error", err.Error()))
	}
	return r, err
}

func (repo *Repo) Find(bean any, condiBeans ...any) error {
	sess := repo.NewSession()
	defer sess.Close()
	err := sess.Find(bean, condiBeans)
	if err != nil {
		log.Error("Mysql", log.Any("error", err.Error()))
	}
	return err
}

func (repo *Repo) Exec(sql string, args ...any) (sql.Result, error) {
	sess := repo.NewSession()
	defer sess.Close()
	params := make([]any, 0, len(args)+1)
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

// Transaction 闭包式事务:fn 返回 error 或 panic 时自动回滚,正常返回时自动提交。
// 相比 BeginTx 免去手动管理 Commit/Rollback/Close,推荐优先使用。
// 使用示例：
//
//	err := igo.App.DB.Transaction("test", func(sess *xorm.Session) error {
//	    if _, err := sess.Table("users").Insert(&user); err != nil {
//	        return err // 自动回滚
//	    }
//	    _, err := sess.Table("orders").Insert(&order)
//	    return err // nil 则自动提交
//	})
func (s *DB) Transaction(dbname string, fn func(sess *xorm.Session) error) error {
	sess, err := s.BeginTx(dbname)
	if err != nil {
		return err
	}
	defer sess.Close()

	defer func() {
		// fn panic 时回滚后继续向上抛,避免连接带着未完成事务归还连接池
		if r := recover(); r != nil {
			_ = sess.Rollback()
			panic(r)
		}
	}()

	if err := fn(sess); err != nil {
		if rbErr := sess.Rollback(); rbErr != nil {
			log.Error("事务回滚失败", log.Any("dbname", dbname), log.Any("error", rbErr))
		}
		return err
	}
	return sess.Commit()
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
