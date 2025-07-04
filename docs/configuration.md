# 配置管理

## 概述

IGo 使用 Viper 作为配置管理库，支持多种配置源和格式，包括本地文件、环境变量、配置中心等。

## 配置文件格式

IGo 支持 TOML 格式的配置文件，结构清晰，易于阅读和维护。

### 基本配置

```toml
[local]
address = ":8001"           # 服务监听地址
debug = true               # 调试模式
shutdown_timeout = 30      # 优雅关闭超时时间（秒）

[local.logger]
dir = "./logs"             # 日志目录
name = "log.log"           # 日志文件名
access = true              # 是否记录访问日志
level = "INFO"             # 日志级别
max_size = 1               # 单个日志文件最大大小（MB）
max_backups = 5            # 保留的备份文件数量
max_age = 7                # 日志文件保留天数
```

### 数据库配置

```toml
[mysql.igo]
max_idle = 10              # 最大空闲连接数
max_open = 20              # 最大打开连接数
is_debug = true            # 是否开启调试模式
data_source = "root:root@tcp(127.0.0.1:3306)/igo?charset=utf8mb4&parseTime=True&loc=Local"

[sqlite.test]
data_source = "test.db"    # SQLite 数据库文件路径
```

### Redis 配置

```toml
[redis.igorediskey]
address = "127.0.0.1:6379" # Redis 服务器地址
password = ""              # Redis 密码
db = 0                     # 数据库编号
poolsize = 50              # 连接池大小
```

## 配置源优先级

IGo 按以下优先级读取配置：

1. **命令行参数** - 最高优先级
2. **环境变量** - 次高优先级
3. **配置文件** - 默认优先级
4. **默认值** - 最低优先级

### 命令行参数

```bash
# 指定配置文件路径
go run main.go -c config.toml

# 指定配置键值
go run main.go --local.address=:8080 --local.debug=false
```

### 环境变量

```bash
# 指定配置文件路径
export CONFIG_PATH=./config.toml

# 指定配置中心
export CONFIG_ADDRESS=127.0.0.1:8500
export CONFIG_KEY=/igo/config

# 覆盖特定配置
export LOCAL_ADDRESS=:8080
export LOCAL_DEBUG=false
```

## 配置读取

### 使用 util 包

```go
import "github.com/aichy126/igo/util"

// 读取字符串配置
address := util.ConfGetString("local.address")

// 读取整数配置
port := util.ConfGetInt("local.port")

// 读取布尔配置
debug := util.ConfGetbool("local.debug")

// 读取浮点数配置
timeout := util.ConfGetFloat64("local.timeout")
```

### 直接使用 Viper

```go
import "github.com/aichy126/igo"

// 读取配置
address := igo.App.Conf.GetString("local.address")
port := igo.App.Conf.GetInt("local.port")
debug := igo.App.Conf.GetBool("local.debug")

// 读取带默认值的配置
timeout := igo.App.Conf.GetInt("local.timeout")
if timeout == 0 {
    timeout = 30 // 默认值
}
```

### 读取嵌套配置

```go
// 读取日志配置
logDir := util.ConfGetString("local.logger.dir")
logLevel := util.ConfGetString("local.logger.level")

// 读取数据库配置
dbMaxIdle := util.ConfGetInt("mysql.igo.max_idle")
dbMaxOpen := util.ConfGetInt("mysql.igo.max_open")
```

## 配置验证

### 基本验证

```go
func validateConfig() error {
    // 检查必需配置
    if util.ConfGetString("local.address") == "" {
        return errors.New("local.address is required")
    }

    // 检查配置范围
    port := util.ConfGetInt("local.port")
    if port < 1 || port > 65535 {
        return errors.New("local.port must be between 1 and 65535")
    }

    return nil
}
```

### 配置热重载

```go
// 监听配置文件变化
igo.App.Conf.WatchConfig()
igo.App.Conf.OnConfigChange(func(e fsnotify.Event) {
    log.Info("配置文件已更新", log.String("file", e.Name))
    // 重新加载配置
    reloadConfig()
})
```

## 配置中心支持

### Consul 配置中心

```toml
[config]
address = "127.0.0.1:8500"  # Consul 地址
key = "/igo/config"         # 配置键
```

### 环境变量配置

```bash
export CONFIG_ADDRESS=127.0.0.1:8500
export CONFIG_KEY=/igo/config
```

## 最佳实践

### 1. 配置文件组织

- 将不同环境的配置文件分开
- 使用有意义的配置键名
- 为配置项添加注释说明

### 2. 配置验证

- 在应用启动时验证必需配置
- 检查配置值的合理性
- 提供有意义的错误信息

### 3. 环境变量

- 敏感信息使用环境变量
- 使用标准的环境变量命名
- 提供环境变量文档

### 4. 配置热重载

- 只重载支持热更新的配置
- 记录配置变更日志
- 处理配置重载错误

## 示例配置

### 开发环境

```toml
[local]
address = ":8001"
debug = true
shutdown_timeout = 30

[local.logger]
dir = "./logs"
name = "dev.log"
access = true
level = "DEBUG"
max_size = 1
max_backups = 3
max_age = 3

[mysql.igo]
max_idle = 5
max_open = 10
is_debug = true
data_source = "root:root@tcp(127.0.0.1:3306)/igo_dev?charset=utf8mb4&parseTime=True&loc=Local"

[redis.igorediskey]
address = "127.0.0.1:6379"
password = ""
db = 0
poolsize = 20
```

### 生产环境

```toml
[local]
address = ":8080"
debug = false
shutdown_timeout = 60

[local.logger]
dir = "/var/log/igo"
name = "app.log"
access = true
level = "INFO"
max_size = 100
max_backups = 10
max_age = 30

[mysql.igo]
max_idle = 20
max_open = 100
is_debug = false
data_source = "user:pass@tcp(db.example.com:3306)/igo_prod?charset=utf8mb4&parseTime=True&loc=Local"

[redis.igorediskey]
address = "redis.example.com:6379"
password = "your_redis_password"
db = 0
poolsize = 100
```
