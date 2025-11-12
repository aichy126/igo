package dao

import (
	"fmt"
	"time"

	"github.com/aichy126/igo"
	"github.com/aichy126/igo/context"
	"github.com/aichy126/igo/log"
)

// Order 订单模型
type Order struct {
	ID        int64     `xorm:"pk autoincr"`
	UserID    string    `xorm:"not null"`
	Amount    float64   `xorm:"not null"`
	Status    string    `xorm:"not null default 'pending'"`
	CreatedAt time.Time `xorm:"created"`
}

// OrderItem 订单项模型
type OrderItem struct {
	ID       int64   `xorm:"pk autoincr"`
	OrderID  int64   `xorm:"not null"`
	Product  string  `xorm:"not null"`
	Quantity int     `xorm:"not null"`
	Price    float64 `xorm:"not null"`
}

// OrderDao 订单DAO
type OrderDao struct{}

// NewOrderDao 创建订单DAO
func NewOrderDao() *OrderDao {
	return &OrderDao{}
}

// CreateOrderWithItems 创建订单及订单项（跨表事务示例）
// 这是使用新的 BeginTx 方法的完整示例
func (d *OrderDao) CreateOrderWithItems(ctx context.IContext, userID string, items []OrderItem) (*Order, error) {
	// 记录开始
	ctx.LogInfo("开始创建订单", log.Any("userID", userID), log.Any("itemCount", len(items)))

	// 开启事务（新功能）
	sess, err := igo.App.DB.BeginTx("test")
	if err != nil {
		ctx.LogError("开启事务失败", log.Any("error", err))
		return nil, fmt.Errorf("开启事务失败: %w", err)
	}
	defer sess.Close()

	// 使用 defer 确保发生错误时回滚
	success := false
	defer func() {
		if !success {
			sess.Rollback()
			log.Warn("事务已回滚")
		}
	}()

	// 1. 计算总金额
	var totalAmount float64
	for _, item := range items {
		totalAmount += item.Price * float64(item.Quantity)
	}

	// 2. 创建订单
	order := &Order{
		UserID: userID,
		Amount: totalAmount,
		Status: "pending",
	}

	affected, err := sess.Table("orders").Insert(order)
	if err != nil {
		ctx.LogError("创建订单失败", log.Any("error", err))
		return nil, fmt.Errorf("创建订单失败: %w", err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("订单创建失败：未插入任何记录")
	}

	ctx.LogInfo("订单创建成功", log.Any("orderID", order.ID))

	// 3. 创建订单项
	for i := range items {
		items[i].OrderID = order.ID
	}

	affected, err = sess.Table("order_items").Insert(&items)
	if err != nil {
		ctx.LogError("创建订单项失败", log.Any("error", err))
		return nil, fmt.Errorf("创建订单项失败: %w", err)
	}

	ctx.LogInfo("订单项创建成功", log.Any("count", affected))

	// 4. 更新用户订单数（演示跨表操作）
	_, err = sess.Exec("UPDATE test0 SET user_id = ? WHERE id = 1", userID)
	if err != nil {
		ctx.LogError("更新用户信息失败", log.Any("error", err))
		return nil, fmt.Errorf("更新用户信息失败: %w", err)
	}

	// 5. 提交事务
	if err := sess.Commit(); err != nil {
		ctx.LogError("提交事务失败", log.Any("error", err))
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	success = true
	ctx.LogInfo("订单创建完成", log.Any("orderID", order.ID), log.Any("amount", totalAmount))
	return order, nil
}

// BatchSync 批量同步数据（跨表操作示例）
func (d *OrderDao) BatchSync(ctx context.IContext) error {
	ctx.LogInfo("开始批量同步数据")

	// 使用 NewSession（手动控制事务）
	sess := igo.App.DB.NewSession("test")
	if sess == nil {
		return fmt.Errorf("创建Session失败")
	}
	defer sess.Close()

	// 手动开启事务
	if err := sess.Begin(); err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}

	success := false
	defer func() {
		if !success {
			sess.Rollback()
		}
	}()

	// 从表1查询数据
	var sourceData []Test0
	err := sess.Table("test0").Limit(10).Find(&sourceData)
	if err != nil {
		ctx.LogError("查询源数据失败", log.Any("error", err))
		return fmt.Errorf("查询源数据失败: %w", err)
	}

	ctx.LogInfo("查询到源数据", log.Any("count", len(sourceData)))

	// 插入到表2
	var targetData []Test2
	for _, src := range sourceData {
		targetData = append(targetData, Test2{
			CreatedAt: src.CreatedAt,
			UpdatedAt: src.UpdatedAt,
		})
	}

	if len(targetData) > 0 {
		_, err = sess.Table("test2").Insert(&targetData)
		if err != nil {
			ctx.LogError("插入目标数据失败", log.Any("error", err))
			return fmt.Errorf("插入目标数据失败: %w", err)
		}
		ctx.LogInfo("插入目标数据成功", log.Any("count", len(targetData)))
	}

	// 提交事务
	if err := sess.Commit(); err != nil {
		ctx.LogError("提交事务失败", log.Any("error", err))
		return fmt.Errorf("提交事务失败: %w", err)
	}

	success = true
	ctx.LogInfo("批量同步完成")
	return nil
}

// SyncTables 同步表结构（演示使用）
func (d *OrderDao) SyncTables() error {
	// 获取数据库引擎
	dm := igo.App.DB.Get("test")
	if dm == nil || dm.WriteDB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	// 同步表结构
	err := dm.WriteDB.Sync2(new(Order), new(OrderItem))
	if err != nil {
		return fmt.Errorf("同步表结构失败: %w", err)
	}

	log.Info("表结构同步成功")
	return nil
}
