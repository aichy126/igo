package db

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"xorm.io/xorm"
)

type DBConfig struct {
	MaxIdle     int    `json:"maxIdle" toml:"maxIdle" yaml:"maxIdle"`
	MaxOpen     int    `json:"maxOpen" toml:"maxOpen" yaml:"maxOpen"`
	MaxIdleLife int    `json:"maxIdleLife" toml:"maxIdleLife" yaml:"maxIdleLife"`
	IsDebug     bool   `json:"isDebug" toml:"isDebug" yaml:"isDebug"`
	Datasource  string `json:"dataSource" toml:"dataSource" yaml:"dataSource"`
}

func NewDBConfigWithYamlPath(path string) (dbcf *DBConfig, err error) {
	dbcf = &DBConfig{}
	err = nil
	return
}

func (db DBConfig) String() string {
	data, _ := json.Marshal(db)
	return string(data)
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
