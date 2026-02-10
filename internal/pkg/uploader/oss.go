package uploader

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"time"
	"user_crud_jwt/internal/pkg/config"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/google/uuid"
)

type Uploader interface {
	UploadFile(file *multipart.FileHeader) (string, error)
}

type AliyunOSSUploader struct {
	client *oss.Client
	bucket *oss.Bucket
	config config.OSSConfig
}

func NewAliyunOSSUploader() (*AliyunOSSUploader, error) {
	cfg := config.GlobalConfig.OSS
	client, err := oss.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	bucket, err := client.Bucket(cfg.BucketName)
	if err != nil {
		return nil, err
	}

	return &AliyunOSSUploader{
		client: client,
		bucket: bucket,
		config: cfg,
	}, nil
}

func (u *AliyunOSSUploader) UploadFile(file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// Generate unique filename: YYYYMMDD/uuid.ext
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s/%s%s", time.Now().Format("20060102"), uuid.New().String(), ext)

	// Upload
	err = u.bucket.PutObject(filename, src)
	if err != nil {
		return "", err
	}

	// Return public URL
	// Note: Assuming bucket is public-read or using CDN. 
	// If private, need to sign URL. For simplicity in this demo, we construct public URL.
	url := fmt.Sprintf("https://%s.%s/%s", u.config.BucketName, u.config.Endpoint, filename)
	return url, nil
}

// GlobalUploader instance
var GlobalUploader Uploader

func InitUploader() error {
	uploader, err := NewAliyunOSSUploader()
	if err != nil {
		return err
	}
	GlobalUploader = uploader
	return nil
}
