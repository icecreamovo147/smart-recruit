package oss

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// PresignSession records the metadata for an active resume upload presign request.
type PresignSession struct {
	UserID   int64  `json:"user_id"`
	OssKey   string `json:"oss_key"`
	FileName string `json:"file_name"`
	FileType string `json:"file_type"`
	MaxSize  int64  `json:"max_size"`
}

// MaxResumeSizeBytes is the hard cap for resume file size (20 MB).
const MaxResumeSizeBytes = 20 * 1024 * 1024

// MaxResumeDownloadSizeBytes is the download limit to prevent OOM (25 MB).
const MaxResumeDownloadSizeBytes = 25 * 1024 * 1024

func generateUploadID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func presignSessionKey(uploadID string) string {
	return fmt.Sprintf("oss:presign_session:%s", uploadID)
}

// SavePresignSession stores an upload session in Redis with 15-minute TTL.
// Returns the generated upload_id.
func (c *Client) SavePresignSession(ctx context.Context, session PresignSession) (string, error) {
	if c.presignCache == nil {
		return "", fmt.Errorf("presign cache is not available")
	}
	uploadID, err := generateUploadID()
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(session)
	if err != nil {
		return "", err
	}
	key := presignSessionKey(uploadID)
	if err := c.presignCache.Rdb.Set(ctx, key, string(data), 15*time.Minute).Err(); err != nil {
		return "", err
	}
	return uploadID, nil
}

// GetAndDeletePresignSession retrieves and atomically deletes the presign session.
// This ensures each upload_id can only be used once (prevents replay).
func (c *Client) GetAndDeletePresignSession(ctx context.Context, uploadID string) (*PresignSession, error) {
	if c.presignCache == nil {
		return nil, fmt.Errorf("presign cache is not available")
	}
	key := presignSessionKey(uploadID)
	data, err := c.presignCache.Rdb.GetDel(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("upload session not found or expired: %w", err)
	}
	var session PresignSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// VerifyObjectSize checks that the uploaded object does not exceed the max size.
func (c *Client) VerifyObjectSize(ctx context.Context, ossKey string, maxSize int64) error {
	resp, err := c.client.Object.Head(ctx, ossKey, nil)
	if err != nil {
		return err
	}
	if resp.ContentLength > maxSize {
		return fmt.Errorf("file size %d exceeds maximum allowed %d bytes", resp.ContentLength, maxSize)
	}
	return nil
}
