# 数据库操作

## 概述

IGo 使用 XORM 作为 ORM 框架，支持多种数据库，包括 MySQL、PostgreSQL、SQLite 等。提供了简洁的 API 和强大的功能。

## 配置

### MySQL 配置

```toml
[mysql.igo]
max_idle = 10              # 最大空闲连接数
max_open = 20              # 最大打开连接数
is_debug = true            # 是否开启调试模式
data_source = "root:root@tcp(127.0.0.1:3306)/igo?charset=utf8mb4&parseTime=True&loc=Local"
```

### SQLite 配置

```toml
[sqlite.test]
data_source = "test.db"    # SQLite 数据库文件路径
```

## 基本使用

### 获取数据库实例

```go
import "github.com/aichy126/igo"

// 获取数据库实例
db := igo.App.DB.NewDBTable("igo", "users")
```

### 模型定义

```go
type User struct {
    Id        int64     `xorm:"pk autoincr" json:"id"`
    Name      string    `xorm:"varchar(100) notnull" json:"name"`
    Email     string    `xorm:"varchar(100) unique" json:"email"`
    Status    int       `xorm:"default 1" json:"status"`
    CreatedAt time.Time `xorm:"created" json:"created_at"`
    UpdatedAt time.Time `xorm:"updated" json:"updated_at"`
}

// 表名
func (User) TableName() string {
    return "users"
}
```

### 基本查询

```go
// 获取数据库实例
db := igo.App.DB.NewDBTable("igo", "users")

// 查询所有用户
var users []User
err := db.Find(&users)

// 条件查询
var activeUsers []User
err = db.Where("status = ?", 1).Find(&activeUsers)

// 分页查询
var pageUsers []User
err = db.Limit(10, 0).Find(&pageUsers) // limit 10, offset 0

// 排序查询
err = db.OrderBy("id desc").Find(&users)

// 统计查询
count, err := db.Count(&User{})
```

### 单条记录操作

```go
// 根据 ID 查询
var user User
has, err := db.ID(1).Get(&user)

// 根据条件查询
has, err = db.Where("email = ?", "user@example.com").Get(&user)

// 插入记录
user := &User{
    Name:  "张三",
    Email: "zhangsan@example.com",
    Status: 1,
}
affected, err := db.Insert(user)

// 更新记录
affected, err = db.ID(1).Update(&User{Name: "李四"})

// 删除记录
affected, err = db.ID(1).Delete(&User{})
```

### 复杂查询

```go
// 多条件查询
var users []User
err := db.Where("status = ? AND created_at > ?", 1, time.Now().AddDate(0, 0, -7)).Find(&users)

// IN 查询
err = db.In("id", []int64{1, 2, 3}).Find(&users)

// LIKE 查询
err = db.Where("name LIKE ?", "%张%").Find(&users)

// 关联查询
type UserWithProfile struct {
    User    `xorm:"extends"`
    Profile `xorm:"extends"`
}

var userProfiles []UserWithProfile
err = db.Table("users").Join("LEFT", "profiles", "users.id = profiles.user_id").Find(&userProfiles)
```

## 事务操作

### 基本事务

```go
// 获取数据库实例
db := igo.App.DB.NewDBTable("igo", "users")

// 开始事务
session := db.NewSession()
defer session.Close()

// 开始事务
err := session.Begin()
if err != nil {
    return err
}

// 执行操作
user := &User{Name: "张三", Email: "zhangsan@example.com"}
_, err = session.Insert(user)
if err != nil {
    session.Rollback()
    return err
}

// 提交事务
err = session.Commit()
if err != nil {
    return err
}
```

### 事务辅助函数

```go
// 使用事务执行函数
err := db.Transaction(func(session *xorm.Session) error {
    // 插入用户
    user := &User{Name: "张三", Email: "zhangsan@example.com"}
    _, err := session.Insert(user)
    if err != nil {
        return err
    }

    // 插入用户配置
    profile := &Profile{UserId: user.Id, Theme: "dark"}
    _, err = session.Insert(profile)
    if err != nil {
        return err
    }

    return nil
})
```

## 高级功能

### 批量操作

```go
// 批量插入
users := []User{
    {Name: "张三", Email: "zhangsan@example.com"},
    {Name: "李四", Email: "lisi@example.com"},
    {Name: "王五", Email: "wangwu@example.com"},
}
affected, err := db.Insert(&users)

// 批量更新
affected, err = db.In("id", []int64{1, 2, 3}).Update(&User{Status: 0})

// 批量删除
affected, err = db.In("id", []int64{1, 2, 3}).Delete(&User{})
```

### 原生 SQL

```go
// 执行原生 SQL
sql := "SELECT * FROM users WHERE status = ? AND created_at > ?"
var users []User
err := db.SQL(sql, 1, time.Now().AddDate(0, 0, -7)).Find(&users)

// 执行更新 SQL
sql = "UPDATE users SET status = ? WHERE id = ?"
result, err := db.Exec(sql, 0, 1)
```

### 统计查询

```go
// 聚合查询
type UserStats struct {
    TotalCount int64 `xorm:"total_count"`
    ActiveCount int64 `xorm:"active_count"`
}

var stats UserStats
err := db.SQL("SELECT COUNT(*) as total_count, SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) as active_count FROM users").Get(&stats)
```

## 连接池管理

### 连接池配置

```toml
[mysql.igo]
max_idle = 10              # 最大空闲连接数
max_open = 20              # 最大打开连接数
conn_max_lifetime = 3600   # 连接最大生命周期（秒）
```

### 连接池监控

```go
// 获取连接池统计信息
stats := db.DB().Stats()
log.Info("数据库连接池统计",
    log.Int("max_open_connections", stats.MaxOpenConnections),
    log.Int("open_connections", stats.OpenConnections),
    log.Int("in_use", stats.InUse),
    log.Int("idle", stats.Idle),
)
```

## 性能优化

### 索引优化

```go
// 创建索引
type User struct {
    Id        int64     `xorm:"pk autoincr" json:"id"`
    Name      string    `xorm:"varchar(100) notnull index" json:"name"`
    Email     string    `xorm:"varchar(100) unique" json:"email"`
    Status    int       `xorm:"default 1 index" json:"status"`
    CreatedAt time.Time `xorm:"created index" json:"created_at"`
    UpdatedAt time.Time `xorm:"updated" json:"updated_at"`
}
```

### 查询优化

```go
// 只查询需要的字段
type UserSummary struct {
    Id   int64  `xorm:"id"`
    Name string `xorm:"name"`
}

var summaries []UserSummary
err := db.Select("id, name").Find(&summaries)

// 使用索引的查询
err = db.Where("status = ? AND created_at > ?", 1, time.Now().AddDate(0, 0, -7)).Find(&users)
```

## 错误处理

### 常见错误处理

```go
// 查询错误处理
var user User
has, err := db.ID(1).Get(&user)
if err != nil {
    log.Error("查询用户失败", log.Any("error", err))
    return err
}
if !has {
    return errors.New("用户不存在")
}

// 插入错误处理
affected, err := db.Insert(&user)
if err != nil {
    if strings.Contains(err.Error(), "Duplicate entry") {
        return errors.New("用户已存在")
    }
    return err
}
```

### 自定义错误

```go
var (
    ErrUserNotFound = errors.New("用户不存在")
    ErrUserExists   = errors.New("用户已存在")
)

func GetUser(id int64) (*User, error) {
    var user User
    has, err := db.ID(id).Get(&user)
    if err != nil {
        return nil, err
    }
    if !has {
        return nil, ErrUserNotFound
    }
    return &user, nil
}
```

## 最佳实践

### 1. 模型设计

- 使用有意义的字段名
- 添加适当的标签和约束
- 考虑查询性能，添加必要的索引

### 2. 查询优化

- 只查询需要的字段
- 使用索引支持的查询条件
- 避免 N+1 查询问题

### 3. 事务管理

- 合理使用事务
- 及时提交或回滚事务
- 避免长事务

### 4. 错误处理

- 区分不同类型的错误
- 提供有意义的错误信息
- 记录详细的错误日志

### 5. 连接池管理

- 根据负载调整连接池大小
- 监控连接池使用情况
- 设置合理的连接超时时间
