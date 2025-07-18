# IGo 详细文档

欢迎使用 IGo 脚手架！本文档提供了轻量级脚手架的详细使用说明。

## 📚 文档目录

### 🎯 核心设计
- **[错误处理规范](error-handling.md)** - 脚手架错误处理原则和职责边界

### 🚀 核心功能
- **[生命周期管理](lifecycle.md)** - 钩子管理、信号处理、优雅关闭等
- **[配置管理](configuration.md)** - 配置文件、热重载、多配置源

### ⚙️ 可选组件
- **[数据库操作](database.md)** - XORM使用指南（可选组件）
- **[缓存操作](cache.md)** - Redis使用指南（可选组件）

## 🎯 设计原则

IGo 脚手架遵循以下核心原则：

- **简单易用** - 方法名通俗易懂，链式调用，开箱即用
- **职责边界清晰** - 只负责基础组件初始化，不干预业务逻辑
- **轻量灵活** - 最小化依赖，配置驱动，支持扩展
- **容错设计** - 可选组件（redis/xorm）初始化失败时静默处理

## 🚫 明确不包含的功能

为保持脚手架的简洁性，以下功能**不在**脚手架范围内：

**开发工具类**:
- 代码生成工具
- 数据库迁移工具
- Docker配置

**监控系统类**:
- 指标收集
- 性能监控
- 链路跟踪服务端

**高级功能类**:
- 分布式锁实现
- 消息队列
- 服务发现

这些功能由业务项目根据实际需求自行实现。

## 🔗 相关链接

- [项目主页](../README.md) - 项目简介和快速开始
- [GitHub 仓库](https://github.com/aichy126/igo) - 源代码和 Issues
- [示例项目](../example/) - 完整的使用示例