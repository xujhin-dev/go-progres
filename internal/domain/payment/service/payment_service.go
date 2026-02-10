package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"user_crud_jwt/internal/domain/payment/model"
	"user_crud_jwt/internal/domain/payment/repository"
	"user_crud_jwt/internal/domain/payment/strategy"
	"user_crud_jwt/internal/pkg/push"
	userService "user_crud_jwt/internal/domain/user/service"

	"github.com/google/uuid"
)

type PaymentService interface {
	CreateOrder(userID uint, amount float64, channel, subject string) (*model.Order, string, error)
	HandleNotify(channel string, params interface{}) error
	RegisterStrategy(channel string, strategy strategy.PaymentStrategy)
}

type paymentService struct {
	repo        repository.PaymentRepository
	strategies  map[string]strategy.PaymentStrategy
	userService userService.UserService // 依赖用户服务
}

func NewPaymentService(repo repository.PaymentRepository, userService userService.UserService) PaymentService {
	return &paymentService{
		repo:        repo,
		strategies:  make(map[string]strategy.PaymentStrategy),
		userService: userService,
	}
}

// RegisterStrategy 注册支付策略
func (s *paymentService) RegisterStrategy(channel string, strategy strategy.PaymentStrategy) {
	s.strategies[channel] = strategy
}

func (s *paymentService) CreateOrder(userID uint, amount float64, channel, subject string) (*model.Order, string, error) {
	strategy, ok := s.strategies[channel]
	if !ok {
		return nil, "", errors.New("unsupported payment channel")
	}

	// 1. 生成订单号
	orderNo := fmt.Sprintf("%s%s", time.Now().Format("20060102150405"), uuid.New().String()[:8])

	// 2. 创建订单记录
	order := &model.Order{
		OrderNo: orderNo,
		UserID:  userID,
		Amount:  amount,
		Status:  model.OrderStatusPending,
		Channel: channel,
		Subject: subject,
	}
	if err := s.repo.CreateOrder(order); err != nil {
		return nil, "", err
	}

	// 3. 调用支付策略获取支付参数
	payParam, err := strategy.Pay(orderNo, amount, subject)
	if err != nil {
		// 也可以选择在这里将订单标记为 cancelled
		return nil, "", err
	}

	return order, payParam, nil
}

func (s *paymentService) HandleNotify(channel string, params interface{}) error {
	strategy, ok := s.strategies[channel]
	if !ok {
		return errors.New("unsupported payment channel")
	}

	// 1. 解析回调参数
	orderNo, _, success, err := strategy.Notify(params)
	if err != nil {
		return err
	}

	if !success {
		return s.repo.UpdateOrderStatus(orderNo, model.OrderStatusCancelled, nil, nil)
	}

	// 2. 更新订单状态
	now := time.Now()
	// 这里可以把原始回调参数存入 extra_params
	extraJSON, _ := json.Marshal(params)

	if err := s.repo.UpdateOrderStatus(orderNo, model.OrderStatusPaid, &now, extraJSON); err != nil {
		return err
	}

	// 3. 业务集成：查询订单信息以获取 UserID
	order, err := s.repo.GetOrderByNo(orderNo)
	if err != nil {
		// 记录严重错误：订单状态已更新但无法获取订单详情
		return err
	}

	// 4. 升级会员 (假设购买的是30天会员)
	// 在实际业务中，应根据 order.Amount 或 order.Subject/ProductID 决定会员时长
	// 这里简化逻辑：只要支付成功就送30天
	if err := s.userService.UpgradeMember(order.UserID, 30*24*time.Hour); err != nil {
		// 记录错误，可能需要人工介入或重试
		fmt.Printf("Failed to upgrade member for user %d: %v\n", order.UserID, err)
	}

	// 5. 推送通知
	// 注意：GlobalPushService 可能为 nil (如果未配置)
	if push.GlobalPushService != nil {
		title := "支付成功"
		body := fmt.Sprintf("您的订单 %s 已支付成功，会员权益已生效。", orderNo)
		// 使用 PushToAccount (假设 UserID 转 string 就是 AccountID)
		accountID := fmt.Sprintf("%d", order.UserID)
		go push.GlobalPushService.PushToAccount(accountID, title, body, nil)
	}

	return nil
}
