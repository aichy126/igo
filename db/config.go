package db

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"
	"xorm.io/xorm"
)

type DBConfig struct {
	MaxIdle     int    `json:"max_idle" toml:"max_idle" yaml:"max_idle"`
	MaxOpen     int    `json:"max_open" toml:"max_open" yaml:"max_open"`
	MaxIdleLife int    `json:"max_idle_life" toml:"max_idle_life" yaml:"max_idle_life"`
	IsDebug     bool   `json:"is_debug" toml:"is_debug" yaml:"is_debug"`
	Datasource  string `json:"datasource" toml:"datasource" yaml:"datasource"`
}

func NewDBConfigWithYamlPath(path string) (dbcf *DBConfig, err error) {
	dbcf = &DBConfig{}
	err = nil
	//err = config.YGetObject(path, "write", dbcf)
	return
}

func (db DBConfig) String() string {
	data, _ := json.Marshal(db)
	return string(data)
}

func (db *DBConfig) parse(tree map[string]interface{}) error {

	var ok bool
	//MaxIdle
	_, ok = tree["max_idle"].(int64)
	db.MaxIdle = 10
	if ok {
		db.MaxIdle = int(tree["max_idle"].(int64))
	}
	//MaxOpen
	_, ok = tree["max_open"].(int64)
	db.MaxOpen = 10
	if ok {
		db.MaxOpen = int(tree["max_open"].(int64))
	}
	//MaxIdleLife
	_, ok = tree["max_idle_life"].(int64)
	db.MaxIdleLife = default_idle_life_time
	if ok {
		db.MaxIdleLife = int(tree["max_idle_life"].(int64))
	}
	//IsDebug
	_, ok = tree["is_debug"].(bool)
	db.IsDebug = true
	if ok {
		db.IsDebug = tree["is_debug"].(bool)
	}
	//Datasource
	_, ok = tree["datasource"].(string)
	db.Datasource = ""
	if ok {
		db.Datasource = tree["datasource"].(string)
	}

	db.Datasource = strings.TrimSpace(db.Datasource)
	return nil
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
