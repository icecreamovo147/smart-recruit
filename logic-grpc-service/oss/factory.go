package oss

import (
	"fmt"
	"strings"
)

const (
	ProviderTencentCOS = "tencent_cos"
	ProviderAliyunOSS  = "aliyun_oss"
)

type Config struct {
	Provider        string
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	BucketName      string
	PublicBaseURL   string
}

func NewStorage(cfg Config) (Storage, error) {
	provider := strings.TrimSpace(cfg.Provider)
	if provider == "" {
		provider = ProviderTencentCOS
	}

	switch provider {
	case ProviderTencentCOS:
		return NewTencentCOSClient(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret, cfg.BucketName, cfg.PublicBaseURL)
	case ProviderAliyunOSS:
		return NewAliyunOSSClient(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret, cfg.BucketName, cfg.PublicBaseURL)
	default:
		return nil, fmt.Errorf("unsupported oss provider: %s", provider)
	}
}
