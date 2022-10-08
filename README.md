# igo
golang web项目脚手架,对常用组件进行封装,通过配置文件初始化后即可方便使用,避免每次创建新项目都需要初始化各种组件的业务逻辑
## 包含组件
 -  `viper` github.com/spf13/viper 配置
 -  `xorm` xorm.io/xorm mysql orm
 -  `gin` github.com/gin-gonic/gin web框架
 -  `pprof` net/http/pprof gin debug 模式默认打开pprof
 -  `zap` go.uber.org/zap 日志处理
 -  `context` 简单封装
 -  `redis` github.com/go-redis/redis/v8
 -  `util` 常用函数
 -  `httpclient` http 请求简单封装

## 如何初始化
```golang
func main() {
	igo.App = igo.NewApp("") //初始化各个组件
	Router(igo.App.Web.Router) //引入 gin路由
	igo.App.Web.Run()
}

func Router(r *gin.Engine) {
	r.GET("ping", Ping)
}

func Ping(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Hello World."})
}
```

## 配置文件
配置文件可以使用本地配置文件和consul配置中心

#### 本地配置文件
```toml
[local]
address = ":8001" # host and port
debug   = true    # debug mode for Gin

[local.logger]
dir   = "./logs" #日志路径
name   = "log.log" #日志路径
access = true # 是否记录access日志
level = "INFO"
max_size = 1  #每个日志文件保存的最大尺寸 单位：M
max_backups = 5 #文件最多保存多少天
max_age = 7 #日志文件最多保存多少个备份


[mysql.igo]
maxIdle = 10
maxOpen = 20
isDebug = true
dataSource = "root:root@tcp(127.0.0.1:3306)/igo?charset=utf8&interpolateParams=true&timeout=3s&readTimeout=3s&writeTimeout=3s"

[redis.igo]
address = "127.0.0.1:6379"
password = "aichenyang"
db = 0
poolsize = 50

```
#### 本地配置文件指向配置中心
```toml
[config]
address = "127.0.0.1:8500"
key ="/igo/config"
```

### 如何找到配置文件
 1. go run main.go -c config.toml 使用 -c 加本地配置文件路径
 2. export CONFIG_PATH=./config.toml 使用环境变量指定本地配置文件
 3. 不使用本地配置文件环境变量直接指向配置中心
 ```shell
  export CONFIG_ADDRESS=127.0.0.1:8500
  export CONFIG_KEY=/igo/config
```

## 如何使用各个组件
配置文件中的 redis 和mysql 可以设置多个使用的时候只需要选择对应的配置即可
```golang
//配置文件 Conf是viper的封装
igo.App.Conf.GetString("xxx.xxx")
//日志 log是zap的封装
log.Info("hello igo", log.Any("nowtime", time.Now().Unix()))
log.Error("error", log.Any("nowtime", time.Now().Unix()))
//xorm db是xorm的封装
db := igo.App.DB.NewDBTable("dbname", "news")
session := db.Where("")
err := session.OrderBy("id desc").Find(&rows)
//redis
//redisconfig是配置文件中的redis配置项
redis, err := igo.App.Cache.Get("redisconfig")
getRedisKey, err := redis.Get(ctx, "sfredsi").Result()
spew.Dump(getRedisKey, err)

```
