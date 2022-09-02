package db

import (
	"errors"
	"reflect"
	"sync"

	"github.com/aichy126/igo/config"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
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
	*xorm.Engine
}

func (s *DB) SelectDB(dbname string) *Repo {
	return &Repo{s.Get(dbname).WriteDB}
}

const (
	default_idle_life_time = 3600
)

// DBResourceManager 数据库源地址管理
type DBResourceManager struct {
	mutex     sync.RWMutex
	resources map[string]*DatabaseManager
}

func (db *DBResourceManager) Get(name string) *DatabaseManager {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	return db.resources[name]
}

func New(conf *viper.Viper) *DBResourceManager {
	m := &DBResourceManager{
		resources: make(map[string]*DatabaseManager),
	}
	err := m.initFromToml(conf)
	if err != nil && reflect.TypeOf(err) != reflect.TypeOf(dbconfigNotFound) {
		panic(err)
	}

	return m
}

var dbconfigNotFound = errors.New("notfound")

func (db *DBResourceManager) initFromToml(conf *viper.Viper) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	mysqlList := make(map[string]*DBConfig, 0)
	err := conf.UnmarshalKey("mysql", &mysqlList)
	if err != nil {
		return err
	}
	for name, itemDBConfig := range mysqlList {
		dm := new(DatabaseManager)
		err := dm.initWriterDb(itemDBConfig)
		if err != nil {
			return err
		}
		db.resources[name] = dm
		continue
	}
	return nil
}
