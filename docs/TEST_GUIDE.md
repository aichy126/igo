# IGo 新功能测试指南

本指南介绍如何测试 IGo v0.2.0 的所有新功能。

## 快速开始

### 1. 启动示例应用

在第一个终端运行：

```bash
cd example
go run main.go
```

你应该看到类似的输出：

```
文件配置自动启用热重载，忽略轮询间隔设置
日志钩子已注册 {"hook": "MockFeishuHook"}
执行启动钩子 {"index": 0}
应用启动完成
表结构同步成功
Web服务器启动 {"address": ":8001"}
```

### 2. 运行测试脚本

在第二个终端运行：

```bash
bash test_new_features.sh
```

测试脚本会自动测试所有新功能。

## 手动测试

如果你想手动测试各个功能，可以使用以下 curl 命令：

### 1. 基础功能测试

```bash
# 健康检查
curl http://localhost:8001/health

# Ping测试（测试TraceId）
curl http://localhost:8001/ping
```

### 2. 日志钩子测试

```bash
# 触发日志钩子
curl http://localhost:8001/test/log-hook
```

**预期结果：**
- 服务端日志中应该看到 `📱 [模拟飞书通知]` 的输出
- 只有 Error 和 Fatal 级别的日志会触发钩子

**查看效果：**
```
📱 [模拟飞书通知] [ERROR] 测试错误日志 - 这应该触发钩子 (TraceID: xxx)
📱 [模拟飞书通知] [ERROR] 模拟业务错误：数据库连接失败 (TraceID: xxx)
```

### 3. 配置热重载测试

```bash
# 手动重载配置
curl -X POST http://localhost:8001/config/reload
```

**自动热重载测试：**
1. 修改 `example/config.toml` 文件
2. 观察服务端日志，应该看到 "配置已更新" 的输出

### 4. 跨表事务测试 - 创建订单

```bash
# 创建订单（演示跨表事务）
curl -X POST http://localhost:8001/order/create \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "items": [
      {
        "product": "iPhone 15",
        "quantity": 1,
        "price": 5999.00
      },
      {
        "product": "AirPods Pro",
        "quantity": 2,
        "price": 1999.00
      }
    ]
  }'
```

**预期结果：**
```json
{
  "code": 0,
  "message": "订单创建成功",
  "data": {
    "ID": 1,
    "UserID": "user123",
    "Amount": 9997,
    "Status": "pending",
    "CreatedAt": "2025-01-12T10:30:00Z"
  }
}
```

**服务端日志应显示：**
```
开始创建订单 {"userID": "user123", "itemCount": 2}
订单创建成功 {"orderID": 1}
订单项创建成功 {"count": 2}
订单创建完成 {"orderID": 1, "amount": 9997}
```

### 5. 跨表事务测试 - 批量同步

```bash
# 批量数据同步（跨表操作）
curl -X POST http://localhost:8001/db/batch-sync
```

**功能说明：**
- 从 `test0` 表查询数据
- 转换后插入 `test2` 表
- 整个过程在一个事务中完成

### 6. 中间件测试

```bash
# 测试中间件功能
curl http://localhost:8001/middleware/test \
  -H "X-Custom-Header: test"
```

**预期结果：**
```json
{
  "code": 0,
  "message": "中间件测试",
  "data": {
    "trace_id": "xxx-xxx-xxx",
    "headers": {
      "X-Custom-Header": ["test"],
      ...
    }
  }
}
```

## 测试优雅关闭

### 测试步骤

1. 启动应用后，按 `Ctrl+C`
2. 观察日志输出

**预期日志：**
```
收到关闭信号，开始执行关闭钩子...
执行关闭钩子 {"index": 3}
应用准备关闭，执行清理工作...
执行关闭钩子 {"index": 2}
正在关闭Web服务器...
执行关闭钩子 {"index": 1}
正在关闭缓存连接...
执行关闭钩子 {"index": 0}
正在关闭数据库连接...
应用已优雅关闭
```

**说明：**
- 关闭钩子按注册顺序的**逆序**执行
- 先关闭 Web 服务器（停止接收新请求）
- 再关闭缓存连接
- 最后关闭数据库连接

## 测试真实的飞书钩子

如果你想测试真实的飞书通知，按以下步骤：

### 1. 获取飞书机器人 Webhook URL

1. 在飞书群组中添加自定义机器人
2. 获取 Webhook URL（格式：`https://open.feishu.cn/open-apis/bot/v2/hook/xxx`）

### 2. 修改 main.go

取消注释并配置飞书钩子：

```go
// 注册真实的飞书钩子
feishuHook := &hooks.FeishuHook{
    WebhookURL: "你的Webhook URL",
    AppName:    "IGo测试应用",
    Enabled:    true,
}
log.AddHook(feishuHook)
```

### 3. 触发测试

```bash
curl http://localhost:8001/test/log-hook
```

你应该在飞书群组中收到错误告警消息。

## 常见问题

### Q1: 服务启动失败

**可能原因：**
- 端口 8001 已被占用
- 配置文件不存在或格式错误
- 数据库连接失败

**解决方法：**
```bash
# 检查端口占用
lsof -i :8001

# 查看配置文件
cat example/config.toml

# 检查日志输出
cd example && go run main.go 2>&1 | tee app.log
```

### Q2: 日志钩子没有触发

**检查清单：**
- [ ] 钩子是否正确注册？
- [ ] 日志级别是否匹配？（只有 Error/Fatal 会触发）
- [ ] 查看服务端日志是否有错误

### Q3: 跨表事务失败

**可能原因：**
- 数据库表不存在
- 数据类型不匹配
- 事务冲突

**解决方法：**
1. 检查表是否创建：`SyncTables()` 在启动时会自动同步
2. 查看详细的错误日志
3. 检查数据库连接配置

## 性能测试

### 压力测试

```bash
# 使用 ab 进行压力测试
ab -n 1000 -c 10 http://localhost:8001/ping

# 使用 wrk 进行压力测试
wrk -t 4 -c 100 -d 30s http://localhost:8001/ping
```

### 观察指标

在压力测试期间观察：
- 响应时间
- 错误率
- CPU 和内存使用
- 日志钩子是否影响性能

## 下一步

- 查看 [CHANGELOG.md](CHANGELOG.md) 了解所有变更
- 查看 [log-hooks.md](log-hooks.md) 学习如何实现自定义钩子
- 查看 [database.md](database.md) 深入了解跨表事务

## 反馈

如果你发现问题或有改进建议，欢迎提交 Issue 或 PR！
