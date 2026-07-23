package repository

import (
	"context"
	"time"

	"smart-recruit-billing-service/internal/domain/model"
)

var ErrInsufficientCredits = &insufficientCreditsError{}

type insufficientCreditsError struct{}

func (*insufficientCreditsError) Error() string { return "insufficient AI credits" }

type BillingRepository interface {
	EnsureMonthlyGrant(context.Context, model.Owner, time.Time) error
	Balance(context.Context, model.Owner, time.Time) (model.Balance, error)
	CurrentRate(context.Context, string, string, time.Time) (model.RateCard, error)
	CreateReservation(context.Context, model.Reservation) (model.Reservation, model.Balance, bool, error)
	SettleReservation(context.Context, string, []model.ProviderUsage, string, time.Time) (model.Settlement, error)
	CancelReservation(context.Context, string, string, string, time.Time) (model.Cancellation, error)
}
