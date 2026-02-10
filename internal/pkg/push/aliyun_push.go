package push

import (
	"encoding/json"
	"fmt"
	"user_crud_jwt/internal/pkg/config"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/push"
)

type PushService interface {
	PushToDevice(deviceID string, title, body string, extParameters map[string]string) error
	PushToAccount(accountID string, title, body string, extParameters map[string]string) error
	PushToAll(title, body string, extParameters map[string]string) error
}

type AliyunPushService struct {
	client *push.Client
	appKey int64
}

func NewAliyunPushService() (*AliyunPushService, error) {
	cfg := config.GlobalConfig.Push
	
	// 如果配置为空，为了不阻塞启动，返回 nil
	if cfg.AccessKeyID == "" || cfg.AppKey == 0 {
		return nil, fmt.Errorf("push config is missing")
	}

	client, err := push.NewClientWithAccessKey(
		cfg.RegionID,
		cfg.AccessKeyID,
		cfg.AccessKeySecret,
	)
	if err != nil {
		return nil, err
	}

	return &AliyunPushService{
		client: client,
		appKey: cfg.AppKey,
	}, nil
}

func (s *AliyunPushService) PushToDevice(deviceID string, title, body string, extParameters map[string]string) error {
	return s.sendPush("DEVICE", deviceID, title, body, extParameters)
}

func (s *AliyunPushService) PushToAccount(accountID string, title, body string, extParameters map[string]string) error {
	return s.sendPush("ACCOUNT", accountID, title, body, extParameters)
}

func (s *AliyunPushService) PushToAll(title, body string, extParameters map[string]string) error {
	return s.sendPush("ALL", "ALL", title, body, extParameters)
}

func (s *AliyunPushService) sendPush(target, targetValue, title, body string, extParameters map[string]string) error {
	request := push.CreatePushRequest()
	request.AppKey = requests.NewInteger(int(s.appKey))
	request.Target = target
	request.TargetValue = targetValue
	request.Title = title
	request.Body = body
	request.DeviceType = "ALL" // iOS & Android
	request.PushType = "NOTICE" // 通知

	// 扩展参数 (JSON 序列化)
	if len(extParameters) > 0 {
		extJSON, _ := json.Marshal(extParameters)
		request.AndroidExtParameters = string(extJSON)
		request.IOSExtParameters = string(extJSON)
	}

	_, err := s.client.Push(request)
	return err
}

// GlobalPushService 实例
var GlobalPushService PushService

func InitPushService() error {
	service, err := NewAliyunPushService()
	if err != nil {
		return err
	}
	GlobalPushService = service
	return nil
}
