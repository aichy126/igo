package db

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aichy126/igo/log"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mitchellh/mapstructure"
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
		return nil, fmt.Errorf("创建 xorm engine 失败: %w", err)
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

// Names 返回所有已初始化的数据库配置名
func (db *DBResourceManager) Names() []string {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	names := make([]string, 0, len(db.resources))
	for name := range db.resources {
		names = append(names, name)
	}
	return names
}

// PingAll 对所有数据库执行 Ping,返回每个库的结果(nil 表示正常)
func (db *DBResourceManager) PingAll() map[string]error {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	result := make(map[string]error, len(db.resources))
	for name, dm := range db.resources {
		result[name] = dm.Ping()
	}
	return result
}

// New 从配置初始化所有数据库连接
// 配置了的数据库必须初始化并 Ping 成功,否则返回错误(fail-fast);
// 完全没有 [mysql]/[sqlite] 配置时返回空 manager,不报错。
func New(conf *viper.Viper) (*DBResourceManager, error) {
	m := &DBResourceManager{
		resources: make(map[string]*DatabaseManager),
	}
	if err := m.initFromToml(conf); err != nil {
		return nil, err
	}
	return m, nil
}

func (db *DBResourceManager) initFromToml(conf *viper.Viper) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	dbConfigList := make(map[string]*DBConfig)
	mysqlConfigList := conf.GetStringMap("mysql")
	sqliteConfigList := conf.GetStringMap("sqlite")

	for k, v := range mysqlConfigList {
		data := new(DBConfig)
		err := mapstructure.Decode(v, data)
		if err != nil {
			return fmt.Errorf("mysql 配置 [mysql.%s] 解析失败: %w", k, err)
		}
		data.DbType = "mysql"
		dbConfigList[k] = data
	}

	for k, v := range sqliteConfigList {
		data := new(DBConfig)
		err := mapstructure.Decode(v, data)
		if err != nil {
			return fmt.Errorf("sqlite 配置 [sqlite.%s] 解析失败: %w", k, err)
		}
		data.DbType = "sqlite3"
		dbConfigList[k] = data
	}

	for name, itemDBConfig := range dbConfigList {
		if strings.TrimSpace(itemDBConfig.Datasource) == "" {
			return fmt.Errorf("数据库 [%s] 缺少 data_source 配置", name)
		}
		dm := new(DatabaseManager)
		if err := dm.initWriterDb(itemDBConfig); err != nil {
			return fmt.Errorf("数据库 [%s] 初始化失败: %w", name, err)
		}
		if err := dm.Ping(); err != nil {
			return fmt.Errorf("数据库 [%s] 连接失败(ping): %w", name, err)
		}
		db.resources[name] = dm
	}
	return nil
}

// DatabaseManager
type DatabaseManager struct {
	datasource string
	WriteDB    *xorm.Engine
}

func (db *DatabaseManager) initWriterDb(conf *DBConfig) (err error) {
	rc := *conf
	db.WriteDB, err = rc.newDB()
	if err != nil {
		return
	}
	db.datasource = strings.TrimSpace(rc.Datasource)
	return
}

// Ping Database
func (db *DatabaseManager) Ping() error {
	if db == nil || db.WriteDB == nil {
		return fmt.Errorf("invalid database config")
	}
	return db.WriteDB.Ping()
}

// Close 关闭所有数据库连接
func (db *DBResourceManager) Close() error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	for name, dm := range db.resources {
		if dm != nil && dm.WriteDB != nil {
			if err := dm.WriteDB.Close(); err != nil {
				log.Error("关闭数据库连接失败", log.Any("name", name), log.Any("error", err))
			}
		}
	}
	return nil
}
