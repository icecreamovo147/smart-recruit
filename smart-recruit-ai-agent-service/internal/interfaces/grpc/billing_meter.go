package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	platformmetadata "smart-recruit-platform-go/metadata"
	"smart-recruit-proto/recruitment/pb"
)

type billingOwnerKind int

const (
	billingOwnerTenant billingOwnerKind = iota + 1
	billingOwnerUser
)

type billingReservationContextKey struct{}

type billingReservationState struct {
	mu            sync.Mutex
	reservationNo string
	runtimeModel  RuntimeModelInfo
	done          bool
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
	// Reserve the configured single-run ceiling before invoking a provider. The
	// unused portion is released during settlement; this keeps enforce mode
	// prepaid even when the provider emits more output than expected.
	estimatedCredits := int64(20)
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
	if response.GetReservationNo() == "" {
		return ctx, nil
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
	if state.done {
		state.mu.Unlock()
		return
	}
	state.done = true
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
		_, _ = s.billing.CancelAIUsage(ctx, &pb.CancelAIUsageRequest{ReservationNo: reservationNo, Reason: "provider_consumed_no_tokens", IdempotencyKey: reservationNo + ":empty"})
		return
	}
	metadata, _ := json.Marshal(map[string]any{
		"requested_model_id":       runtimeModel.RequestedModelID,
		"effective_model_id":       runtimeModel.ID,
		"model_fallback_reason":    runtimeModel.FallbackReason,
		"capability_version_id":    runtimeModel.CapabilityVersionID,
		"capability_snapshot_hash": runtimeModel.CapabilitySnapshotHash,
	})
	_, _ = s.billing.SettleAIUsage(ctx, &pb.SettleAIUsageRequest{
		ReservationNo: reservationNo, IdempotencyKey: reservationNo + ":settle",
		ProviderUsages: []*pb.AIProviderUsage{{
			ProviderCallSeq: 1, ProviderKey: row.Provider, ModelKey: row.Model,
			InputTokens: int64(max(promptTokens, 0)), OutputTokens: int64(max(completionTokens, 0)),
			CachedInputTokens: int64(max(row.CachedInputTokens, 0)), ProviderRequestId: row.RequestID,
			Estimated: estimated, OccurredAtUnixMs: time.Now().UnixMilli(), MetadataJson: string(metadata),
		}},
	})
}

func (s *nativeAIService) cancelUnsettledBilling(ctx context.Context, reason string) {
	state, _ := ctx.Value(billingReservationContextKey{}).(*billingReservationState)
	if state == nil || s == nil || s.billing == nil {
		return
	}
	state.mu.Lock()
	if state.done {
		state.mu.Unlock()
		return
	}
	state.done = true
	reservationNo := state.reservationNo
	state.mu.Unlock()
	_, _ = s.billing.CancelAIUsage(ctx, &pb.CancelAIUsageRequest{ReservationNo: reservationNo, Reason: reason, IdempotencyKey: reservationNo + ":cancel"})
}
