package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"go.uber.org/zap"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/repository"
	"logic-grpc-service/pkg/metadata"
)

// AuditLogEntry contains the fields for a third-party usage audit record.
type AuditLogEntry struct {
	UserID          int64
	Role            int32
	ServiceType     string
	Endpoint        string
	Provider        string
	Model           string
	RequestChars    int
	ResponseChars   int
	ObjectKey       string
	ObjectSize      int64
	Status          string
	ErrorCode       string
	CostMs          int
	RequestID       string
	IP              string
}

// writeAuditLog asynchronously writes a usage audit log entry.
func writeAuditLog(ctx context.Context, repo *repository.UsageLogRepo, entry AuditLogEntry) {
	if entry.RequestID == "" {
		if rid := metadata.GetRequestID(ctx); rid != "" {
			entry.RequestID = rid
		}
	}
	if entry.IP == "" {
		if ip := metadata.GetClientIP(ctx); ip != "" {
			entry.IP = ip
		}
	}
	if repo == nil {
		return
	}
	if entry.RequestID == "" {
		entry.RequestID = newRequestID()
	}
	if entry.Status == "" {
		entry.Status = "ok"
	}
	log := &model.ThirdPartyUsageLog{
		UserID:          entry.UserID,
		Role:            entry.Role,
		ServiceType:     entry.ServiceType,
		Endpoint:        entry.Endpoint,
		Provider:        entry.Provider,
		Model:           entry.Model,
		RequestChars:    entry.RequestChars,
		ResponseChars:   entry.ResponseChars,
		EstimatedTokens: estimateTokens(entry.RequestChars, entry.ResponseChars),
		ObjectKey:       entry.ObjectKey,
		ObjectSize:      entry.ObjectSize,
		Status:          entry.Status,
		ErrorCode:       entry.ErrorCode,
		CostMs:          entry.CostMs,
		RequestID:       entry.RequestID,
		IP:              entry.IP,
		CreatedAt:       time.Now(),
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := repo.Create(ctx, log); err != nil {
			logger.L().Warn("failed to write audit log", zap.String("service_type", entry.ServiceType), zap.Error(err))
		}
	}()
}

// estimateTokens provides a rough token count estimate.
// Chinese/mixed text: ~1.5 chars per token. Pure ASCII: ~4 chars per token.
// This is a rough first version — not a precise tokenizer.
func estimateTokens(requestChars, responseChars int) int {
	total := requestChars + responseChars
	if total == 0 {
		return 0
	}
	return total/2 + 1
}

func newRequestID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405")))
	}
	return hex.EncodeToString(b)
}
