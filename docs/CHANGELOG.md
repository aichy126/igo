# 更新日志

## v0.2.0 (2025-01-12)

本次更新在 v0.1.1 基础上进行了兼容性升级，主要新增以下功能：

### 🎉 新功能

#### 1. 生命周期管理器
- 新增 `lifecycle` 包，提供应用生命周期管理
- 支持 `AddStartupHook()` 和 `AddShutdownHook()` 方法
- 自动处理优雅关闭流程
- 新增 `RunWithGracefulShutdown()` 方法简化启动

**使用示例：**
```go
app, _ := igo.NewApp("")

// 添加启动钩子
app.AddStartupHook(func() error {
    log.Info("应用启动完成")
    return nil
})

// 添加关闭钩子
app.AddShutdownHook(func() error {
    log.Info("执行自定义清理")
    return nil
})

// 运行应用（自动处理优雅关闭）
app.RunWithGracefulShutdown()
```

#### 2. 配置热重载
- 文件配置：自动使用 fsnotify 监听（IO影响极小）
- Consul配置：支持用户自定义轮询间隔
- 配置变更回调机制
- 线程安全的配置读写

**使用示例：**
```go
// 添加配置变更回调
app.AddConfigChangeCallback(func() {
    log.Info("配置已更新")
})

// 设置Consul配置热重载间隔（文件配置自动启用）
app.SetConfigHotReloadInterval(30) // 30秒轮询一次

// 手动重载配置
app.ReloadConfig()
```

#### 3. Log 钩子系统
- 支持自定义日志钩子，可根据日志级别触发
- 异步执行，不阻塞日志记录
- 适用于飞书、企业微信等第三方通知

**使用示例：**
```go
// 实现钩子接口
type FeishuHook struct {}

func (h *FeishuHook) Levels() []zapcore.Level {
    return []zapcore.Level{zapcore.ErrorLevel, zapcore.FatalLevel}
}

func (h *FeishuHook) Fire(entry *log.LogEntry) error {
    // 发送错误到飞书
    return sendToFeishu(entry.Message, entry.Fields)
}

// 注册钩子
log.AddHook(&FeishuHook{})
```

#### 4. Xorm 跨表 Session 支持
- 新增 `NewSession(dbname)` 方法获取原始 Session（未绑定表）
- 新增 `BeginTx(dbname)` 方法启动跨表事务
- 完全兼容现有 Repo API

**使用示例：**
```go
// 方式1：使用 NewSession（需手动开启事务）
sess := igo.App.DB.NewSession("test")
defer sess.Close()
sess.Begin()
sess.Table("users").Insert(&user)
sess.Table("orders").Insert(&order)
sess.Commit()

// 方式2：使用 BeginTx（自动开启事务）
sess, err := igo.App.DB.BeginTx("test")
if err != nil {
    return err
}
defer sess.Close()
sess.Table("users").Insert(&user)
sess.Table("orders").Insert(&order)
sess.Commit()
```

#### 5. Web 服务优化
- 新增 `Web.Shutdown(ctx)` 方法支持优雅关闭
- 新增 `Run()` 方法返回 error
- 自动注册 Web 服务器关闭钩子

**说明**：
中间件不作为框架功能提供，业务层可参考 `example/middleware` 中的示例自行实现。
IGo 保持简洁，只提供核心脚手架功能，不干预业务逻辑。

### 🔧 改进

#### 错误处理规范化
- `NewApp()` 现在返回 `(*Application, error)` 而不是 panic
- 数据库/缓存作为可选组件，失败时静默处理（创建空实例避免 nil 指针）
- 配置验证机制
- 安全的类型断言

**迁移指南：**
```go
// 旧版本
igo.App = igo.NewApp("")

// 新版本
app, err := igo.NewApp("")
if err != nil {
    log.Error("应用初始化失败", log.Any("error", err))
    os.Exit(1)
}
igo.App = app
```

#### 资源管理优化
- 新增 `DB.Close()` 方法关闭所有数据库连接
- 新增 `Cache.Close()` 方法关闭所有Redis连接
- 新增 `Web.Shutdown(ctx)` 方法优雅关闭Web服务器
- 生命周期管理器自动注册资源关闭钩子

### 📚 配置新增方法
- `Validate()` - 验证配置
- `IsHotReloadEnabled()` - 检查是否启用热重载
- `WatchConfig()` - 启动配置监听
- `AddChangeCallback(callback)` - 添加配置变更回调
- `ReloadConfig()` - 手动重载配置
- `SetHotReloadInterval(seconds)` - 设置热重载间隔
- `GetWithDefault(key, default)` - 获取配置值（带默认值）

### ⚠️ 破坏性变更
无。所有改动保持向后兼容，仅 `NewApp()` 的返回值发生变化。

### 🎯 升级建议
1. 修改 `NewApp()` 调用以处理错误返回值
2. 将 `app.Web.Run()` 改为 `app.RunWithGracefulShutdown()` 以使用优雅关闭
3. 根据需要添加启动/关闭钩子和配置变更回调
4. 对于跨表操作，使用新的 `NewSession()` 或 `BeginTx()` 方法
5. 根据需要实现和注册日志钩子

### 🙏 致谢
感谢社区反馈，本次更新主要解决了：
- 跨表事务操作不便的问题
- 缺少日志通知机制的问题
- 错误处理不规范的问题
- 缺少生命周期管理的问题
