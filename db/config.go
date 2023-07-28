package db

import (
	"container/ring"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/aichy126/igo/log"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"xorm.io/xorm"
)

const (
	defaultIdleLifeTime = 3600
)

type DBConfig struct {
	MaxIdle     int    `json:"max_idle" toml:"max_idle" yaml:"max_idle" mapstructure:"max_idle"`
	MaxOpen     int    `json:"max_open" toml:"max_open" yaml:"max_open" mapstructure:"max_open"`
	MaxIdleLife int    `json:"max_idle_life" toml:"max_idle_life" yaml:"max_idle_life" mapstructure:"max_idle_life"`
	IsDebug     bool   `json:"is_debug" toml:"is_debug" yaml:"is_debug" mapstructure:"is_debug"`
	Datasource  string `json:"data_source" toml:"data_source" yaml:"data_source" mapstructure:"data_source"`
	DbType      string `json:"-" toml:"-" yaml:"-" mapstructure:"-"`
}

func (db DBConfig) newDB() (engine *xorm.Engine, err error) {
	orm, err := xorm.NewEngine(db.DbType, db.Datasource)
	if err != nil {
		err = errors.Wrap(err, "conn xorm db error ")
		return
	}

	orm.DatabaseTZ = time.Local
	orm.TZLocation = time.Local
	orm.SetMaxOpenConns(db.MaxOpen)
	orm.SetMaxIdleConns(db.MaxIdle)
	if db.MaxIdleLife == 0 {
		db.MaxIdleLife = defaultIdleLifeTime
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
	if err != nil && reflect.TypeOf(err) != reflect.TypeOf(dbConfigNotFound) {
		panic(err)
	}

	return m
}

var dbConfigNotFound = errors.New("Db config not found")

func (db *DBResourceManager) initFromToml(conf *viper.Viper) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	mysqlList := make(map[string]*DBConfig, 0)
	mysqlConfigList := conf.GetStringMap("mysql")
	sqliteConfigList := conf.GetStringMap("sqlite")

	for k, v := range mysqlConfigList {
		data := new(DBConfig)
		err := mapstructure.Decode(v, data)
		if err != nil {
			continue
		}
		data.DbType = "mysql"
		mysqlList[k] = data
	}

	for k, v := range sqliteConfigList {
		data := new(DBConfig)
		err := mapstructure.Decode(v, data)
		if err != nil {
			continue
		}
		data.DbType = "sqlite3"
		mysqlList[k] = data
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

// Ping Database
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
