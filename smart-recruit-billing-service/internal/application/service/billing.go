package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"smart-recruit-billing-service/internal/domain/model"
	"smart-recruit-billing-service/internal/domain/repository"
	"smart-recruit-platform-go/businessclock"
)

type AccessPolicy interface {
	Enabled(context.Context, model.Owner, string) (bool, error)
}

type Billing struct {
	repository repository.BillingRepository
	policy     AccessPolicy
	mode       model.EnforcementMode
	now        func() time.Time
}

func NewBilling(repo repository.BillingRepository, policy AccessPolicy, mode model.EnforcementMode) (*Billing, error) {
	if repo == nil {
		return nil, errors.New("billing repository is required")
	}
	if policy == nil {
		return nil, errors.New("AI access policy is required")
	}
	if mode != model.ModeShadow && mode != model.ModeEnforce {
		return nil, errors.New("billing mode must be shadow or enforce")
	}
	return &Billing{repository: repo, policy: policy, mode: mode, now: businessclock.Now}, nil
}

func (b *Billing) businessNow() time.Time { return b.now().In(businessclock.Location) }

type AccessDecision struct {
	Allowed             bool
	Reason              string
	Balance             model.Balance
	Mode                model.EnforcementMode
	CapabilityVersionID uint64
}

type capabilityVersionPolicy interface {
	ReleaseVersionID(context.Context, model.Owner, string) (uint64, error)
}

type reservationLimitPolicy interface {
	ReservationCreditLimit(context.Context, model.Owner) (uint64, error)
}

const defaultReservationCreditLimit uint64 = 20

func (b *Billing) CheckAccess(ctx context.Context, owner model.Owner, capability string, estimated uint64) (AccessDecision, error) {
	if err := owner.Validate(); err != nil {
		return AccessDecision{}, err
	}
	if strings.TrimSpace(capability) == "" {
		return AccessDecision{}, errors.New("AI capability is required")
	}
	if err := b.repository.EnsureMonthlyGrant(ctx, owner, b.businessNow()); err != nil {
		return AccessDecision{}, fmt.Errorf("provision monthly AI credits: %w", err)
	}
	enabled, err := b.policy.Enabled(ctx, owner, capability)
	if err != nil {
		return AccessDecision{}, fmt.Errorf("resolve AI entitlement: %w", err)
	}
	balance, err := b.repository.Balance(ctx, owner, b.businessNow())
	if err != nil {
		return AccessDecision{}, fmt.Errorf("load AI credit balance: %w", err)
	}
	decision := AccessDecision{Allowed: true, Balance: balance, Mode: b.mode}
	if versionPolicy, ok := b.policy.(capabilityVersionPolicy); ok {
		decision.CapabilityVersionID, err = versionPolicy.ReleaseVersionID(ctx, owner, capability)
		if err != nil {
			return AccessDecision{}, fmt.Errorf("resolve AI capability release: %w", err)
		}
	}
	if !enabled {
		decision.Allowed = b.mode == model.ModeShadow
		decision.Reason = "ai_capability_disabled"
	} else if balance.AvailableCredits < estimated {
		decision.Allowed = b.mode == model.ModeShadow
		decision.Reason = "insufficient_credits"
	}
	return decision, nil
}

type ReserveCommand struct {
	Owner            model.Owner
	UserID           uint64
	Capability       string
	Operation        string
	ProviderKey      string
	ModelKey         string
	EstimatedCredits uint64
	IdempotencyKey   string
	TTL              time.Duration
}

type ReserveResult struct {
	Reservation         model.Reservation
	Balance             model.Balance
	Allowed             bool
	Reason              string
	Existing            bool
	CapabilityVersionID uint64
}

func (b *Billing) Reserve(ctx context.Context, command ReserveCommand) (ReserveResult, error) {
	if command.TTL <= 0 {
		command.TTL = defaultTTL(command.Operation)
	}
	if command.TTL > 2*time.Hour {
		return ReserveResult{}, errors.New("reservation TTL exceeds two hours")
	}
	if command.EstimatedCredits == 0 {
		command.EstimatedCredits = defaultReservationCreditLimit
		if policy, ok := b.policy.(reservationLimitPolicy); ok {
			limit, limitErr := policy.ReservationCreditLimit(ctx, command.Owner)
			if limitErr != nil {
				return ReserveResult{}, fmt.Errorf("resolve AI single-run credit limit: %w", limitErr)
			}
			if limit > 0 {
				command.EstimatedCredits = limit
			}
		}
	}
	decision, err := b.CheckAccess(ctx, command.Owner, command.Capability, 0)
	if err != nil {
		return ReserveResult{}, err
	}
	if !decision.Allowed {
		return ReserveResult{Allowed: false, Reason: decision.Reason, Balance: decision.Balance, CapabilityVersionID: decision.CapabilityVersionID}, nil
	}
	if b.mode == model.ModeEnforce {
		// Fail before invoking the provider when the exact provider/model cannot
		// be priced. Otherwise usage could succeed but remain impossible to bill.
		if _, rateErr := b.repository.CurrentRate(ctx, strings.TrimSpace(command.ProviderKey), strings.TrimSpace(command.ModelKey), b.businessNow()); rateErr != nil {
			return ReserveResult{}, fmt.Errorf("load AI rate card for %s/%s: %w", command.ProviderKey, command.ModelKey, rateErr)
		}
		if decision.Balance.AvailableCredits == 0 {
			return ReserveResult{Allowed: false, Reason: "insufficient_credits", Balance: decision.Balance, CapabilityVersionID: decision.CapabilityVersionID}, nil
		}
		// A low balance may still fund a small run. Reserve the lower of the
		// commercial single-run ceiling and the remaining prepaid balance.
		if command.EstimatedCredits > decision.Balance.AvailableCredits {
			command.EstimatedCredits = decision.Balance.AvailableCredits
		}
	}
	now := b.businessNow()
	reservation := model.Reservation{
		No:              uuid.NewString(),
		Owner:           command.Owner,
		UserID:          command.UserID,
		Capability:      command.Capability,
		Operation:       strings.TrimSpace(command.Operation),
		ProviderKey:     strings.TrimSpace(command.ProviderKey),
		ModelKey:        strings.TrimSpace(command.ModelKey),
		IdempotencyKey:  strings.TrimSpace(command.IdempotencyKey),
		ReservedCredits: command.EstimatedCredits,
		Mode:            b.mode,
		Status:          "active",
		ExpiresAt:       now.Add(command.TTL),
		CreatedAt:       now,
	}
	if err := reservation.Validate(); err != nil {
		return ReserveResult{}, err
	}
	created, balance, existing, err := b.repository.CreateReservation(ctx, reservation)
	if errors.Is(err, repository.ErrInsufficientCredits) {
		return ReserveResult{Allowed: false, Reason: "insufficient_credits", Balance: balance}, nil
	}
	if err != nil {
		return ReserveResult{}, fmt.Errorf("reserve AI credits: %w", err)
	}
	return ReserveResult{Reservation: created, Balance: balance, Allowed: true, Existing: existing, CapabilityVersionID: decision.CapabilityVersionID}, nil
}

func (b *Billing) Settle(ctx context.Context, reservationNo string, usages []model.ProviderUsage, idempotencyKey string) (model.Settlement, error) {
	if strings.TrimSpace(reservationNo) == "" || strings.TrimSpace(idempotencyKey) == "" {
		return model.Settlement{}, errors.New("reservation and settlement idempotency keys are required")
	}
	if len(usages) == 0 {
		return model.Settlement{}, errors.New("at least one provider usage item is required")
	}
	now := b.businessNow()
	for index := range usages {
		if usages[index].CallSequence == 0 {
			return model.Settlement{}, errors.New("provider call sequence must start at one")
		}
		if usages[index].OccurredAt.IsZero() {
			usages[index].OccurredAt = now
		}
		rate, err := b.repository.CurrentRate(ctx, usages[index].ProviderKey, usages[index].ModelKey, usages[index].OccurredAt)
		if err != nil {
			if b.mode == model.ModeShadow {
				usages[index].SupplierCostMicros = 0
				usages[index].Credits = 0
				continue
			}
			return model.Settlement{}, fmt.Errorf("load AI rate card for %s/%s: %w", usages[index].ProviderKey, usages[index].ModelKey, err)
		}
		cost, credits, err := rate.Price(usages[index])
		if err != nil {
			return model.Settlement{}, err
		}
		usages[index].SupplierCostMicros = cost
		usages[index].Credits = credits
	}
	result, err := b.repository.SettleReservation(ctx, reservationNo, usages, idempotencyKey, now)
	if err != nil {
		return model.Settlement{}, fmt.Errorf("settle AI usage: %w", err)
	}
	return result, nil
}

func (b *Billing) Cancel(ctx context.Context, reservationNo, reason, idempotencyKey string) (model.Cancellation, error) {
	if strings.TrimSpace(reservationNo) == "" || strings.TrimSpace(reason) == "" || strings.TrimSpace(idempotencyKey) == "" {
		return model.Cancellation{}, errors.New("reservation, reason and cancellation idempotency keys are required")
	}
	result, err := b.repository.CancelReservation(ctx, reservationNo, reason, idempotencyKey, b.businessNow())
	if err != nil {
		return model.Cancellation{}, fmt.Errorf("cancel AI usage: %w", err)
	}
	return result, nil
}

func (b *Billing) Balance(ctx context.Context, owner model.Owner) (model.Balance, error) {
	if err := owner.Validate(); err != nil {
		return model.Balance{}, err
	}
	now := b.businessNow()
	if err := b.repository.EnsureMonthlyGrant(ctx, owner, now); err != nil {
		return model.Balance{}, fmt.Errorf("provision monthly AI credits: %w", err)
	}
	return b.repository.Balance(ctx, owner, now)
}

func (b *Billing) Mode() model.EnforcementMode { return b.mode }

func defaultTTL(operation string) time.Duration {
	if strings.Contains(strings.ToLower(operation), "agent") {
		return 2 * time.Hour
	}
	return 10 * time.Minute
}
