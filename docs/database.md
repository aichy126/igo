# 数据库使用指南

## 概述

IGo 使用 xorm 作为 ORM 框架，提供了便捷的数据库操作封装。本指南介绍如何使用 IGo 进行数据库操作。

## 配置

在 `config.toml` 中配置数据库连接：

```toml
[mysql.test]
data_source = "user:password@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
max_idle = 10
max_open = 100
max_idle_life = 3600
is_debug = false

[sqlite.local]
data_source = "file:test.db?cache=shared&mode=rwc"
max_idle = 5
max_open = 10
is_debug = true
```

## 基本使用

### 1. 单表操作（传统方式）

```go
// 创建 Repo
repo := igo.App.DB.NewDBTable("test", "users")

// 插入
user := &User{Name: "张三", Age: 25}
repo.InsertOne(user)

// 查询
var user User
repo.Where("id = ?", 1).Get(&user)

// 查询多条
var users []User
repo.Where("age > ?", 18).Find(&users)

// 更新
repo.Where("id = ?", 1).Update(&User{Name: "李四"})

// 删除
repo.Where("id = ?", 1).Delete(&User{})
```

### 2. 跨表操作（新功能）

**问题**: 传统的 Repo 方式无法在同一个事务中操作多个表。

**解决方案**: 使用 `NewSession()` 或 `BeginTx()` 方法。

#### 方式1：使用 NewSession

```go
// 获取原始 Session（未绑定表）
sess := igo.App.DB.NewSession("test")
defer sess.Close()

// 手动开启事务
sess.Begin()

// 操作多个表
sess.Table("users").Insert(&user)
sess.Table("orders").Insert(&order)
sess.Table("order_items").Insert(&items)

// 提交事务
if err := sess.Commit(); err != nil {
    sess.Rollback()
    return err
}
```

#### 方式2：使用 BeginTx（推荐）

```go
// 自动开启事务
sess, err := igo.App.DB.BeginTx("test")
if err != nil {
    return err
}
defer sess.Close()

// 操作多个表
if _, err := sess.Table("users").Insert(&user); err != nil {
    sess.Rollback()
    return err
}

if _, err := sess.Table("orders").Insert(&order); err != nil {
    sess.Rollback()
    return err
}

// 提交事务
if err := sess.Commit(); err != nil {
    return err
}
```

## 完整示例

### 订单创建（跨表事务）

```go
package service

import (
    "fmt"
    "github.com/aichy126/igo"
)

type Order struct {
    ID     int64  `xorm:"pk autoincr"`
    UserID int64
    Amount float64
}

type OrderItem struct {
    ID       int64 `xorm:"pk autoincr"`
    OrderID  int64
    Product  string
    Quantity int
    Price    float64
}

// CreateOrder 创建订单（涉及多表操作）
func CreateOrder(userID int64, items []OrderItem) error {
    // 开启事务
    sess, err := igo.App.DB.BeginTx("test")
    if err != nil {
        return fmt.Errorf("开启事务失败: %w", err)
    }
    defer sess.Close()

    // 1. 计算总金额
    var totalAmount float64
    for _, item := range items {
        totalAmount += item.Price * float64(item.Quantity)
    }

    // 2. 创建订单
    order := &Order{
        UserID: userID,
        Amount: totalAmount,
    }
    if _, err := sess.Table("orders").Insert(order); err != nil {
        sess.Rollback()
        return fmt.Errorf("创建订单失败: %w", err)
    }

    // 3. 创建订单项
    for i := range items {
        items[i].OrderID = order.ID
    }
    if _, err := sess.Table("order_items").Insert(&items); err != nil {
        sess.Rollback()
        return fmt.Errorf("创建订单项失败: %w", err)
    }

    // 4. 更新用户订单数
    _, err = sess.Exec("UPDATE users SET order_count = order_count + 1 WHERE id = ?", userID)
    if err != nil {
        sess.Rollback()
        return fmt.Errorf("更新用户订单数失败: %w", err)
    }

    // 5. 提交事务
    if err := sess.Commit(); err != nil {
        return fmt.Errorf("提交事务失败: %w", err)
    }

    return nil
}
```

### 批量数据同步

```go
// SyncData 从一个表同步数据到另一个表
func SyncData() error {
    sess := igo.App.DB.NewSession("test")
    defer sess.Close()

    sess.Begin()

    // 从源表查询
    var sourceData []SourceModel
    err := sess.Table("source_table").
        Where("status = ?", "pending").
        Find(&sourceData)
    if err != nil {
        sess.Rollback()
        return err
    }

    // 转换并插入目标表
    var targetData []TargetModel
    for _, src := range sourceData {
        targetData = append(targetData, convertToTarget(src))
    }

    if _, err := sess.Table("target_table").Insert(&targetData); err != nil {
        sess.Rollback()
        return err
    }

    // 更新源表状态
    _, err = sess.Table("source_table").
        In("id", getIDs(sourceData)).
        Update(map[string]interface{}{"status": "synced"})
    if err != nil {
        sess.Rollback()
        return err
    }

    return sess.Commit()
}
```

## 与传统 Repo 方式的对比

### 传统方式的局限性

```go
// ❌ 无法在同一事务中操作多表
userRepo := igo.App.DB.NewDBTable("test", "users")
orderRepo := igo.App.DB.NewDBTable("test", "orders")

// 这两个操作在不同的事务中，不是原子性的
userRepo.Insert(&user)   // 事务1
orderRepo.Insert(&order) // 事务2
```

### 新方式的优势

```go
// ✅ 同一事务中操作多表
sess, _ := igo.App.DB.BeginTx("test")
defer sess.Close()

sess.Table("users").Insert(&user)
sess.Table("orders").Insert(&order)
sess.Commit() // 原子性提交
```

## 最佳实践

### 1. 优先使用 Repo（单表操作）

```go
// 单表操作用 Repo，代码更简洁
repo := igo.App.DB.NewDBTable("test", "users")
repo.Where("id = ?", 1).Get(&user)
```

### 2. 跨表用 Session/事务

```go
// 跨表操作用 Session
sess, _ := igo.App.DB.BeginTx("test")
defer sess.Close()

sess.Table("users").Insert(&user)
sess.Table("orders").Insert(&order)
sess.Commit()
```

### 3. 错误处理

```go
sess, err := igo.App.DB.BeginTx("test")
if err != nil {
    return err
}
defer sess.Close()

// 使用 defer 确保回滚
success := false
defer func() {
    if !success {
        sess.Rollback()
    }
}()

// 业务逻辑
if _, err := sess.Table("users").Insert(&user); err != nil {
    return err
}

if err := sess.Commit(); err != nil {
    return err
}

success = true
return nil
```

### 4. 上下文传递

```go
// 在业务层传递 traceId
func CreateOrder(ctx ictx.Context, order *Order) error {
    sess, _ := igo.App.DB.BeginTx("test")
    defer sess.Close()

    // 记录日志
    ctx.LogInfo("开始创建订单", log.Any("order", order))

    // 数据库操作
    if _, err := sess.Table("orders").Insert(order); err != nil {
        ctx.LogError("创建订单失败", log.Any("error", err))
        sess.Rollback()
        return err
    }

    ctx.LogInfo("订单创建成功", log.Any("orderId", order.ID))
    return sess.Commit()
}
```

## 注意事项

1. **及时关闭 Session**: 使用 `defer sess.Close()` 确保资源释放
2. **错误回滚**: 操作失败时务必调用 `sess.Rollback()`
3. **避免长事务**: 事务时间过长会锁表，影响性能
4. **幂等性**: 考虑事务失败重试的场景，确保操作幂等

## 常见问题

### Q: 什么时候使用 Repo，什么时候使用 Session？

**A**:
- 单表 CRUD 操作 → 使用 Repo
- 跨表事务操作 → 使用 Session/BeginTx
- 需要精确控制事务边界 → 使用 Session/BeginTx

### Q: BeginTx 和 NewSession 有什么区别？

**A**:
- `BeginTx`: 自动开启事务，推荐使用
- `NewSession`: 返回原始 Session，需要手动调用 `Begin()`

### Q: 可以混用 Repo 和 Session 吗？

**A**: 可以，但要注意它们不在同一事务中。如果需要事务一致性，应该只使用 Session。

```go
// ❌ 错误：不在同一事务
sess, _ := igo.App.DB.BeginTx("test")
sess.Table("orders").Insert(&order)

repo := igo.App.DB.NewDBTable("test", "order_items")
repo.Insert(&items) // 这个操作不在 sess 的事务中

sess.Commit()
```

```go
// ✅ 正确：都在同一事务
sess, _ := igo.App.DB.BeginTx("test")
sess.Table("orders").Insert(&order)
sess.Table("order_items").Insert(&items)
sess.Commit()
```
