package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	recruitingruntime "smart-recruit-ai-agent-service/internal/application/recruiting_intelligence"
	platformlogger "smart-recruit-platform-go/logger"
	platformmetadata "smart-recruit-platform-go/metadata"
	platformobservability "smart-recruit-platform-go/observability"
	"smart-recruit-proto/recruitment/pb"
)

type billingOwnerKind int

const (
	billingOwnerTenant billingOwnerKind = iota + 1
	billingOwnerUser
)

type billingReservationContextKey struct{}

type billingReservationState struct {
	mu                 sync.Mutex
	reservationNo      string
	runtimeModel       RuntimeModelInfo
	done               bool
	processing         bool
	settlementRequired bool
}

type BillingOutboxReservation struct {
	ReservationNo string
	OwnerType     string
	OwnerID       int64
	UserID        int64
	Capability    string
	Operation     string
	ProviderKey   string
	ModelKey      string
}

type billingMeterStore interface {
	CreateBillingOutboxReservation(context.Context, BillingOutboxReservation) error
	QueueBillingOutboxSettlement(context.Context, *pb.SettleAIUsageRequest) error
	QueueBillingOutboxCancellation(context.Context, *pb.CancelAIUsageRequest) error
	MarkBillingOutboxSettled(context.Context, string) error
	MarkBillingOutboxCancelled(context.Context, string) error
}

func (s *nativeAIService) resolveCapabilityRuntimeModel(ctx context.Context, kind billingOwnerKind, userID int64, capability, audience string, requestedModelID int64) (RuntimeModelInfo, error) {
	capabilityKey := strings.TrimSuffix(strings.TrimSpace(capability), ".enabled")
	if capabilityKey == "" {
		return RuntimeModelInfo{}, status.Error(codes.InvalidArgument, "AI capability is required")
	}
	versionID := int64(0)
	if s != nil && s.billing != nil {
		owner := &pb.BillingOwner{}
		switch kind {
		case billingOwnerTenant:
			owner.Type = pb.BillingOwnerType_BILLING_OWNER_TYPE_TENANT
			owner.Id = platformmetadata.GetAuthTenantID(ctx)
		case billingOwnerUser:
			owner.Type = pb.BillingOwnerType_BILLING_OWNER_TYPE_USER
			owner.Id = userID
		}
		if owner.Id <= 0 {
			return RuntimeModelInfo{}, status.Error(codes.FailedPrecondition, "AI billing owner is missing")
		}
		access, err := s.billing.CheckAIAccess(ctx, &pb.CheckAIAccessRequest{Owner: owner, UserId: userID, Capability: capabilityKey})
		if err != nil {
			return RuntimeModelInfo{}, status.Error(codes.Unavailable, "AI entitlement check failed")
		}
		if s.billingRequired && access.GetEnforcementMode() != pb.BillingEnforcementMode_BILLING_ENFORCEMENT_MODE_ENFORCE {
			return RuntimeModelInfo{}, status.Error(codes.Unavailable, "AI billing enforcement mode mismatch")
		}
		if !access.GetAllowed() {
			return RuntimeModelInfo{}, status.Error(codes.ResourceExhausted, access.GetReason())
		}
		versionID = access.GetCapabilityVersionId()
	} else if s != nil && s.billingRequired {
		return RuntimeModelInfo{}, status.Error(codes.Unavailable, "AI billing service is unavailable")
	}

	resolver, ok := s.store.(capabilityRuntimeModelResolver)
	if !ok {
		// Compatibility for isolated unit tests and non-native stores. Production
		// NativeStore always implements the capability resolver.
		return s.resolveRuntimeModelInfo(ctx, requestedModelID), nil
	}
	resolved, err := resolver.ResolveCapabilityRuntimeModel(ctx, capabilityKey, audience, versionID, requestedModelID)
	if err != nil {
		return RuntimeModelInfo{}, status.Error(codes.FailedPrecondition, err.Error())
	}
	return RuntimeModelInfo{
		ID:                     resolved.EffectiveModelID,
		Name:                   resolved.ModelName,
		ProviderName:           resolved.ProviderName,
		ContextWindowTokens:    resolved.ContextWindowTokens,
		MaxOutputTokens:        resolved.MaxOutputTokens,
		RequestedModelID:       resolved.RequestedModelID,
		FallbackReason:         resolved.FallbackReason,
		CapabilityVersionID:    resolved.CapabilityVersionID,
		CapabilitySnapshotHash: resolved.CapabilitySnapshotHash,
		ConfigurationRefs:      resolved.ConfigurationRefs,
	}, nil
}

func (s *nativeAIService) reserveAIBilling(ctx context.Context, kind billingOwnerKind, userID int64, capability, operation, provider, modelName string, inputChars int, runtimeModel RuntimeModelInfo) (context.Context, error) {
	if s == nil || s.billing == nil {
		if s != nil && s.billingRequired {
			return ctx, status.Error(codes.Unavailable, "AI billing service is unavailable")
		}
		return ctx, nil
	}
	owner := &pb.BillingOwner{}
	switch kind {
	case billingOwnerTenant:
		owner.Type = pb.BillingOwnerType_BILLING_OWNER_TYPE_TENANT
		owner.Id = platformmetadata.GetAuthTenantID(ctx)
	case billingOwnerUser:
		owner.Type = pb.BillingOwnerType_BILLING_OWNER_TYPE_USER
		owner.Id = userID
	}
	if owner.Id <= 0 {
		if s.billingRequired {
			return ctx, status.Error(codes.FailedPrecondition, "AI billing owner is missing")
		}
		return ctx, nil
	}
	requestID := strings.TrimSpace(platformmetadata.GetRequestID(ctx))
	if requestID == "" {
		requestID = fmt.Sprintf("ai-%d-%d", userID, time.Now().UnixNano())
	}
	// Billing owns the plan-specific single-run ceiling. Passing zero asks it to
	// resolve ai.single_run.max_credits for the effective owner instead of
	// duplicating commercial policy in every AI runtime.
	estimatedCredits := int64(0)
	response, err := s.billing.ReserveAIUsage(ctx, &pb.ReserveAIUsageRequest{
		Owner: owner, UserId: userID, Capability: capability, Operation: operation,
		ProviderKey: provider, ModelKey: modelName, EstimatedCredits: estimatedCredits,
		IdempotencyKey: requestID + ":" + operation, TtlSeconds: reservationTTLSeconds(operation),
	})
	if err != nil {
		if s.billingRequired {
			return ctx, status.Error(codes.Unavailable, "AI billing reservation failed")
		}
		return ctx, nil
	}
	if !response.GetAllowed() {
		return ctx, status.Error(codes.ResourceExhausted, response.GetReason())
	}
	if s.billingRequired && response.GetEnforcementMode() != pb.BillingEnforcementMode_BILLING_ENFORCEMENT_MODE_ENFORCE {
		return ctx, status.Error(codes.Unavailable, "AI billing enforcement mode mismatch")
	}
	if response.GetReservationNo() == "" {
		return ctx, nil
	}
	if outbox, ok := s.store.(billingMeterStore); ok {
		persistCtx, cancel := billingOutboxContext(ctx)
		defer cancel()
		record := BillingOutboxReservation{
			ReservationNo: response.GetReservationNo(), OwnerID: owner.Id, UserID: userID,
			Capability: capability, Operation: operation, ProviderKey: provider, ModelKey: modelName,
		}
		if owner.Type == pb.BillingOwnerType_BILLING_OWNER_TYPE_TENANT {
			record.OwnerType = "tenant"
		} else {
			record.OwnerType = "user"
		}
		if persistErr := outbox.CreateBillingOutboxReservation(persistCtx, record); persistErr != nil {
			platformobservability.DefaultMetrics.RecordBillingEvent("reservation_outbox", "error")
			platformlogger.L().Error("persist AI billing reservation outbox failed",
				zap.String("reservation_no", response.GetReservationNo()),
				zap.String("owner_type", record.OwnerType),
				zap.Int64("owner_id", record.OwnerID),
				zap.String("capability", record.Capability),
				zap.String("operation", record.Operation),
				zap.Error(persistErr),
			)
			_, _ = s.billing.CancelAIUsage(persistCtx, &pb.CancelAIUsageRequest{ReservationNo: response.GetReservationNo(), Reason: "reservation_outbox_failed", IdempotencyKey: response.GetReservationNo() + ":outbox-failed"})
			return ctx, status.Error(codes.Unavailable, "AI billing reservation persistence failed")
		} else {
			platformobservability.DefaultMetrics.RecordBillingEvent("reservation_outbox", "created")
		}
	} else {
		persistCtx, cancel := billingOutboxContext(ctx)
		defer cancel()
		_, _ = s.billing.CancelAIUsage(persistCtx, &pb.CancelAIUsageRequest{ReservationNo: response.GetReservationNo(), Reason: "reservation_outbox_unavailable", IdempotencyKey: response.GetReservationNo() + ":outbox-unavailable"})
		return ctx, status.Error(codes.Unavailable, "AI billing settlement outbox is unavailable")
	}
	return context.WithValue(ctx, billingReservationContextKey{}, &billingReservationState{reservationNo: response.GetReservationNo(), runtimeModel: runtimeModel}), nil
}

func reservationTTLSeconds(operation string) int64 {
	if strings.Contains(operation, "agent") {
		return int64((2 * time.Hour) / time.Second)
	}
	return int64((10 * time.Minute) / time.Second)
}

func (s *nativeAIService) bestEffortMeterUsage(ctx context.Context, row UsageAuditRow) {
	state, _ := ctx.Value(billingReservationContextKey{}).(*billingReservationState)
	if state == nil || s == nil || s.billing == nil {
		return
	}
	state.mu.Lock()
	reservationNo := state.reservationNo
	runtimeModel := state.runtimeModel
	state.mu.Unlock()
	promptTokens := row.PromptTokens
	completionTokens := row.CompletionTokens
	estimated := false
	if promptTokens <= 0 && completionTokens <= 0 {
		promptTokens = row.TokenUsageTotal
		if promptTokens <= 0 {
			promptTokens = row.EstimatedTokens
		}
		estimated = true
	}
	if promptTokens <= 0 && completionTokens <= 0 {
		s.cancelUnsettledBilling(ctx, "provider_consumed_no_tokens")
		return
	}
	state.mu.Lock()
	if state.done || state.processing {
		state.mu.Unlock()
		return
	}
	state.processing = true
	state.settlementRequired = true
	state.mu.Unlock()
	metadata, _ := json.Marshal(map[string]any{
		"requested_model_id":       runtimeModel.RequestedModelID,
		"effective_model_id":       runtimeModel.ID,
		"model_fallback_reason":    runtimeModel.FallbackReason,
		"capability_version_id":    runtimeModel.CapabilityVersionID,
		"capability_snapshot_hash": runtimeModel.CapabilitySnapshotHash,
	})
	request := &pb.SettleAIUsageRequest{
		ReservationNo: reservationNo, IdempotencyKey: reservationNo + ":settle",
		ProviderUsages: []*pb.AIProviderUsage{{
			ProviderCallSeq: 1, ProviderKey: row.Provider, ModelKey: row.Model,
			InputTokens: int64(max(promptTokens, 0)), OutputTokens: int64(max(completionTokens, 0)),
			CachedInputTokens: int64(max(row.CachedInputTokens, 0)), ProviderRequestId: row.RequestID,
			Estimated: estimated, OccurredAtUnixMs: time.Now().UnixMilli(), MetadataJson: string(metadata),
		}},
	}
	persistCtx, cancel := billingOutboxContext(ctx)
	defer cancel()
	outbox, ok := s.store.(billingMeterStore)
	if !ok {
		state.mu.Lock()
		state.processing = false
		state.mu.Unlock()
		platformobservability.DefaultMetrics.RecordBillingEvent("settlement_outbox", "unavailable")
		platformlogger.L().Error("AI billing settlement outbox is unavailable", zap.String("reservation_no", reservationNo))
		return
	}
	if err := outbox.QueueBillingOutboxSettlement(persistCtx, request); err != nil {
		state.mu.Lock()
		state.processing = false
		state.mu.Unlock()
		platformobservability.DefaultMetrics.RecordBillingEvent("settlement_outbox", "error")
		platformlogger.L().Error("queue AI billing settlement failed", zap.String("reservation_no", reservationNo), zap.Error(err))
		return
	}
	state.mu.Lock()
	state.done = true
	state.processing = false
	state.mu.Unlock()
	response, err := s.billing.SettleAIUsage(persistCtx, request)
	if err != nil {
		platformobservability.DefaultMetrics.RecordBillingEvent("settlement", "retry_pending")
		platformlogger.L().Warn("AI billing settlement deferred for retry", zap.String("reservation_no", reservationNo), zap.Error(err))
		return
	}
	if err := outbox.MarkBillingOutboxSettled(persistCtx, reservationNo); err != nil {
		platformlogger.L().Warn("mark AI billing settlement completed failed", zap.String("reservation_no", reservationNo), zap.Error(err))
	}
	platformobservability.DefaultMetrics.RecordBillingEvent("settlement", "settled")
	_ = response
}

func (s *nativeAIService) finalizeStructuredBilling(ctx context.Context, runtimeModel RuntimeModelInfo, collector *recruitingruntime.BillingUsageCollector) {
	if collector == nil {
		s.cancelUnsettledBilling(ctx, "structured_runtime_consumed_no_tokens")
		return
	}
	items := collector.Usages()
	if len(items) == 0 {
		s.cancelUnsettledBilling(ctx, "structured_runtime_consumed_no_tokens")
		return
	}
	state, _ := ctx.Value(billingReservationContextKey{}).(*billingReservationState)
	if state == nil || s == nil || s.billing == nil {
		return
	}
	state.mu.Lock()
	reservationNo := state.reservationNo
	state.mu.Unlock()
	metadata, _ := json.Marshal(map[string]any{
		"requested_model_id": runtimeModel.RequestedModelID, "effective_model_id": runtimeModel.ID,
		"model_fallback_reason": runtimeModel.FallbackReason, "capability_version_id": runtimeModel.CapabilityVersionID,
		"capability_snapshot_hash": runtimeModel.CapabilitySnapshotHash,
	})
	providerUsages := make([]*pb.AIProviderUsage, 0, len(items))
	for _, item := range items {
		provider := strings.TrimSpace(item.ProviderKey)
		if provider == "" {
			provider = runtimeModel.ProviderName
		}
		modelName := strings.TrimSpace(item.ModelName)
		if modelName == "" {
			modelName = runtimeModel.Name
		}
		inputTokens := item.EstimatedInputTokens
		outputTokens := item.EstimatedOutputTokens
		estimated := item.TokenUsage == nil
		cachedTokens := 0
		if item.TokenUsage != nil {
			inputTokens = item.TokenUsage.PromptTokens
			outputTokens = item.TokenUsage.CompletionTokens
			cachedTokens = item.TokenUsage.PromptTokenDetails.CachedTokens
		}
		providerUsages = append(providerUsages, &pb.AIProviderUsage{
			ProviderCallSeq: int32(len(providerUsages) + 1), ProviderKey: provider, ModelKey: modelName,
			InputTokens: int64(max(inputTokens, 0)), OutputTokens: int64(max(outputTokens, 0)),
			CachedInputTokens: int64(max(cachedTokens, 0)), Estimated: estimated,
			OccurredAtUnixMs: time.Now().UnixMilli(), MetadataJson: string(metadata),
		})
	}
	if len(providerUsages) == 0 {
		s.cancelUnsettledBilling(ctx, "structured_runtime_consumed_no_tokens")
		return
	}
	state.mu.Lock()
	if state.done || state.processing {
		state.mu.Unlock()
		return
	}
	state.processing = true
	state.settlementRequired = true
	state.mu.Unlock()
	request := &pb.SettleAIUsageRequest{ReservationNo: reservationNo, IdempotencyKey: reservationNo + ":settle", ProviderUsages: providerUsages}
	persistCtx, cancel := billingOutboxContext(ctx)
	defer cancel()
	outbox, ok := s.store.(billingMeterStore)
	if !ok {
		state.mu.Lock()
		state.processing = false
		state.mu.Unlock()
		platformobservability.DefaultMetrics.RecordBillingEvent("structured_settlement_outbox", "unavailable")
		return
	}
	if err := outbox.QueueBillingOutboxSettlement(persistCtx, request); err != nil {
		state.mu.Lock()
		state.processing = false
		state.mu.Unlock()
		platformobservability.DefaultMetrics.RecordBillingEvent("structured_settlement_outbox", "error")
		platformlogger.L().Error("queue structured AI billing settlement failed", zap.String("reservation_no", reservationNo), zap.Error(err))
		return
	}
	state.mu.Lock()
	state.done = true
	state.processing = false
	state.mu.Unlock()
	if _, err := s.billing.SettleAIUsage(persistCtx, request); err != nil {
		platformobservability.DefaultMetrics.RecordBillingEvent("structured_settlement", "retry_pending")
		return
	}
	if err := outbox.MarkBillingOutboxSettled(persistCtx, reservationNo); err != nil {
		platformlogger.L().Warn("mark structured AI billing settlement completed failed", zap.String("reservation_no", reservationNo), zap.Error(err))
	}
	platformobservability.DefaultMetrics.RecordBillingEvent("structured_settlement", "settled")
}

func (s *nativeAIService) cancelUnsettledBilling(ctx context.Context, reason string) {
	state, _ := ctx.Value(billingReservationContextKey{}).(*billingReservationState)
	if state == nil || s == nil || s.billing == nil {
		return
	}
	state.mu.Lock()
	if state.done || state.processing || state.settlementRequired {
		state.mu.Unlock()
		return
	}
	reservationNo := state.reservationNo
	state.processing = true
	state.mu.Unlock()
	request := &pb.CancelAIUsageRequest{ReservationNo: reservationNo, Reason: reason, IdempotencyKey: reservationNo + ":cancel"}
	persistCtx, cancel := billingOutboxContext(ctx)
	defer cancel()
	outbox, ok := s.store.(billingMeterStore)
	if !ok {
		state.mu.Lock()
		state.processing = false
		state.mu.Unlock()
		platformobservability.DefaultMetrics.RecordBillingEvent("cancellation_outbox", "unavailable")
		return
	}
	if err := outbox.QueueBillingOutboxCancellation(persistCtx, request); err != nil {
		state.mu.Lock()
		state.processing = false
		state.mu.Unlock()
		platformobservability.DefaultMetrics.RecordBillingEvent("cancellation_outbox", "error")
		return
	}
	state.mu.Lock()
	state.done = true
	state.processing = false
	state.mu.Unlock()
	if _, err := s.billing.CancelAIUsage(persistCtx, request); err != nil {
		platformobservability.DefaultMetrics.RecordBillingEvent("cancellation", "retry_pending")
		return
	}
	if err := outbox.MarkBillingOutboxCancelled(persistCtx, reservationNo); err != nil {
		platformlogger.L().Warn("mark AI billing cancellation completed failed", zap.String("reservation_no", reservationNo), zap.Error(err))
	}
	platformobservability.DefaultMetrics.RecordBillingEvent("cancellation", "cancelled")
}

func billingOutboxContext(ctx context.Context) (context.Context, context.CancelFunc) {
	base := context.Background()
	if ctx != nil {
		base = context.WithoutCancel(ctx)
	}
	return context.WithTimeout(base, 5*time.Second)
}
