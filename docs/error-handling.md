# IGo 脚手架错误处理规范

## 脚手架职责

IGo 作为脚手架，专注于：
- **组件初始化**: 安全地初始化各个组件（web、db、cache、config等）
- **组件协调**: 确保组件间的正确启动顺序和生命周期管理
- **错误传递**: 将组件错误正确传递给业务层，由业务层决定处理方式

**不干预**: 业务逻辑的错误处理、错误码定义、业务异常处理等应由业务层自行决定。

## 脚手架错误处理原则

### 1. 组件初始化错误处理

#### 关键组件（必须成功）
- **配置系统**: 配置加载失败返回错误，应用无法启动
- **Web服务**: Web服务初始化失败返回错误
- **日志系统**: 日志初始化失败返回错误

#### 可选组件（失败可容忍）
- **数据库**: 初始化失败记录警告，应用继续启动（业务可能不需要数据库）
- **缓存**: 初始化失败记录警告，应用继续启动（业务可能不需要缓存）

```go
// ✅ 脚手架正确做法：关键组件失败返回错误
func NewApp(configPath string) (*Application, error) {
    // 配置加载失败 -> 返回错误
    conf, err := config.NewConfig(configPath)
    if err != nil {
        return nil, fmt.Errorf("配置文件加载失败: %w", err)
    }

    // 可选组件失败 -> 记录警告，继续启动
    db, err := db.NewDb(conf)
    if err != nil {
        log.Warn("数据库初始化失败，如果不需要数据库功能请忽略此警告", log.Any("error", err))
        a.DB = &db.DB{} // 创建空实例避免nil
    } else {
        a.DB = db
    }
}
```

### 2. 避免不当panic

#### ❌ 脚手架中禁止panic的场景
```go
// 配置解析失败
if err != nil {
    panic(err) // ❌ 不要panic，返回错误
}

// 外部依赖连接失败  
consul, err := consulapi.NewClient(config)
if err != nil {
    panic(err) // ❌ 不要panic，返回错误
}

// 类型断言
addr := config.Get("address").(string) // ❌ 可能panic
```

#### ✅ 正确的错误处理
```go
// 返回错误让调用方处理
if err != nil {
    return fmt.Errorf("配置解析失败: %w", err)
}

// 安全的类型断言
addr, ok := config.Get("address").(string)
if !ok {
    return fmt.Errorf("address配置必须是字符串类型")
}
```

### 3. 生命周期错误处理

#### 启动阶段
- 关键组件失败：停止启动，返回错误
- 可选组件失败：记录警告，继续启动

#### 运行阶段  
- 记录组件运行时错误，不干预业务处理

#### 关闭阶段
- 尽力关闭所有组件，记录关闭失败但继续清理流程

```go
// ✅ 关闭阶段错误处理
func (lm *LifecycleManager) shutdown() {
    // 尽力关闭所有资源，不因单个失败而停止
    if err := lm.app.GetCache().Close(); err != nil {
        log.Error("关闭缓存失败", log.Any("error", err))
        // 继续关闭其他资源
    }
    
    if err := lm.app.GetDB().Close(); err != nil {
        log.Error("关闭数据库失败", log.Any("error", err))
        // 继续关闭其他资源
    }
}
```

### 4. 配置自动修复

对于可以自动修复的配置问题，记录警告并修复：

```go
// ✅ 配置自动修复
if !strings.Contains(address, ":") {
    log.Warn("Redis地址缺少端口号，自动添加默认端口6379", log.Any("address", address))
    address = address + ":6379"
}
```

## 已修复的问题

1. **igo.go**: 初始化panic改为返回错误
2. **config.go**: consul连接panic改为返回错误
3. **web/gin.go**: 危险类型断言改为安全版本
4. **db/config.go**: 危险类型断言改为安全版本，增加nil检查
5. **cache/redis_config.go**: 配置警告级别调整

## 业务层职责

业务层应自行处理：
- 业务逻辑错误处理
- 自定义错误类型和错误码
- API响应格式
- 错误监控和告警
- 重试和降级策略

脚手架只负责提供稳定的基础设施，错误处理的具体策略由业务层决定。