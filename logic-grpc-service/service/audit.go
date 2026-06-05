package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"go.uber.org/zap"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/pkg/metadata"
	"logic-grpc-service/repository"
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

// createUsageLogSync creates a usage log entry synchronously and returns its ID.
// Unlike writeAuditLog (which is fire-and-forget), this function blocks on the DB write
// so the caller can use the returned ID to link related records (e.g., AI usage auth context).
func createUsageLogSync(ctx context.Context, repo *repository.UsageLogRepo, entry AuditLogEntry) int64 {
	if repo == nil {
		return 0
	}
	if entry.RequestID == "" {
		if rid := metadata.GetRequestID(ctx); rid != "" {
			entry.RequestID = rid
		}
	}
	if entry.RequestID == "" {
		entry.RequestID = newRequestID()
	}
	if entry.IP == "" {
		if ip := metadata.GetClientIP(ctx); ip != "" {
			entry.IP = ip
		}
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
	if err := repo.Create(ctx, log); err != nil {
		logger.L().Warn("failed to write usage log sync", zap.String("service_type", entry.ServiceType), zap.Error(err))
		return 0
	}
	return log.ID
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

// AIUsageAuthContextEntry carries the RBAC context for an AI usage audit.
type AIUsageAuthContextEntry struct {
	UsageLogID    uint64
	ActorUserID   uint64
	AccountType   string
	RoleKeys      []string
	PermissionKey string
	ScopeKeys     []string
	ResourceType  string
	ResourceID    uint64
	Decision      string
	RequestID     string
}

// writeAIUsageAuthContext asynchronously writes an AI usage RBAC context record.
func writeAIUsageAuthContext(ctx context.Context, repo *repository.UsageAuditContextRepo, entry AIUsageAuthContextEntry) {
	if repo == nil {
		return
	}
	if entry.RequestID == "" {
		if rid := metadata.GetRequestID(ctx); rid != "" {
			entry.RequestID = rid
		}
	}
	if entry.RequestID == "" {
		entry.RequestID = newRequestID()
	}
	if entry.Decision == "" {
		entry.Decision = "allowed"
	}
	record := &model.AIUsageAuthContext{
		UsageLogID:    entry.UsageLogID,
		ActorUserID:   entry.ActorUserID,
		AccountType:   entry.AccountType,
		RoleKeys:      strings.Join(entry.RoleKeys, ","),
		PermissionKey: entry.PermissionKey,
		ScopeKeys:     strings.Join(entry.ScopeKeys, ","),
		ResourceType:  entry.ResourceType,
		ResourceID:    entry.ResourceID,
		Decision:      entry.Decision,
		RequestID:     entry.RequestID,
		CreatedAt:     time.Now(),
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := repo.Create(ctx, record); err != nil {
			logger.L().Warn("failed to write AI usage auth context",
				zap.String("permission", entry.PermissionKey),
				zap.Uint64("actor", entry.ActorUserID),
				zap.Error(err),
			)
		}
	}()
}
