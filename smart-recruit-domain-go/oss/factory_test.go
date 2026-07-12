package oss

import (
	"testing"
)

func TestNewStorage_DefaultProvider(t *testing.T) {
	storage, err := NewStorage(Config{
		Provider:        "",
		Endpoint:        "cos.ap-shanghai.myqcloud.com",
		AccessKeyID:     "test-id",
		AccessKeySecret: "test-secret",
		BucketName:      "test-bucket-12345",
		PublicBaseURL:   "https://test-bucket-12345.cos.ap-shanghai.myqcloud.com",
	})
	if err != nil {
		t.Fatalf("NewStorage with empty provider: %v", err)
	}
	if _, ok := storage.(*TencentCOSClient); !ok {
		t.Errorf("empty provider should return *TencentCOSClient, got %T", storage)
	}
}

func TestNewStorage_TencentCOS(t *testing.T) {
	storage, err := NewStorage(Config{
		Provider:        ProviderTencentCOS,
		Endpoint:        "cos.ap-shanghai.myqcloud.com",
		AccessKeyID:     "test-id",
		AccessKeySecret: "test-secret",
		BucketName:      "test-bucket-12345",
		PublicBaseURL:   "https://test-bucket-12345.cos.ap-shanghai.myqcloud.com",
	})
	if err != nil {
		t.Fatalf("NewStorage with tencent_cos: %v", err)
	}
	if _, ok := storage.(*TencentCOSClient); !ok {
		t.Errorf("tencent_cos provider should return *TencentCOSClient, got %T", storage)
	}
}

func TestNewStorage_AliyunOSS(t *testing.T) {
	storage, err := NewStorage(Config{
		Provider:        ProviderAliyunOSS,
		Endpoint:        "oss-cn-shanghai.aliyuncs.com",
		AccessKeyID:     "test-id",
		AccessKeySecret: "test-secret",
		BucketName:      "test-bucket",
	})
	if err != nil {
		t.Fatalf("NewStorage with aliyun_oss: %v", err)
	}
	if _, ok := storage.(*AliyunOSSClient); !ok {
		t.Errorf("aliyun_oss provider should return *AliyunOSSClient, got %T", storage)
	}
}

func TestNewStorage_UnsupportedProvider(t *testing.T) {
	_, err := NewStorage(Config{
		Provider:        "aws_s3",
		Endpoint:        "s3.amazonaws.com",
		AccessKeyID:     "test-id",
		AccessKeySecret: "test-secret",
		BucketName:      "test-bucket",
	})
	if err == nil {
		t.Fatal("unsupported provider should return error")
	}
	expected := "unsupported oss provider: aws_s3"
	if err.Error() != expected {
		t.Errorf("error message = %q, want %q", err.Error(), expected)
	}
}

func TestContentTypeFromFileType(t *testing.T) {
	cases := map[string]string{
		"pdf":     "application/pdf",
		"PDF":     "application/pdf",
		"docx":    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"doc":     "application/msword",
		"":        "application/octet-stream",
		"unknown": "application/octet-stream",
	}
	for input, want := range cases {
		if got := ContentTypeFromFileType(input); got != want {
			t.Errorf("ContentTypeFromFileType(%q) = %q, want %q", input, got, want)
		}
	}
}
