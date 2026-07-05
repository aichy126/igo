# IGo [![Go Report Card](https://goreportcard.com/badge/github.com/aichy126/igo)](https://goreportcard.com/report/github.com/aichy126/igo) [![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/) ![GitHub](https://img.shields.io/github/license/aichy126/igo)

golang web项目脚手架,对常用组件进行封装,通过配置文件初始化后即可方便使用,避免每次创建新项目都需要初始化各种组件的业务逻辑

## 包含组件

- `viper` github.com/spf13/viper 配置(支持文件/Consul,热重载,`IGO_` 前缀环境变量覆盖)
- `xorm` xorm.io/xorm mysql/sqlite orm(闭包事务、ctx 传递)
- `gin` github.com/gin-gonic/gin web框架
- `pprof` net/http/pprof(仅 debug 模式或 `local.pprof = true` 时开启)
- `zap` go.uber.org/zap 日志处理(支持日志钩子、级别热更新)
- `context` 简单封装(traceId 自动生成/透传)
- `redis` github.com/redis/go-redis/v9
- `res` 统一 JSON 响应格式
- `util` 常用函数(类型转换、分页等,零第三方依赖)
- `httpclient` 轻量 HTTP 客户端(ctx-first、JSON 便捷方法、重试、traceId 自动透传)
- 内置 `/health` 健康检查(可选开启)、CORS 中间件、优雅关闭

## 如何初始化

```golang
func main() {
	app, err := igo.NewApp("") //初始化各个组件,igo.App 全局实例自动设置
	if err != nil {
		fmt.Println("初始化失败:", err)
		os.Exit(1)
	}
	app.EnableHealthCheck()          //可选:开启 GET /health(带 db/redis 连通性检测)
	app.Web.Router.Use(web.Cors())   //可选:开启跨域

	Router(app.Web.Router) //引入 gin路由

	//启动并等待退出信号,自动优雅关闭(Web → Cache → DB 依次关闭)
	//Web 启动失败(如端口被占用)会立即返回错误
	if err := app.Run(); err != nil {
		os.Exit(1)
	}
}

func Router(r *gin.Engine) {
	r.GET("ping", Ping)
}

func Ping(c *gin.Context) {
	res.Rsucc(c, gin.H{"message": "Hello World."})
}
```

### 错误处理语义(fail-fast)

- **配置了的组件必须初始化成功**:`[mysql.*]`/`[sqlite.*]`/`[redis.*]` 配置存在但连接失败时,`NewApp` 返回错误,尽早暴露问题。
- **没配置的组件自动跳过**:不写 db/redis 配置就不初始化,不报错。
- `NewDBTable`/`Cache.Get` 使用不存在的配置名会给出包含配置名的明确报错。
- 日志系统未初始化时调用 `log.Info` 等不会 panic,自动降级为控制台输出。

## 配置文件

配置文件可以使用本地配置文件和consul配置中心

#### 本地配置文件

```toml

[local]
address = ":8001" # host and port
debug   = true    # debug mode for Gin

[local.logger]
dir   = "./logs" #日志目录
name   = "log.log" #日志文件名
access = true # 是否记录access日志
level = "INFO"
max_size = 100  #每个日志文件保存的最大尺寸 单位:MB(默认100)
max_backups = 5 #日志文件最多保存多少个备份(默认5)
max_age = 7 #文件最多保存多少天(默认7)


[mysql.igo]
max_idle = 10
max_open = 20
is_debug = true
data_source = "root:root@tcp(127.0.0.1:3306)/igo?interpolateParams=true&timeout=3s&readTimeout=3s&writeTimeout=3s"

[sqlite.test]
data_source = "test.db"

[redis.igorediskey]
address = "127.0.0.1:6379"
password = "xxx"
db = 0
poolsize = 50
dial_timeout = 1000  # 毫秒,可选
read_timeout = 500   # 毫秒,可选
write_timeout = 500  # 毫秒,可选

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
//配置文件 Conf是viper的封装(热重载时并发安全,请使用 Conf 的 Get* 方法读取)
igo.App.Conf.GetString("xxx.xxx") //直接通过viper读取
util.ConfGetString("local.debug") //util方法读取配置文件

//日志 log是zap的封装
log.Info("hello igo", log.Any("now_time", time.Now().Unix())) //不带traceId
log.Error("error", log.Any("now_time", time.Now().Unix())) //不带traceId
ctx.LogInfo("main-info", log.Any("info", "test")) //包含traceId
ctx.LogError("main-error", log.Any("error", "test")) //包含traceId

//xorm db是xorm的封装
db := igo.App.DB.NewDBTable("dbname", "news")
session := db.Where("")
err := session.OrderBy("id desc").Find(&rows)
//带 context 的查询:请求取消/超时后查询自动中断(推荐)
err = db.WithCtx(ctx).Where("uid = ?", uid).Find(&rows)

//redis(go-redis v9)
//igorediskey是配置文件中的redis配置项
redis, err := igo.App.Cache.Get("igorediskey")
getRedisKey, err := redis.Get(ctx, "redis_key").Result()

//统一响应
res.Rsucc(c, data)                //{"code":0,"msg":"success","data":..}
res.Rfail(c, "错误信息")           //{"code":1,"msg":"..","data":null}
res.Rlist(c, total, items)        //{"code":0,"data":{"total":..,"items":..}}
res.SetCodes(200, 1)              //可选:全局调整成功/失败业务码

//分页(内嵌进业务 Search 结构)
type NoteSearch struct {
	util.PageQuery //Page/PageSize
	Keyword string `form:"keyword"`
}
search.Normalize(20, 100) //Page 从1起,PageSize 默认20、上限100
session.Limit(search.PageSize, search.Offset())
//排序白名单,防SQL注入
orderBy := util.SafeOrderBy(search.Sort, map[string]string{"created": "created_at desc"}, "id desc")
```

### traceId

- 请求进入时自动生成/透传 traceId(读取 `traceId` 或 `X-Trace-Id` 请求头),并写回 `X-Trace-Id` 响应头。
- `ctx := context.Ginform(c)` 后使用 `ctx.LogInfo/LogError` 输出的日志自动带 traceId。

### 事务(跨表操作)

推荐用闭包式 Transaction,自动提交/回滚,不会漏 Close:

```golang
err := igo.App.DB.Transaction("dbname", func(sess *xorm.Session) error {
	if _, err := sess.Table("users").Insert(&user); err != nil {
		return err // 返回 error 自动回滚
	}
	_, err := sess.Table("orders").Insert(&order)
	return err // 返回 nil 自动提交
})
```

也可以手动管理(BeginTx + Commit/Rollback):

```golang
sess, err := igo.App.DB.BeginTx("dbname")
if err != nil {
	return err
}
defer sess.Close()
sess.Table("users").Insert(&user)
sess.Table("orders").Insert(&order)
err = sess.Commit() //失败时 sess.Rollback()
```

### httpclient(HTTP 客户端)

ctx-first 设计;传入 igo 的 `context.IContext` 时,`SetMeta` 设置的 header(含 traceId)自动透传给下游服务:

```golang
client := httpclient.New(
	httpclient.WithTimeout(3*time.Second),
	httpclient.WithRetries(2),          //网络层错误重试,HTTP 状态码错误不重试
	httpclient.WithUserAgent("my-app"),
)

//JSON 便捷方法
var out SomeResp
err := client.PostJSON(ctx, url, reqBody, &out)
err = client.GetJSON(ctx, url, &out)

//通用请求
resp, err := client.Get(ctx, url)
fmt.Println(resp.StatusCode, resp.String())

//下载原始内容(非 2xx 自动报错)
data, err := client.GetBytes(ctx, url)

//单次请求附加 header
err = client.GetJSON(ctx, url, &out, httpclient.WithReqHeader("Authorization", "Bearer xxx"))

//表单提交 + JSON 响应
err = client.PostFormJSON(ctx, url, formValues, &out)

//代理与自签证书场景
proxyClient := httpclient.New(
	httpclient.WithProxyURL("http://127.0.0.1:7890"), //固定代理;动态代理用 WithProxyFunc
	httpclient.WithInsecureSkipVerify(),               //跳过 TLS 校验(生产慎用)
)

//简单场景直接用包级默认客户端
resp, err = httpclient.Get(ctx, url)
```

### 环境变量覆盖配置

`IGO_` 前缀 + 配置路径点号换下划线,优先级高于配置文件,适合 Docker/K8s 部署:

```shell
IGO_LOCAL_ADDRESS=:9000 IGO_LOCAL_DEBUG=false ./myapp
```

### 日志级别热更新

- 文件配置修改 `local.logger.level` 保存后即时生效(配置热重载自动同步),无需重启
- 也可代码调用 `log.SetLevel("debug")` 临时调整

### 日志钩子(如飞书告警)

```golang
type MyHook struct{}
func (h *MyHook) Levels() []zapcore.Level {
	return []zapcore.Level{zapcore.ErrorLevel, zapcore.FatalLevel}
}
func (h *MyHook) Fire(entry *log.LogEntry) error {
	// 发送到飞书、企业微信等(异步执行,不阻塞日志)
	return nil
}
log.AddHook(&MyHook{})
```

### 生命周期钩子

```golang
app.AddStartupHook(func() error { ... })   //Run 时按注册顺序执行
app.AddShutdownHook(func() error { ... })  //关闭时按注册反序执行,默认10秒超时
app.AddConfigChangeCallback(func() { ... })//配置热重载后触发
app.GetShutdownContext()                   //应用关闭时被 cancel,用于停止后台goroutine
```

## 从旧版本升级(迁移说明)

### v0.3.x → v0.4.0

1. **httpclient 完全重写**(API 不兼容):旧的 `NewClient().Debug().SetDefaultTimeout()` 链式 API 和 `HttpSettings`/`HttpRequest` 已删除。迁移示例:

   ```golang
   //旧
   client := httpclient.NewClient().SetDefaultTimeout(3*time.Second).SetDefaultRetries(5)
   err := client.PostJsonAs(ctx, url, in, &out)
   //新
   client := httpclient.New(httpclient.WithTimeout(3*time.Second), httpclient.WithRetries(5))
   err := client.PostJSON(ctx, url, in, &out)
   ```

2. **pprof 默认关闭**:`/debug/pprof/*` 和 `/debug/http/routers` 仅在 `local.debug = true` 或 `local.pprof = true` 时注册(此前无条件公开,有信息泄露风险)。生产环境需要 pprof 的加 `local.pprof = true`。
3. **`ctx.WithValue` 语义变化**:返回携带新值的派生副本,不再原地修改自身(与标准 `context.WithValue` 一致)。依赖旧"修改自身"行为的代码需要改用 `ctx.Set(key, val)`。
4. **`SetMeta`/`GetHeaders` 真正生效**:`SetMeta` 设置的 key 现在会通过 httpclient 自动透传到下游(此前 `GetHeaders` 是空实现);traceId 自动进入透传列表。
5. `util` 类型转换函数移除 gookit/goutil 依赖,行为基本一致(`String(fmt.Stringer)` 仍输出 `String()` 结果)。

### v0.2.x → v0.3.0

依赖全面升级(go 1.23+ / gin v1.12 / go-redis v9 / xorm v1.4 / zap v1.28),下游项目需要注意:

1. **go-redis v8 → v9**:业务代码里如果直接 import 了 `github.com/go-redis/redis/v8`,改为 `github.com/redis/go-redis/v9`,调用代码基本不用动(igo 的 `cache.Redis` 封装 API 不变)。
2. **`context.Ginform` 参数类型**从 `IGetter` 改为 `any`(gin v1.10+ 的 `Context.Get` 签名变化所致),调用方代码不受影响。
3. **行为变化(更健壮)**:
   - db/redis 配置了但连不上时 `NewApp` 返回错误(以前静默跳过,运行时才 panic)。
   - Web 端口被占用等启动失败时 `Run()` 立即返回错误(以前只打日志继续挂起)。
   - `ctx.LogError` 现在正确输出 ERROR 级别(以前误输出 INFO,导致 Error 级别日志钩子不触发)。
   - redis 配置中的 `dial_timeout`/`read_timeout`/`write_timeout` 现在真正生效(以前被静默丢弃)。
4. `httpclient.SetDefaultSetting` 参数从值类型改为 `*HttpSettings`。
5. `lifecycle.NewLifecycleManager()` 不再接收参数。

## example

[example/main.go](example/main.go) 包含完整可运行的示例:数据库 CRUD、跨表事务、日志钩子、配置热重载、健康检查等。

```shell
cd example && go run . -c config.toml
```
