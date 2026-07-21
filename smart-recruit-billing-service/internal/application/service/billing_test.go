package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"smart-recruit-billing-service/internal/domain/model"
	"smart-recruit-billing-service/internal/domain/repository"
)

type fakePolicy struct {
	enabled bool
	limit   uint64
}

func (p fakePolicy) Enabled(context.Context, model.Owner, string) (bool, error) {
	return p.enabled, nil
}

func (p fakePolicy) ReservationCreditLimit(context.Context, model.Owner) (uint64, error) {
	return p.limit, nil
}

type fakeRepository struct {
	balance      model.Balance
	rate         model.RateCard
	reservation  model.Reservation
	createCalls  int
	settleUsages []model.ProviderUsage
	rateErr      error
}

func (r *fakeRepository) EnsureMonthlyGrant(context.Context, model.Owner, time.Time) error {
	return nil
}

func (r *fakeRepository) Balance(context.Context, model.Owner, time.Time) (model.Balance, error) {
	return r.balance, nil
}
func (r *fakeRepository) CurrentRate(context.Context, string, string, time.Time) (model.RateCard, error) {
	return r.rate, r.rateErr
}
func (r *fakeRepository) CreateReservation(_ context.Context, value model.Reservation) (model.Reservation, model.Balance, bool, error) {
	r.createCalls++
	if value.Mode == model.ModeEnforce && r.balance.AvailableCredits < value.ReservedCredits {
		return model.Reservation{}, r.balance, false, repository.ErrInsufficientCredits
	}
	if r.reservation.No != "" {
		return r.reservation, r.balance, true, nil
	}
	r.reservation = value
	return value, r.balance, false, nil
}
func (r *fakeRepository) SettleReservation(_ context.Context, _ string, usages []model.ProviderUsage, _ string, _ time.Time) (model.Settlement, error) {
	r.settleUsages = usages
	var result model.Settlement
	for _, usage := range usages {
		result.ChargedCredits += usage.Credits
		result.SupplierCostMicros += usage.SupplierCostMicros
	}
	return result, nil
}
func (r *fakeRepository) CancelReservation(context.Context, string, string, string, time.Time) (model.Cancellation, error) {
	return model.Cancellation{}, nil
}

func TestShadowModeRecordsReservationWhenBalanceIsEmpty(t *testing.T) {
	repo := &fakeRepository{}
	billing, err := NewBilling(repo, fakePolicy{enabled: true}, model.ModeShadow)
	if err != nil {
		t.Fatal(err)
	}
	billing.now = func() time.Time { return time.Unix(1000, 0) }
	result, err := billing.Reserve(context.Background(), ReserveCommand{
		Owner: model.Owner{Type: model.OwnerTenant, ID: 9}, UserID: 7, Capability: "ai.chat.enabled",
		Operation: "chat", ProviderKey: "openai", ModelKey: "gpt-test", EstimatedCredits: 5, IdempotencyKey: "request-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Allowed || repo.createCalls != 1 {
		t.Fatalf("shadow reservation = %+v, calls = %d", result, repo.createCalls)
	}
	if got := result.Reservation.ExpiresAt.Sub(time.Unix(1000, 0)); got != 10*time.Minute {
		t.Fatalf("TTL = %s", got)
	}
}

func TestEnforceModeUsesRemainingBalanceBelowSingleRunCeiling(t *testing.T) {
	repo := &fakeRepository{balance: model.Balance{AvailableCredits: 2}}
	billing, err := NewBilling(repo, fakePolicy{enabled: true, limit: 10}, model.ModeEnforce)
	if err != nil {
		t.Fatal(err)
	}
	result, err := billing.Reserve(context.Background(), ReserveCommand{
		Owner: model.Owner{Type: model.OwnerUser, ID: 3}, UserID: 3, Capability: "ai.chat.enabled",
		Operation: "chat", EstimatedCredits: 3, IdempotencyKey: "request-2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Allowed || result.Reservation.ReservedCredits != 2 {
		t.Fatalf("result = %+v", result)
	}
	if repo.createCalls != 1 {
		t.Fatalf("create calls = %d", repo.createCalls)
	}
}

func TestEnforceModeRejectsZeroBalanceBeforeRepositoryWrite(t *testing.T) {
	repo := &fakeRepository{}
	billing, err := NewBilling(repo, fakePolicy{enabled: true, limit: 10}, model.ModeEnforce)
	if err != nil {
		t.Fatal(err)
	}
	result, err := billing.Reserve(context.Background(), ReserveCommand{
		Owner: model.Owner{Type: model.OwnerUser, ID: 3}, UserID: 3, Capability: "ai.chat.enabled",
		Operation: "chat", IdempotencyKey: "request-zero",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Allowed || result.Reason != "insufficient_credits" || repo.createCalls != 0 {
		t.Fatalf("result = %+v, create calls = %d", result, repo.createCalls)
	}
}

func TestReserveUsesPlanSingleRunLimit(t *testing.T) {
	repo := &fakeRepository{balance: model.Balance{AvailableCredits: 500}}
	billing, err := NewBilling(repo, fakePolicy{enabled: true, limit: 75}, model.ModeEnforce)
	if err != nil {
		t.Fatal(err)
	}
	result, err := billing.Reserve(context.Background(), ReserveCommand{
		Owner: model.Owner{Type: model.OwnerTenant, ID: 9}, UserID: 7, Capability: "ai.chat",
		Operation: "hr_chat", IdempotencyKey: "request-limit",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Allowed || result.Reservation.ReservedCredits != 75 {
		t.Fatalf("result = %+v", result)
	}
}

func TestEnforceModeRejectsUnpricedModelBeforeReservation(t *testing.T) {
	repo := &fakeRepository{balance: model.Balance{AvailableCredits: 100}, rateErr: errors.New("rate not found")}
	billing, err := NewBilling(repo, fakePolicy{enabled: true, limit: 10}, model.ModeEnforce)
	if err != nil {
		t.Fatal(err)
	}
	_, err = billing.Reserve(context.Background(), ReserveCommand{
		Owner: model.Owner{Type: model.OwnerTenant, ID: 9}, UserID: 7, Capability: "ai.chat",
		Operation: "hr_chat", ProviderKey: "openai", ModelKey: "unpriced", IdempotencyKey: "request-unpriced",
	})
	if err == nil || repo.createCalls != 0 {
		t.Fatalf("err = %v, create calls = %d", err, repo.createCalls)
	}
}

func TestSettlePricesEveryProviderCall(t *testing.T) {
	repo := &fakeRepository{rate: model.RateCard{InputMicrosPer1K: 1000, OutputMicrosPer1K: 2000, CachedInputMicrosPer1K: 200, CreditMicros: 1000}}
	billing, err := NewBilling(repo, fakePolicy{enabled: true}, model.ModeShadow)
	if err != nil {
		t.Fatal(err)
	}
	result, err := billing.Settle(context.Background(), "reservation-1", []model.ProviderUsage{{
		CallSequence: 1, ProviderKey: "openai", ModelKey: "gpt-test", InputTokens: 1000, OutputTokens: 500, CachedInputTokens: 1000,
	}}, "settle-1")
	if err != nil {
		t.Fatal(err)
	}
	if result.SupplierCostMicros != 2200 || result.ChargedCredits != 3 {
		t.Fatalf("settlement = %+v", result)
	}
	if len(repo.settleUsages) != 1 || repo.settleUsages[0].Credits != 3 {
		t.Fatalf("priced usages = %+v", repo.settleUsages)
	}
}

func TestNewBillingRejectsMissingDependencies(t *testing.T) {
	if _, err := NewBilling(nil, fakePolicy{}, model.ModeShadow); err == nil {
		t.Fatal("expected repository error")
	}
	if _, err := NewBilling(&fakeRepository{}, nil, model.ModeShadow); err == nil {
		t.Fatal("expected policy error")
	}
	if _, err := NewBilling(&fakeRepository{}, fakePolicy{}, "invalid"); err == nil {
		t.Fatal("expected mode error")
	}
}

var _ repository.BillingRepository = (*fakeRepository)(nil)
