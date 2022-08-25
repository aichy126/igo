package igo

import (
	"github.com/aichy126/igo/cache"
	"github.com/aichy126/igo/config"
	"github.com/aichy126/igo/db"
	"github.com/aichy126/igo/log"
	"github.com/aichy126/igo/web"
)

type Application struct {
	Conf  *config.Config
	Web   *web.Web
	DB    *db.DB
	Cache *cache.Cache
}

var App *Application

func NewApp(ConfigPath string) *Application {
	a := new(Application)
	//config
	conf, err := config.NewConfig(ConfigPath)
	if err != nil {
		panic(err)
	}
	a.Conf = conf

	//web
	web, err := web.NewWeb(conf)
	if err != nil {
		panic(err)
	}
	a.Web = web

	//log
	_, err = log.NewLog(conf)
	if err != nil {
		panic(err)
	}

	//db
	db, err := db.NewDb(conf)
	if err != nil {
		panic(err)
	}
	a.DB = db

	//cahce
	cache, err := cache.NewCache(conf)
	if err != nil {
		panic(err)
	}
	a.Cache = cache

	return a
}
