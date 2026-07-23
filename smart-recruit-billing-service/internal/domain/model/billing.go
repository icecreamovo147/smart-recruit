package model

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

var (
	ErrInvalidProviderUsage = errors.New("invalid provider usage")
	ErrPriceOverflow        = errors.New("AI usage price overflow")
)

type OwnerType string

const (
	OwnerTenant OwnerType = "tenant"
	OwnerUser   OwnerType = "user"
)

type EnforcementMode string

const (
	ModeShadow  EnforcementMode = "shadow"
	ModeEnforce EnforcementMode = "enforce"
)

type Owner struct {
	Type OwnerType
	ID   uint64
}

func (o Owner) Validate() error {
	if o.Type != OwnerTenant && o.Type != OwnerUser {
		return errors.New("billing owner type must be tenant or user")
	}
	if o.ID == 0 {
		return errors.New("billing owner id is required")
	}
	return nil
}

type Reservation struct {
	ID              uint64
	No              string
	Owner           Owner
	UserID          uint64
	Capability      string
	Operation       string
	ProviderKey     string
	ModelKey        string
	IdempotencyKey  string
	ReservedCredits uint64
	SettledCredits  uint64
	Mode            EnforcementMode
	Status          string
	ExpiresAt       time.Time
	CreatedAt       time.Time
}

func (r Reservation) Validate() error {
	if err := r.Owner.Validate(); err != nil {
		return err
	}
	if r.UserID == 0 || strings.TrimSpace(r.Operation) == "" || strings.TrimSpace(r.IdempotencyKey) == "" {
		return errors.New("user, operation and idempotency key are required")
	}
	if r.Mode != ModeShadow && r.Mode != ModeEnforce {
		return errors.New("invalid billing enforcement mode")
	}
	if r.ExpiresAt.IsZero() {
		return errors.New("reservation expiry is required")
	}
	return nil
}

type ProviderUsage struct {
	CallSequence       uint32
	ProviderKey        string
	ModelKey           string
	InputTokens        uint64
	OutputTokens       uint64
	CachedInputTokens  uint64
	ProviderRequestID  string
	Estimated          bool
	OccurredAt         time.Time
	MetadataJSON       string
	SupplierCostMicros uint64
	Credits            uint64
}

type RateCard struct {
	InputMicrosPer1K       uint64
	OutputMicrosPer1K      uint64
	CachedInputMicrosPer1K uint64
	CreditMicros           uint64
}

func (r RateCard) Price(usage ProviderUsage) (uint64, uint64, error) {
	if r.CreditMicros == 0 {
		return 0, 0, errors.New("rate card credit conversion is zero")
	}
	if usage.CachedInputTokens > usage.InputTokens {
		return 0, 0, fmt.Errorf("%w: cached input tokens exceed input tokens", ErrInvalidProviderUsage)
	}
	uncachedInputTokens := usage.InputTokens - usage.CachedInputTokens
	uncachedCost, err := priceTokenComponent(uncachedInputTokens, r.InputMicrosPer1K)
	if err != nil {
		return 0, 0, err
	}
	cachedCost, err := priceTokenComponent(usage.CachedInputTokens, r.CachedInputMicrosPer1K)
	if err != nil {
		return 0, 0, err
	}
	outputCost, err := priceTokenComponent(usage.OutputTokens, r.OutputMicrosPer1K)
	if err != nil {
		return 0, 0, err
	}
	cost, err := checkedAdd(uncachedCost, cachedCost)
	if err != nil {
		return 0, 0, err
	}
	cost, err = checkedAdd(cost, outputCost)
	if err != nil {
		return 0, 0, err
	}
	return cost, ceilDiv(cost, r.CreditMicros), nil
}

func priceTokenComponent(tokens, microsPer1K uint64) (uint64, error) {
	if tokens == 0 || microsPer1K == 0 {
		return 0, nil
	}
	if tokens > math.MaxUint64/microsPer1K {
		return 0, ErrPriceOverflow
	}
	return ceilDiv(tokens*microsPer1K, 1000), nil
}

func checkedAdd(left, right uint64) (uint64, error) {
	if left > math.MaxUint64-right {
		return 0, ErrPriceOverflow
	}
	return left + right, nil
}

func ceilDiv(value, divisor uint64) uint64 {
	if value == 0 {
		return 0
	}
	return 1 + (value-1)/divisor
}

type Balance struct {
	AvailableCredits uint64
	ReservedCredits  uint64
	NextExpiryAt     *time.Time
}

type Settlement struct {
	ChargedCredits     uint64
	SupplierCostMicros uint64
	AvailableCredits   uint64
	AlreadySettled     bool
}

type Cancellation struct {
	ReleasedCredits  uint64
	AvailableCredits uint64
	AlreadyCancelled bool
}
