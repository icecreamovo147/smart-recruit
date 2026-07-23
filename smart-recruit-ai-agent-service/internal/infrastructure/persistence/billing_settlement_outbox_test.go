package persistence

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	aiagentgrpc "smart-recruit-ai-agent-service/internal/interfaces/grpc"
	"smart-recruit-proto/recruitment/pb"
)

type outboxBillingClient struct {
	pb.BillingServiceClient
	settleErr  error
	cancelErr  error
	settleCall int
	cancelCall int
}

func (c *outboxBillingClient) SettleAIUsage(context.Context, *pb.SettleAIUsageRequest, ...grpc.CallOption) (*pb.SettleAIUsageResponse, error) {
	c.settleCall++
	if c.settleErr != nil {
		return nil, c.settleErr
	}
	return &pb.SettleAIUsageResponse{Code: 0}, nil
}

func (c *outboxBillingClient) CancelAIUsage(context.Context, *pb.CancelAIUsageRequest, ...grpc.CallOption) (*pb.CancelAIUsageResponse, error) {
	c.cancelCall++
	if c.cancelErr != nil {
		return nil, c.cancelErr
	}
	return &pb.CancelAIUsageResponse{Code: 0}, nil
}

func newBillingOutboxTestStore(t *testing.T) *NativeStore {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&billingSettlementOutboxRow{}); err != nil {
		t.Fatal(err)
	}
	return NewNativeStore(db)
}

func seedBillingOutboxReservation(t *testing.T, store *NativeStore, reservationNo string) {
	t.Helper()
	err := store.CreateBillingOutboxReservation(context.Background(), aiagentgrpc.BillingOutboxReservation{
		ReservationNo: reservationNo, OwnerType: "tenant", OwnerID: 9, UserID: 7,
		Capability: "ai.chat", Operation: "hr_chat", ProviderKey: "provider", ModelKey: "model",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBillingSettlementOutboxRetriesAndSettles(t *testing.T) {
	store := newBillingOutboxTestStore(t)
	seedBillingOutboxReservation(t, store, "11111111-1111-1111-1111-111111111111")
	request := &pb.SettleAIUsageRequest{
		ReservationNo: "11111111-1111-1111-1111-111111111111", IdempotencyKey: "settle-1",
		ProviderUsages: []*pb.AIProviderUsage{{ProviderCallSeq: 1, ProviderKey: "provider", ModelKey: "model", InputTokens: 10}},
	}
	if err := store.QueueBillingOutboxSettlement(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	client := &outboxBillingClient{settleErr: errors.New("billing unavailable")}
	if err := store.processBillingSettlementOutbox(context.Background(), client, 10); err != nil {
		t.Fatal(err)
	}
	var row billingSettlementOutboxRow
	if err := store.db.Where("reservation_no = ?", request.ReservationNo).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != billingOutboxPendingSettle || row.RetryCount != 1 || client.settleCall != 1 {
		t.Fatalf("after failure row=%+v calls=%d", row, client.settleCall)
	}
	client.settleErr = nil
	now := time.Now().UTC().Add(-time.Second)
	if err := store.db.Model(&billingSettlementOutboxRow{}).Where("id = ?", row.ID).Update("next_attempt_at", now).Error; err != nil {
		t.Fatal(err)
	}
	if err := store.processBillingSettlementOutbox(context.Background(), client, 10); err != nil {
		t.Fatal(err)
	}
	if err := store.db.Where("id = ?", row.ID).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != billingOutboxSettled || row.CompletedAt == nil || client.settleCall != 2 {
		t.Fatalf("after retry row=%+v calls=%d", row, client.settleCall)
	}
}

func TestBillingSettlementOutboxDeliversCancellation(t *testing.T) {
	store := newBillingOutboxTestStore(t)
	reservationNo := "22222222-2222-2222-2222-222222222222"
	seedBillingOutboxReservation(t, store, reservationNo)
	if err := store.QueueBillingOutboxCancellation(context.Background(), &pb.CancelAIUsageRequest{
		ReservationNo: reservationNo, Reason: "no usage", IdempotencyKey: "cancel-1",
	}); err != nil {
		t.Fatal(err)
	}
	client := &outboxBillingClient{}
	if err := store.processBillingSettlementOutbox(context.Background(), client, 10); err != nil {
		t.Fatal(err)
	}
	var row billingSettlementOutboxRow
	if err := store.db.Where("reservation_no = ?", reservationNo).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != billingOutboxCancelled || client.cancelCall != 1 {
		t.Fatalf("row=%+v calls=%d", row, client.cancelCall)
	}
}

func TestBillingSettlementOutboxDoesNotReplaceSettlementWithCancellation(t *testing.T) {
	store := newBillingOutboxTestStore(t)
	reservationNo := "33333333-3333-3333-3333-333333333333"
	seedBillingOutboxReservation(t, store, reservationNo)
	if err := store.QueueBillingOutboxSettlement(context.Background(), &pb.SettleAIUsageRequest{
		ReservationNo: reservationNo, IdempotencyKey: "settle-3",
		ProviderUsages: []*pb.AIProviderUsage{{ProviderCallSeq: 1, ProviderKey: "provider", ModelKey: "model", InputTokens: 1}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.QueueBillingOutboxCancellation(context.Background(), &pb.CancelAIUsageRequest{
		ReservationNo: reservationNo, Reason: "late cancellation", IdempotencyKey: "cancel-3",
	}); err == nil {
		t.Fatal("cancellation must not overwrite pending consumed usage")
	}
	var row billingSettlementOutboxRow
	if err := store.db.Where("reservation_no = ?", reservationNo).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != billingOutboxPendingSettle {
		t.Fatalf("status=%s", row.Status)
	}
}

func TestBillingSettlementOutboxRejectsConcurrentDuplicateInvocation(t *testing.T) {
	store := newBillingOutboxTestStore(t)
	reservationNo := "44444444-4444-4444-4444-444444444444"
	seedBillingOutboxReservation(t, store, reservationNo)
	err := store.CreateBillingOutboxReservation(context.Background(), aiagentgrpc.BillingOutboxReservation{
		ReservationNo: reservationNo, OwnerType: "tenant", OwnerID: 9, UserID: 7,
		Capability: "ai.chat", Operation: "hr_chat", ProviderKey: "provider", ModelKey: "model",
	})
	if err == nil {
		t.Fatal("duplicate invocation must be rejected while the reservation lease is active")
	}
}
