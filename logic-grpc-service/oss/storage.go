package oss

import (
	"context"
	"time"
)

// Timing constants used across the OSS package.
const (
	PresignExpiry = 15 * time.Minute
)

type Storage interface {
	SetPresignCache(cache *PresignCache)
	ProviderName() string
	GeneratePresignedPutURL(ossKey string, contentType string) (string, time.Time, error)
	GeneratePresignedGetURL(ossKey string) (string, error)
	VerifyObject(ctx context.Context, ossKey string) error
	VerifyObjectSize(ctx context.Context, ossKey string, maxSize int64) error
	DownloadObject(ctx context.Context, ossKey string) ([]byte, error)
	CopyObject(ctx context.Context, srcKey, dstKey string) error
	DeleteObject(ctx context.Context, ossKey string) error
	SavePresignSession(ctx context.Context, session PresignSession) (string, error)
	SavePresignSessionWithID(ctx context.Context, uploadID string, session PresignSession) error
	GetAndDeletePresignSession(ctx context.Context, uploadID string) (*PresignSession, error)
}
