package db

import (
	"container/ring"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/aichy126/igo/log"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"xorm.io/xorm"
)

const (
	default_idle_life_time = 3600
)

type DBConfig struct {
	MaxIdle     int    `json:"max_idle" toml:"max_idle" yaml:"max_idle"`
	MaxOpen     int    `json:"max_open" toml:"max_open" yaml:"max_open"`
	MaxIdleLife int    `json:"max_idle_life" toml:"max_idle_life" yaml:"max_idle_life"`
	IsDebug     bool   `json:"is_debug" toml:"is_debug" yaml:"is_debug"`
	Datasource  string `json:"data_source" toml:"data_source" yaml:"data_source"`
}

func (db DBConfig) newDB() (engine *xorm.Engine, err error) {
	orm, err := xorm.NewEngine("mysql", db.Datasource)
	if err != nil {
		err = errors.Wrap(err, "conn mysql error ")
		return
	}

	orm.DatabaseTZ = time.Local
	orm.TZLocation = time.Local
	orm.SetMaxOpenConns(db.MaxOpen)
	orm.SetMaxIdleConns(db.MaxIdle)
	if db.MaxIdleLife == 0 {
		db.MaxIdleLife = default_idle_life_time
	}
	orm.SetConnMaxLifetime(time.Duration(db.MaxIdleLife) * time.Second)
	orm.ShowSQL(db.IsDebug)
	return orm, err
}

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

var dbconfigNotFound = errors.New("Db config not found")

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
		err = dm.Ping()
		if err != nil {
			log.Error("mysql ping error", log.Any("config_name", name), log.Any("error", err))
			continue
		}
		db.resources[name] = dm
		continue
	}
	return nil
}

// DatabaseManager
type DatabaseManager struct {
	datasource string
	r          *ring.Ring
	WriteDB    *xorm.Engine
}

func (db *DatabaseManager) initWriterDb(conf *DBConfig) (err error) {
	var rc DBConfig
	rc = *conf
	db.WriteDB, err = rc.newDB()
	if err != nil {
		return
	}
	//db.datasource = rc.Datasource
	db.datasource = strings.TrimSpace(rc.Datasource)
	return
}

// Ping
func (db *DatabaseManager) Ping() error {
	if db == nil {
		return errors.New("invalid database config")
	}
	err := db.WriteDB.Ping()
	if err != nil {
		return err
	}

	for i := 0; i < db.r.Len(); i++ {
		engine := db.r.Value.(*xorm.Engine)
		if err := engine.Ping(); err != nil {
			return err
		}
		db.r = db.r.Next()
	}

	return nil
}
