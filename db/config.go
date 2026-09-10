package db

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aichy126/igo/log"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"modernc.org/sqlite"
	"xorm.io/xorm"
)

// SQLite 驱动用 modernc.org/sqlite（纯 Go）而不是 mattn/go-sqlite3：
// mattn 要 CGO，`CGO_ENABLED=0` 交叉编译出来的二进制里它只剩一个 Open 就报错的桩。
// xorm 的 sqlite 方言固定找 database/sql 里叫 "sqlite3" 的驱动，所以把 modernc 也注册到这个名字上
// （modernc 自己注册的是 "sqlite"，两个名字指向同一个实现）。
func init() {
	sql.Register("sqlite3", &sqlite.Driver{})
}

// sqliteDefaultPragmas 没写 _pragma 的 SQLite 连接串补上这两条：
//   - busy_timeout：另一个连接在写时等它，而不是立刻报 "database is locked"；
//   - journal_mode(WAL)：读不阻塞写、写不阻塞读，多连接下锁冲突基本只剩「两个写者同时提交」这一种，
//     由 busy_timeout 兜底。
//
// 这两条是 SQLite 多连接使用的底线配置，忘写就会在并发下随机报锁错误，所以由框架兜底而不是靠每个项目记得。
const sqliteDefaultPragmas = "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"

func sqliteDSN(ds string) string {
	if strings.Contains(ds, "_pragma=") {
		return ds
	}
	if strings.Contains(ds, "?") {
		return ds + "&" + sqliteDefaultPragmas
	}
	return ds + "?" + sqliteDefaultPragmas
}

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
		applyEnvOverride(conf, "mysql."+k, data)
		dbConfigList[k] = data
	}

	for k, v := range sqliteConfigList {
		data := new(DBConfig)
		err := mapstructure.Decode(v, data)
		if err != nil {
			return fmt.Errorf("sqlite 配置 [sqlite.%s] 解析失败: %w", k, err)
		}
		data.DbType = "sqlite3"
		applyEnvOverride(conf, "sqlite."+k, data)
		data.Datasource = sqliteDSN(data.Datasource)
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

// applyEnvOverride 让 IGO_ 环境变量对数据库配置生效。
//
// 为什么需要这一步：viper 的 AutomaticEnv 只在按【完整 key】调用 Get 系列方法时生效，
// 而上面用的 GetStringMap 返回的是配置文件里的原始 map，环境变量不会被合并进去
// （viper 的已知行为）。缺了这一步，config 包承诺的
// 「环境变量优先级高于配置文件，适合 Docker/K8s 部署时无需改配置文件」
// 对 mysql / sqlite 这类嵌套配置就不成立——IGO_MYSQL_DEFAULT_DATA_SOURCE
// 会被静默忽略，而 data_source 恰恰是最需要在容器里注入的东西。
//
// 只在环境变量给出有效值时才覆盖：GetString 在环境变量缺失时会回落到配置文件的值，
// 所以这里的判空既挡住了空环境变量，也保证了无环境变量时行为不变。
func applyEnvOverride(conf *viper.Viper, prefix string, c *DBConfig) {
	if v := strings.TrimSpace(conf.GetString(prefix + ".data_source")); v != "" {
		c.Datasource = v
	}
	if v := conf.GetInt(prefix + ".max_idle"); v > 0 {
		c.MaxIdle = v
	}
	if v := conf.GetInt(prefix + ".max_open"); v > 0 {
		c.MaxOpen = v
	}
	if v := conf.GetInt(prefix + ".max_idle_life"); v > 0 {
		c.MaxIdleLife = v
	}
	if conf.IsSet(prefix + ".is_debug") {
		c.IsDebug = conf.GetBool(prefix + ".is_debug")
	}
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
