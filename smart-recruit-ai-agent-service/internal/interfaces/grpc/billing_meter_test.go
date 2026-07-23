package grpc

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"

	platformmetadata "smart-recruit-platform-go/metadata"
	"smart-recruit-proto/recruitment/pb"
)

type meterBillingClient struct {
	pb.BillingServiceClient
	mode          pb.BillingEnforcementMode
	reserveCalls  int
	settleCalls   int
	cancelCalls   int
	reserveReq    *pb.ReserveAIUsageRequest
	settlementReq *pb.SettleAIUsageRequest
}

func (c *meterBillingClient) ReserveAIUsage(_ context.Context, request *pb.ReserveAIUsageRequest, _ ...grpc.CallOption) (*pb.ReserveAIUsageResponse, error) {
	c.reserveCalls++
	c.reserveReq = request
	return &pb.ReserveAIUsageResponse{
		Allowed: true, ReservationNo: "11111111-1111-1111-1111-111111111111",
		ReservedCredits: 75, EnforcementMode: c.mode,
	}, nil
}

func (c *meterBillingClient) SettleAIUsage(_ context.Context, request *pb.SettleAIUsageRequest, _ ...grpc.CallOption) (*pb.SettleAIUsageResponse, error) {
	c.settleCalls++
	c.settlementReq = request
	return &pb.SettleAIUsageResponse{Code: 0, ChargedCredits: 3}, nil
}

func (c *meterBillingClient) CancelAIUsage(context.Context, *pb.CancelAIUsageRequest, ...grpc.CallOption) (*pb.CancelAIUsageResponse, error) {
	c.cancelCalls++
	return &pb.CancelAIUsageResponse{Code: 0}, nil
}

func TestBillingMeterUsesServerResolvedLimitAndDurableSettlement(t *testing.T) {
	store := &fakeAIStore{}
	billing := &meterBillingClient{mode: pb.BillingEnforcementMode_BILLING_ENFORCEMENT_MODE_ENFORCE}
	service := &nativeAIService{store: store, billing: billing, billingRequired: true}
	ctx := platformmetadata.WithTenantActor(context.Background(), platformmetadata.TenantContext{TenantID: 9, UserID: 7, AccountType: "staff"})
	ctx = context.WithValue(ctx, platformmetadata.KeyRequestID, "request-1")
	runtimeModel := RuntimeModelInfo{ID: 2, Name: "model-a", ProviderName: "provider-a", CapabilityVersionID: 8}
	reserved, err := service.reserveAIBilling(ctx, billingOwnerTenant, 7, "ai.chat", "hr_chat", "provider-a", "model-a", 100, runtimeModel)
	if err != nil {
		t.Fatal(err)
	}
	if billing.reserveReq.GetEstimatedCredits() != 0 || store.billingOutboxRecord == nil || store.billingOutboxStatus != "reserved" {
		t.Fatalf("reserve=%+v outbox=%+v status=%s", billing.reserveReq, store.billingOutboxRecord, store.billingOutboxStatus)
	}
	service.bestEffortMeterUsage(reserved, UsageAuditRow{Provider: "provider-a", Model: "model-a", PromptTokens: 100, CompletionTokens: 20})
	if billing.settleCalls != 1 || store.billingOutboxStatus != "settled" || billing.settlementReq.GetProviderUsages()[0].GetInputTokens() != 100 {
		t.Fatalf("settle calls=%d status=%s request=%+v", billing.settleCalls, store.billingOutboxStatus, billing.settlementReq)
	}
}

func TestBillingMeterRejectsEnforcementModeMismatch(t *testing.T) {
	store := &fakeAIStore{}
	billing := &meterBillingClient{mode: pb.BillingEnforcementMode_BILLING_ENFORCEMENT_MODE_SHADOW}
	service := &nativeAIService{store: store, billing: billing, billingRequired: true}
	ctx := platformmetadata.WithTenantActor(context.Background(), platformmetadata.TenantContext{TenantID: 9, UserID: 7, AccountType: "staff"})
	_, err := service.reserveAIBilling(ctx, billingOwnerTenant, 7, "ai.chat", "hr_chat", "provider-a", "model-a", 0, RuntimeModelInfo{})
	if err == nil {
		t.Fatal("expected enforcement mode mismatch")
	}
	if store.billingOutboxRecord != nil {
		t.Fatal("mode mismatch must fail before creating the outbox record")
	}
}

func TestBillingMeterNeverCancelsConsumedUsageWhenSettlementPersistenceFails(t *testing.T) {
	store := &fakeAIStore{billingSettlementErr: errors.New("database unavailable")}
	billing := &meterBillingClient{mode: pb.BillingEnforcementMode_BILLING_ENFORCEMENT_MODE_ENFORCE}
	service := &nativeAIService{store: store, billing: billing, billingRequired: true}
	ctx := platformmetadata.WithTenantActor(context.Background(), platformmetadata.TenantContext{TenantID: 9, UserID: 7, AccountType: "staff"})
	reserved, err := service.reserveAIBilling(ctx, billingOwnerTenant, 7, "ai.chat", "hr_chat", "provider-a", "model-a", 0, RuntimeModelInfo{})
	if err != nil {
		t.Fatal(err)
	}
	service.bestEffortMeterUsage(reserved, UsageAuditRow{Provider: "provider-a", Model: "model-a", PromptTokens: 10})
	service.cancelUnsettledBilling(reserved, "runtime_completed")
	if billing.settleCalls != 0 || billing.cancelCalls != 0 || store.billingOutboxStatus != "reserved" {
		t.Fatalf("settle=%d cancel=%d status=%s", billing.settleCalls, billing.cancelCalls, store.billingOutboxStatus)
	}
}
