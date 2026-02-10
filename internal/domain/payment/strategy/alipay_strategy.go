package strategy

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"user_crud_jwt/internal/pkg/config"

	"github.com/smartwalle/alipay/v3"
)

type AlipayStrategy struct {
	client *alipay.Client
	config config.AlipayConfig
}

func NewAlipayStrategy() (*AlipayStrategy, error) {
	cfg := config.GlobalConfig.Alipay
	if cfg.AppID == "" {
		return nil, errors.New("alipay config missing")
	}

	client, err := alipay.New(cfg.AppID, cfg.PrivateKey, cfg.IsProduction)
	if err != nil {
		return nil, err
	}

	// 加载支付宝公钥 (用于验证签名)
	if err = client.LoadAliPayPublicKey(cfg.PublicKey); err != nil {
		return nil, err
	}

	return &AlipayStrategy{
		client: client,
		config: cfg,
	}, nil
}

// Pay 发起支付 (App支付)
func (s *AlipayStrategy) Pay(orderNo string, amount float64, subject string) (string, error) {
	p := alipay.TradeAppPay{}
	p.NotifyURL = s.config.NotifyURL
	p.Subject = subject
	p.OutTradeNo = orderNo
	p.TotalAmount = fmt.Sprintf("%.2f", amount)
	p.ProductCode = "QUICK_MSECURITY_PAY" // App支付产品码

	// 生成签名后的参数字符串
	result, err := s.client.TradeAppPay(p)
	if err != nil {
		return "", err
	}
	return result, nil
}

// Notify 处理回调
func (s *AlipayStrategy) Notify(params interface{}) (string, float64, bool, error) {
	// params 预期是 url.Values (gin context.Request.Form)
	values, ok := params.(url.Values)
	if !ok {
		return "", 0, false, errors.New("invalid params type, expected url.Values")
	}

	// 1. 验证签名
	noti, err := s.client.DecodeNotification(values)
	if err != nil {
		return "", 0, false, err
	}

	// 2. 检查交易状态
	// TRADE_SUCCESS 或 TRADE_FINISHED 表示成功
	success := false
	if noti.TradeStatus == alipay.TradeStatusSuccess || noti.TradeStatus == alipay.TradeStatusFinished {
		success = true
	}

	// 3. 解析金额
	amount, _ := strconv.ParseFloat(noti.TotalAmount, 64)

	return noti.OutTradeNo, amount, success, nil
}

// 确保实现了接口
var _ PaymentStrategy = (*AlipayStrategy)(nil)
