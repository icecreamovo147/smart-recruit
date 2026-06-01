package oss

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

type TencentCOSClient struct {
	client       *cos.Client
	secretID     string
	secretKey    string
	presignCache *PresignCache
}

func (c *TencentCOSClient) SetPresignCache(cache *PresignCache) { c.presignCache = cache }
func (c *TencentCOSClient) ProviderName() string                { return ProviderTencentCOS }

func NewTencentCOSClient(endpoint, secretID, secretKey, bucketName, publicBaseURL string) (*TencentCOSClient, error) {
	baseURL := strings.TrimRight(publicBaseURL, "/")
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s.%s", bucketName, endpoint)
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse cos public base url: %w", err)
	}
	c := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
		},
	})
	return &TencentCOSClient{client: c, secretID: secretID, secretKey: secretKey}, nil
}

func (c *TencentCOSClient) GeneratePresignedPutURL(ossKey string, _ string) (string, time.Time, error) {
	return c.signURL(context.Background(), http.MethodPut, ossKey)
}

func (c *TencentCOSClient) GeneratePresignedGetURL(ossKey string) (string, error) {
	if c.presignCache != nil {
		if cached, ok := c.presignCache.Get(context.Background(), ossKey); ok {
			return cached, nil
		}
	}
	signed, _, err := c.signURL(context.Background(), http.MethodGet, ossKey)
	if err != nil {
		return "", err
	}
	if c.presignCache != nil {
		c.presignCache.Set(context.Background(), ossKey, signed)
	}
	return signed, err
}

func (c *TencentCOSClient) VerifyObject(ctx context.Context, ossKey string) error {
	if strings.TrimSpace(ossKey) == "" {
		return fmt.Errorf("oss key is empty")
	}
	_, err := c.client.Object.Head(ctx, ossKey, nil)
	return err
}

func (c *TencentCOSClient) VerifyObjectSize(ctx context.Context, ossKey string, maxSize int64) error {
	resp, err := c.client.Object.Head(ctx, ossKey, nil)
	if err != nil {
		return err
	}
	if resp.ContentLength > maxSize {
		return fmt.Errorf("file size %d exceeds maximum allowed %d bytes", resp.ContentLength, maxSize)
	}
	return nil
}

func (c *TencentCOSClient) DownloadObject(ctx context.Context, ossKey string) ([]byte, error) {
	if strings.TrimSpace(ossKey) == "" {
		return nil, fmt.Errorf("oss key is empty")
	}
	resp, err := c.client.Object.Get(ctx, ossKey, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, MaxResumeDownloadSizeBytes))
}

func (c *TencentCOSClient) DeleteObject(ctx context.Context, ossKey string) error {
	if strings.TrimSpace(ossKey) == "" {
		return fmt.Errorf("oss key is empty")
	}
	_, err := c.client.Object.Delete(ctx, ossKey)
	return err
}

func (c *TencentCOSClient) SavePresignSessionWithID(ctx context.Context, uploadID string, session PresignSession) error {
	return savePresignSessionWithID(ctx, c.presignCache, uploadID, session)
}

func (c *TencentCOSClient) SavePresignSession(ctx context.Context, session PresignSession) (string, error) {
	return savePresignSessionWithCache(ctx, c.presignCache, session)
}

func (c *TencentCOSClient) GetAndDeletePresignSession(ctx context.Context, uploadID string) (*PresignSession, error) {
	return getAndDeletePresignSessionWithCache(ctx, c.presignCache, uploadID)
}

func (c *TencentCOSClient) CopyObject(ctx context.Context, srcKey, dstKey string) error {
	if strings.TrimSpace(srcKey) == "" || strings.TrimSpace(dstKey) == "" {
		return fmt.Errorf("oss key is empty")
	}
	srcURL := fmt.Sprintf("%s/%s", c.client.BaseURL.BucketURL.Host, strings.TrimLeft(srcKey, "/"))
	_, _, err := c.client.Object.Copy(ctx, dstKey, srcURL, nil)
	return err
}

func (c *TencentCOSClient) signURL(ctx context.Context, method, ossKey string) (string, time.Time, error) {
	if c.client == nil {
		return "", time.Time{}, fmt.Errorf("cos client is nil")
	}
	if strings.TrimSpace(ossKey) == "" {
		return "", time.Time{}, fmt.Errorf("oss key is empty")
	}
	expireAt := time.Now().Add(PresignExpiry)
	presignedURL, err := c.client.Object.GetPresignedURL(ctx, method, ossKey, c.secretID, c.secretKey, 15*time.Minute, nil)
	if err != nil {
		return "", time.Time{}, err
	}
	return presignedURL.String(), expireAt, nil
}
