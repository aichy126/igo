package db

import (
	"container/ring"
	"errors"
	"strings"

	"xorm.io/xorm"
)

// DatabaseManager
type DatabaseManager struct {
	datasource string
	r          *ring.Ring
	WriteDB    *xorm.Engine
}

// String
func (db DatabaseManager) String() string {
	return db.datasource
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

func (db *DatabaseManager) initWriterDb(conf *DBConfig) (err error) {
	var rc DBConfig
	rc = *conf
	db.WriteDB, err = rc.newDB()
	if err != nil {
		return
	}
	db.datasource = rc.Datasource
	return
}

func (db *DatabaseManager) initWriteDbWithConfig(rc DBConfig) (err error) {
	db.WriteDB, err = rc.newDB()
	if err != nil {
		return
	}
	db.datasource = strings.TrimSpace(rc.Datasource)
	return
}

func (db *DatabaseManager) db(typ string) *xorm.Engine {
	return db.WriteDB
}
