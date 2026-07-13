package oss

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	aliyunoss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/tencentyun/cos-go-sdk-v5"
)

type AliyunOSSClient struct {
	bucket       *aliyunoss.Bucket
	presignCache *PresignCache
}

func NewAliyunOSSClient(endpoint, accessKeyID, accessKeySecret, bucketName, publicBaseURL string) (*AliyunOSSClient, error) {
	client, err := aliyunoss.New(endpoint, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("create aliyun oss client: %w", err)
	}
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, fmt.Errorf("get aliyun oss bucket %s: %w", bucketName, err)
	}
	return &AliyunOSSClient{bucket: bucket}, nil
}

func (c *AliyunOSSClient) SetPresignCache(cache *PresignCache) { c.presignCache = cache }
func (c *AliyunOSSClient) ProviderName() string                { return ProviderAliyunOSS }

func (c *AliyunOSSClient) GeneratePresignedPutURL(ossKey string, contentType string) (string, time.Time, error) {
	if strings.TrimSpace(ossKey) == "" {
		return "", time.Time{}, fmt.Errorf("oss key is empty")
	}
	options := []aliyunoss.Option{}
	if strings.TrimSpace(contentType) != "" {
		options = append(options, aliyunoss.ContentType(contentType))
	}
	signed, err := c.bucket.SignURL(ossKey, aliyunoss.HTTPPut, 15*60, options...)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign put url: %w", err)
	}
	expireAt := time.Now().Add(PresignExpiry)
	return signed, expireAt, nil
}

func (c *AliyunOSSClient) GeneratePresignedGetURL(ossKey string) (string, error) {
	if c.presignCache != nil {
		if cached, ok := c.presignCache.Get(context.Background(), ossKey); ok {
			return cached, nil
		}
	}
	signed, err := c.bucket.SignURL(ossKey, aliyunoss.HTTPGet, 15*60)
	if err != nil {
		return "", fmt.Errorf("sign get url: %w", err)
	}
	if c.presignCache != nil {
		c.presignCache.Set(context.Background(), ossKey, signed)
	}
	return signed, nil
}

func (c *AliyunOSSClient) VerifyObject(ctx context.Context, ossKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(ossKey) == "" {
		return fmt.Errorf("oss key is empty")
	}
	exists, err := c.bucket.IsObjectExist(ossKey)
	if err != nil {
		return err
	}
	if !exists {
		return &aliyunoss.ServiceError{Code: "NoSuchKey", StatusCode: http.StatusNotFound}
	}
	return ctx.Err()
}

func (c *AliyunOSSClient) VerifyObjectSize(ctx context.Context, ossKey string, maxSize int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	meta, err := c.bucket.GetObjectDetailedMeta(ossKey)
	if err != nil {
		return err
	}
	contentLengthStr := meta.Get("Content-Length")
	if contentLengthStr != "" {
		var size int64
		if _, err := fmt.Sscanf(contentLengthStr, "%d", &size); err == nil && size > maxSize {
			return fmt.Errorf("file size %d exceeds maximum allowed %d bytes", size, maxSize)
		}
	}
	return ctx.Err()
}

func (c *AliyunOSSClient) DownloadObject(ctx context.Context, ossKey string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(ossKey) == "" {
		return nil, fmt.Errorf("oss key is empty")
	}
	body, err := c.bucket.GetObject(ossKey)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	data, readErr := io.ReadAll(io.LimitReader(body, MaxResumeDownloadSizeBytes))
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return data, readErr
}

func (c *AliyunOSSClient) DeleteObject(ctx context.Context, ossKey string) error {
	if strings.TrimSpace(ossKey) == "" {
		return fmt.Errorf("oss key is empty")
	}
	return c.bucket.DeleteObject(ossKey)
}

func (c *AliyunOSSClient) CopyObject(ctx context.Context, srcKey, dstKey string) error {
	if strings.TrimSpace(srcKey) == "" || strings.TrimSpace(dstKey) == "" {
		return fmt.Errorf("oss key is empty")
	}
	_, err := c.bucket.CopyObject(srcKey, dstKey)
	return err
}

func (c *AliyunOSSClient) SavePresignSessionWithID(ctx context.Context, uploadID string, session PresignSession) error {
	return savePresignSessionWithID(ctx, c.presignCache, uploadID, session)
}

func (c *AliyunOSSClient) SavePresignSession(ctx context.Context, session PresignSession) (string, error) {
	return savePresignSessionWithCache(ctx, c.presignCache, session)
}

func (c *AliyunOSSClient) GetAndDeletePresignSession(ctx context.Context, uploadID string) (*PresignSession, error) {
	return getAndDeletePresignSessionWithCache(ctx, c.presignCache, uploadID)
}

// IsNotFound checks whether an error indicates that the requested object does not exist.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if cos.IsNotFoundError(err) {
		return true
	}
	var serviceErr aliyunoss.ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr.StatusCode == http.StatusNotFound || serviceErr.Code == "NoSuchKey"
	}
	return false
}
