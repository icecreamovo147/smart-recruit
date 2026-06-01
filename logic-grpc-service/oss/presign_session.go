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
	UserID      int64  `json:"user_id"`
	OssKey      string `json:"oss_key"`
	FileName    string `json:"file_name"`
	FileType    string `json:"file_type"`
	ContentType string `json:"content_type"`
	MaxSize     int64  `json:"max_size"`
	Status      string `json:"status"`
}

// MaxResumeSizeBytes is the hard cap for resume file size (20 MB).
const MaxResumeSizeBytes = 20 * 1024 * 1024

// MaxResumeDownloadSizeBytes is the download limit to prevent OOM (25 MB).
const MaxResumeDownloadSizeBytes = 25 * 1024 * 1024

func GenerateUploadID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func presignSessionKey(uploadID string) string {
	return fmt.Sprintf("oss:presign_session:%s", uploadID)
}

// SavePresignSessionWithID stores an upload session with a pre-generated upload ID.
func savePresignSessionWithID(ctx context.Context, cache *PresignCache, uploadID string, session PresignSession) error {
	if cache == nil {
		return fmt.Errorf("presign cache is not available")
	}
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	key := presignSessionKey(uploadID)
	return cache.Rdb.Set(ctx, key, string(data), 15*time.Minute).Err()
}

// SavePresignSession stores an upload session in Redis with 15-minute TTL via the given cache.
func savePresignSessionWithCache(ctx context.Context, cache *PresignCache, session PresignSession) (string, error) {
	if cache == nil {
		return "", fmt.Errorf("presign cache is not available")
	}
	uploadID, err := GenerateUploadID()
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(session)
	if err != nil {
		return "", err
	}
	key := presignSessionKey(uploadID)
	if err := cache.Rdb.Set(ctx, key, string(data), 15*time.Minute).Err(); err != nil {
		return "", err
	}
	return uploadID, nil
}

// getAndDeletePresignSessionWithCache atomically retrieves and deletes the session.
func getAndDeletePresignSessionWithCache(ctx context.Context, cache *PresignCache, uploadID string) (*PresignSession, error) {
	if cache == nil {
		return nil, fmt.Errorf("presign cache is not available")
	}
	key := presignSessionKey(uploadID)
	data, err := cache.Rdb.GetDel(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("upload session not found or expired: %w", err)
	}
	var session PresignSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}
	return &session, nil
}
