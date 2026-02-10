package strategy

import (
	"context"
	"errors"
	"net/http"
	"user_crud_jwt/internal/pkg/config"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/app"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

type WechatStrategy struct {
	client    *core.Client
	config    config.WechatPayConfig
	certMgr   core.CertificateVisitor
	handler   *notify.Handler
}

func NewWechatStrategy() (*WechatStrategy, error) {
	cfg := config.GlobalConfig.Wechat
	if cfg.MchID == "" {
		return nil, errors.New("wechat pay config missing")
	}

	// 1. 加载商户私钥
	mchPrivateKey, err := utils.LoadPrivateKey(cfg.MchPrivateKey)
	if err != nil {
		return nil, err
	}

	// 2. 初始化 Client
	ctx := context.Background()
	opts := []core.ClientOption{
		option.WithWechatPayAutoAuthCipher(cfg.MchID, cfg.MchCertificateSerial, mchPrivateKey, cfg.APIv3Key),
	}
	
	client, err := core.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	// 3. 初始化证书管理器 (用于验签)
	certVisitor := downloader.MgrInstance().GetCertificateVisitor(cfg.MchID)
	
	// 4. 初始化 Notify Handler
	handler := notify.NewNotifyHandler(cfg.APIv3Key, verifiers.NewSHA256WithRSAVerifier(certVisitor))

	return &WechatStrategy{
		client:    client,
		config:    cfg,
		certMgr:   certVisitor,
		handler:   handler,
	}, nil
}

func (s *WechatStrategy) Pay(orderNo string, amount float64, subject string) (string, error) {
	// 转换为分
	amountFen := int64(amount * 100)
	
	req := app.PrepayRequest{
		Appid:       core.String(s.config.AppID),
		Mchid:       core.String(s.config.MchID),
		Description: core.String(subject),
		OutTradeNo:  core.String(orderNo),
		NotifyUrl:   core.String(s.config.NotifyURL),
		Amount: &app.Amount{
			Total: core.Int64(amountFen),
		},
	}

	svc := app.AppApiService{Client: s.client}
	resp, _, err := svc.Prepay(context.Background(), req)
	if err != nil {
		return "", err
	}
	
	return *resp.PrepayId, nil
}

func (s *WechatStrategy) Notify(params interface{}) (string, float64, bool, error) {
	req, ok := params.(*http.Request)
	if !ok {
		return "", 0, false, errors.New("invalid params type, expected *http.Request")
	}

	transaction := new(payments.Transaction)
	_, err := s.handler.ParseNotifyRequest(context.Background(), req, transaction)
	if err != nil {
		return "", 0, false, err
	}

	success := false
	if *transaction.TradeState == "SUCCESS" {
		success = true
	}

	amount := float64(*transaction.Amount.Total) / 100.0
	return *transaction.OutTradeNo, amount, success, nil
}

var _ PaymentStrategy = (*WechatStrategy)(nil)
