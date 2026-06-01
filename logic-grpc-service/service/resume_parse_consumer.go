package service

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"logic-grpc-service/mq"
	"logic-grpc-service/oss"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/repository"
)

type resumeParsePayload struct {
	EventID  string `json:"event_id"`
	ResumeID int64  `json:"resume_id"`
	FileType string `json:"file_type"`
	OSSKey   string `json:"oss_key"`
}

type ResumeParseConsumer struct {
	resumeRepo *repository.ResumeRepo
	ossClient  oss.Storage
}

func NewResumeParseConsumer(resumeRepo *repository.ResumeRepo, ossClient oss.Storage) *ResumeParseConsumer {
	return &ResumeParseConsumer{resumeRepo: resumeRepo, ossClient: ossClient}
}

func (c *ResumeParseConsumer) Start(ctx context.Context, mqConn *mq.Conn) error {
	return mqConn.Consume(ctx, mqConn.ResumeParseQueue(), func(ctx context.Context, body []byte) error {
		return c.handle(ctx, body)
	})
}

func (c *ResumeParseConsumer) handle(ctx context.Context, body []byte) error {
	var p resumeParsePayload
	if err := json.Unmarshal(body, &p); err != nil {
		logger.L().Error("resume parse consumer: invalid payload", zap.Error(err))
		return fmt.Errorf("invalid payload: %w", err)
	}

	text, err := extractAndStoreResumeText(ctx, p.ResumeID, p.FileType, p.OSSKey, c.ossClient, c.resumeRepo)
	if err != nil {
		logger.L().Error("resume parse consumer: extraction failed",
			zap.Int64("resume_id", p.ResumeID),
			zap.Error(err),
		)
		return err
	}

	logger.L().Info("resume parse consumer: extraction completed",
		zap.Int64("resume_id", p.ResumeID),
		zap.Int("text_len", len(text)),
	)
	return nil
}
