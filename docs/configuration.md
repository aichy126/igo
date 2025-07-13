# 配置管理

## 概述

IGo使用Viper作为配置管理库，支持多种配置源和热重载功能。

## 支持的配置源

### 1. 本地文件 (推荐)
- **格式**: TOML
- **热重载**: 自动启用，基于fsnotify实时监听
- **性能**: 几乎无IO开销

### 2. Consul配置中心
- **热重载**: 用户配置轮询间隔
- **性能**: 可控的网络请求频率
- **适用**: 生产环境集中配置管理

### 3. 环境变量
- **热重载**: 不支持（需重启应用）
- **适用**: 容器化部署

## 配置文件格式

### 基本配置
```toml
[local]
address = ":8001"           # 服务监听地址
debug = true               # 调试模式（启用pprof）

[local.logger]
dir = "./logs"             # 日志目录
name = "log.log"           # 日志文件名
level = "INFO"             # 日志级别
access = true              # 是否记录访问日志
max_size = 1               # 单个日志文件最大大小（MB）
max_backups = 5            # 保留的备份文件数量
max_age = 7                # 日志文件保留天数
```

### 数据库配置 (可选)
```toml
[mysql.igo]
max_idle = 10
max_open = 20
data_source = "root:password@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4"

[mysql.other_db]
max_idle = 5
max_open = 10
data_source = "root:password@tcp(127.0.0.1:3306)/other?charset=utf8mb4"
```

### 缓存配置 (可选)
```toml
[redis.default]
address = "127.0.0.1:6379"
password = ""
db = 0
pool_size = 10

[redis.session]
address = "127.0.0.1:6379"
password = ""
db = 1
pool_size = 5
```

### Consul配置
```toml
[config]
address = "127.0.0.1:8500"
key = "igo/config"
```

## 配置热重载

### 文件配置热重载
```go
// 文件配置自动启用热重载，无需额外配置
app, err := igo.NewApp("")

// 添加配置变更回调
app.AddConfigChangeCallback(func() {
    // 配置文件变更时执行
    debug := app.Conf.GetBool("local.debug")
    log.Info("配置已更新", log.Bool("debug", debug))
})
```

### Consul配置热重载
```go
// 设置Consul配置轮询间隔
app.SetConfigHotReloadInterval(60) // 60秒轮询一次

// 禁用Consul热重载
app.SetConfigHotReloadInterval(0)  // 或 -1
```

### 手动重载配置
```go
// 适用于所有配置源
if err := app.ReloadConfig(); err != nil {
    log.Error("配置重载失败", log.String("error", err.Error()))
}
```

## 配置读取

### 基本用法
```go
// 获取配置值
debug := igo.App.Conf.GetBool("local.debug")
address := igo.App.Conf.GetString("local.address")
maxIdle := igo.App.Conf.GetInt("mysql.igo.max_idle")

// 带默认值获取
timeout := igo.App.Conf.GetIntWithDefault("timeout", 30)
name := igo.App.Conf.GetStringWithDefault("app.name", "igo")
```

### 检查配置项是否存在
```go
if igo.App.Conf.IsSet("mysql.igo") {
    // 数据库配置存在
    db := igo.App.DB.NewDBTable("igo", "users")
}
```

## 性能影响

### 不同配置源的性能特点
- **文件监听**: 基于操作系统inotify机制，几乎无IO开销
- **Consul轮询**: 用户自定义间隔，推荐30-300秒
- **手动重载**: 仅在API调用时执行，无持续开销

### 使用建议
- **开发环境**: 文件配置，自动热重载
- **测试环境**: Consul配置，短间隔轮询（30-60秒）
- **生产环境**: Consul配置，长间隔轮询（300秒）或禁用热重载

## 配置验证

IGo只验证最基本的必需配置项：

```go
// 必需配置项
if !conf.IsSet("local.address") {
    return fmt.Errorf("缺少必要的配置项: local.address")
}
```

业务相关的配置验证由业务层自行实现。

## 环境变量

支持通过环境变量指定配置文件路径和Consul配置：

```bash
# 配置文件路径
export CONFIG_PATH="/path/to/config.toml"

# Consul配置
export CONFIG_ADDRESS="127.0.0.1:8500"
export CONFIG_KEY="igo/config"
```

## 最佳实践

1. **配置分层**: 区分基础配置和业务配置
2. **默认值**: 为可选配置提供合理默认值
3. **环境隔离**: 不同环境使用不同配置文件
4. **热重载谨慎使用**: 生产环境建议较长轮询间隔或禁用
5. **配置变更回调**: 在回调中重新初始化相关组件