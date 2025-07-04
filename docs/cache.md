# 缓存操作

## 概述

IGo 使用 go-redis 作为 Redis 客户端，提供了简洁的 API 和强大的功能，支持连接池、集群、哨兵等模式。

## 配置

### 基本配置

```toml
[redis.igorediskey]
address = "127.0.0.1:6379"  # Redis 服务器地址
password = ""               # Redis 密码
db = 0                      # 数据库编号
poolsize = 50               # 连接池大小
```

### 集群配置

```toml
[redis.cluster]
addresses = ["127.0.0.1:7000", "127.0.0.1:7001", "127.0.0.1:7002"]
password = ""
poolsize = 50
```

### 哨兵配置

```toml
[redis.sentinel]
master_name = "mymaster"
addresses = ["127.0.0.1:26379", "127.0.0.1:26380"]
password = ""
sentinel_password = ""
poolsize = 50
```

## 基本使用

### 获取 Redis 客户端

```go
import "github.com/aichy126/igo"

// 获取 Redis 客户端
redis, err := igo.App.Cache.Get("igorediskey")
if err != nil {
    return err
}
```

### 基本操作

```go
// 设置值
err := redis.Set(ctx, "key", "value", time.Hour).Err()

// 获取值
value, err := redis.Get(ctx, "key").Result()

// 删除值
err = redis.Del(ctx, "key").Err()

// 检查键是否存在
exists, err := redis.Exists(ctx, "key").Result()

// 设置过期时间
err = redis.Expire(ctx, "key", time.Hour).Err()

// 获取剩余过期时间
ttl, err := redis.TTL(ctx, "key").Result()
```

### 字符串操作

```go
// 设置字符串
err := redis.Set(ctx, "name", "张三", 0).Err()

// 获取字符串
name, err := redis.Get(ctx, "name").Result()

// 设置带过期时间的字符串
err = redis.SetEX(ctx, "temp_key", "temp_value", time.Minute).Err()

// 设置不存在的键
err = redis.SetNX(ctx, "unique_key", "value", time.Hour).Err()

// 批量设置
err = redis.MSet(ctx, "key1", "value1", "key2", "value2").Err()

// 批量获取
values, err := redis.MGet(ctx, "key1", "key2").Result()

// 递增
count, err := redis.Incr(ctx, "counter").Result()

// 递减
count, err = redis.Decr(ctx, "counter").Result()

// 递增指定值
count, err = redis.IncrBy(ctx, "counter", 10).Result()
```

### 哈希操作

```go
// 设置哈希字段
err := redis.HSet(ctx, "user:1", "name", "张三", "age", "25").Err()

// 获取哈希字段
name, err := redis.HGet(ctx, "user:1", "name").Result()

// 获取所有哈希字段
userData, err := redis.HGetAll(ctx, "user:1").Result()

// 检查哈希字段是否存在
exists, err := redis.HExists(ctx, "user:1", "name").Result()

// 删除哈希字段
err = redis.HDel(ctx, "user:1", "age").Err()

// 获取哈希字段数量
count, err := redis.HLen(ctx, "user:1").Result()

// 获取所有字段名
fields, err := redis.HKeys(ctx, "user:1").Result()

// 获取所有字段值
values, err := redis.HVals(ctx, "user:1").Result()

// 递增哈希字段
age, err := redis.HIncrBy(ctx, "user:1", "age", 1).Result()
```

### 列表操作

```go
// 从左侧推入元素
err := redis.LPush(ctx, "list", "item1", "item2").Err()

// 从右侧推入元素
err = redis.RPush(ctx, "list", "item3", "item4").Err()

// 从左侧弹出元素
item, err := redis.LPop(ctx, "list").Result()

// 从右侧弹出元素
item, err = redis.RPop(ctx, "list").Result()

// 获取列表长度
length, err := redis.LLen(ctx, "list").Result()

// 获取列表元素
items, err := redis.LRange(ctx, "list", 0, -1).Result()

// 根据索引获取元素
item, err = redis.LIndex(ctx, "list", 0).Result()

// 设置元素值
err = redis.LSet(ctx, "list", 0, "new_item").Err()

// 删除元素
err = redis.LRem(ctx, "list", 0, "item1").Err()
```

### 集合操作

```go
// 添加集合元素
err := redis.SAdd(ctx, "set", "member1", "member2").Err()

// 移除集合元素
err = redis.SRem(ctx, "set", "member1").Err()

// 检查元素是否存在
exists, err := redis.SIsMember(ctx, "set", "member1").Result()

// 获取集合大小
size, err := redis.SCard(ctx, "set").Result()

// 获取所有元素
members, err := redis.SMembers(ctx, "set").Result()

// 随机获取元素
member, err := redis.SRandMember(ctx, "set").Result()

// 弹出元素
member, err = redis.SPop(ctx, "set").Result()

// 集合运算
// 交集
intersection, err := redis.SInter(ctx, "set1", "set2").Result()

// 并集
union, err := redis.SUnion(ctx, "set1", "set2").Result()

// 差集
difference, err := redis.SDiff(ctx, "set1", "set2").Result()
```

### 有序集合操作

```go
// 添加有序集合元素
err := redis.ZAdd(ctx, "zset", &redis.Z{Score: 1.0, Member: "member1"}).Err()

// 获取元素分数
score, err := redis.ZScore(ctx, "zset", "member1").Result()

// 获取元素排名
rank, err := redis.ZRank(ctx, "zset", "member1").Result()

// 获取元素数量
count, err := redis.ZCard(ctx, "zset").Result()

// 获取指定范围的元素
members, err := redis.ZRange(ctx, "zset", 0, -1).Result()

// 获取指定范围的元素和分数
membersWithScores, err := redis.ZRangeWithScores(ctx, "zset", 0, -1).Result()

// 移除元素
err = redis.ZRem(ctx, "zset", "member1").Err()
```

## 高级功能

### 管道操作

```go
// 使用管道批量执行命令
pipe := redis.Pipeline()

pipe.Set(ctx, "key1", "value1", time.Hour)
pipe.Set(ctx, "key2", "value2", time.Hour)
pipe.Get(ctx, "key1")

// 执行管道
cmds, err := pipe.Exec(ctx)
if err != nil {
    return err
}

// 获取结果
for _, cmd := range cmds {
    if cmd.Err() != nil {
        log.Error("管道命令执行失败", log.Any("error", cmd.Err()))
    }
}
```

### 事务操作

```go
// 开始事务
tx := redis.TxPipeline()

tx.Set(ctx, "key1", "value1", time.Hour)
tx.Set(ctx, "key2", "value2", time.Hour)
tx.Get(ctx, "key1")

// 执行事务
cmds, err := tx.Exec(ctx)
if err != nil {
    return err
}

// 处理结果
for _, cmd := range cmds {
    if cmd.Err() != nil {
        log.Error("事务命令执行失败", log.Any("error", cmd.Err()))
    }
}
```

### 发布订阅

```go
// 订阅频道
pubsub := redis.Subscribe(ctx, "channel1", "channel2")
defer pubsub.Close()

// 接收消息
ch := pubsub.Channel()
for msg := range ch {
    log.Info("收到消息",
        log.String("channel", msg.Channel),
        log.String("payload", msg.Payload),
    )
}

// 发布消息
err := redis.Publish(ctx, "channel1", "hello").Err()
```

### 键空间通知

```go
// 订阅键空间通知
pubsub := redis.Subscribe(ctx, "__keyspace@0__:mykey")
defer pubsub.Close()

ch := pubsub.Channel()
for msg := range ch {
    log.Info("键空间通知",
        log.String("channel", msg.Channel),
        log.String("payload", msg.Payload),
    )
}
```

## 连接池管理

### 连接池配置

```toml
[redis.igorediskey]
address = "127.0.0.1:6379"
password = ""
db = 0
poolsize = 50               # 连接池大小
min_idle_conns = 10         # 最小空闲连接数
max_retries = 3             # 最大重试次数
dial_timeout = 5            # 连接超时时间（秒）
read_timeout = 3            # 读取超时时间（秒）
write_timeout = 3           # 写入超时时间（秒）
pool_timeout = 4            # 连接池超时时间（秒）
idle_timeout = 5            # 空闲连接超时时间（秒）
```

### 连接池监控

```go
// 获取连接池统计信息
stats := redis.PoolStats()
log.Info("Redis 连接池统计",
    log.Int("total_connections", stats.TotalConnections),
    log.Int("idle_connections", stats.IdleConnections),
    log.Int("stale_connections", stats.StaleConnections),
)
```

## 错误处理

### 常见错误处理

```go
// 连接错误
redis, err := igo.App.Cache.Get("igorediskey")
if err != nil {
    log.Error("获取 Redis 客户端失败", log.Any("error", err))
    return err
}

// 键不存在错误
value, err := redis.Get(ctx, "key").Result()
if err != nil {
    if err == redis.Nil {
        // 键不存在
        return errors.New("键不存在")
    }
    return err
}

// 网络错误
err = redis.Set(ctx, "key", "value", time.Hour).Err()
if err != nil {
    if strings.Contains(err.Error(), "connection refused") {
        log.Error("Redis 连接被拒绝")
        return err
    }
    return err
}
```

### 重试机制

```go
// 带重试的操作
func GetWithRetry(redis *redis.Client, key string) (string, error) {
    var value string
    var err error

    for i := 0; i < 3; i++ {
        value, err = redis.Get(ctx, key).Result()
        if err == nil {
            return value, nil
        }

        if err == redis.Nil {
            return "", err
        }

        // 网络错误，等待后重试
        time.Sleep(time.Duration(i+1) * time.Second)
    }

    return "", err
}
```

## 性能优化

### 批量操作

```go
// 批量设置
pairs := []interface{}{
    "key1", "value1",
    "key2", "value2",
    "key3", "value3",
}
err := redis.MSet(ctx, pairs...).Err()

// 批量获取
keys := []string{"key1", "key2", "key3"}
values, err := redis.MGet(ctx, keys...).Result()
```

### 管道优化

```go
// 使用管道减少网络往返
pipe := redis.Pipeline()
for i := 0; i < 100; i++ {
    key := fmt.Sprintf("key%d", i)
    pipe.Set(ctx, key, fmt.Sprintf("value%d", i), time.Hour)
}
_, err := pipe.Exec(ctx)
```

## 最佳实践

### 1. 键命名规范

- 使用冒号分隔的层次结构：`user:1:profile`
- 使用有意义的键名
- 避免过长的键名

### 2. 过期时间设置

- 为缓存设置合理的过期时间
- 使用不同的过期时间避免缓存雪崩
- 考虑业务特点设置过期策略

### 3. 连接池管理

- 根据负载调整连接池大小
- 监控连接池使用情况
- 设置合理的超时时间

### 4. 错误处理

- 区分不同类型的错误
- 实现重试机制
- 记录详细的错误日志

### 5. 性能优化

- 使用管道减少网络往返
- 合理使用批量操作
- 避免大键和大值

### 6. 监控和告警

- 监控 Redis 性能指标
- 设置连接池告警
- 监控键空间使用情况
