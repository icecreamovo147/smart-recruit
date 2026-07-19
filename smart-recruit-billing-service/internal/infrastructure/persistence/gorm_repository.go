package persistence

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"smart-recruit-billing-service/internal/domain/model"
	"smart-recruit-billing-service/internal/domain/repository"
)

type GormRepository struct{ db *gorm.DB }

func NewGormRepository(db *gorm.DB) (*GormRepository, error) {
	if db == nil {
		return nil, errors.New("billing database is required")
	}
	return &GormRepository{db: db}, nil
}

type creditGrantRow struct {
	ID               uint64     `gorm:"column:id"`
	RemainingCredits uint64     `gorm:"column:remaining_credits"`
	ExpiresAt        *time.Time `gorm:"column:expires_at"`
}

func (creditGrantRow) TableName() string { return "ai_credit_grants" }

type reservationRow struct {
	ID              uint64     `gorm:"column:id;primaryKey"`
	ReservationNo   string     `gorm:"column:reservation_no"`
	OwnerType       string     `gorm:"column:owner_type"`
	OwnerID         uint64     `gorm:"column:owner_id"`
	UserID          uint64     `gorm:"column:user_id"`
	Capability      string     `gorm:"column:capability"`
	Operation       string     `gorm:"column:operation"`
	ProviderKey     string     `gorm:"column:provider_key"`
	ModelKey        string     `gorm:"column:model_key"`
	IdempotencyKey  string     `gorm:"column:idempotency_key"`
	ReservedCredits uint64     `gorm:"column:reserved_credits"`
	SettledCredits  uint64     `gorm:"column:settled_credits"`
	Status          string     `gorm:"column:status"`
	EnforcementMode string     `gorm:"column:enforcement_mode"`
	ExpiresAt       time.Time  `gorm:"column:expires_at"`
	SettledAt       *time.Time `gorm:"column:settled_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (reservationRow) TableName() string { return "ai_credit_reservations" }

type usageEventRow struct {
	ID                   uint64    `gorm:"column:id;primaryKey"`
	EventID              string    `gorm:"column:event_id"`
	OwnerType            string    `gorm:"column:owner_type"`
	OwnerID              uint64    `gorm:"column:owner_id"`
	TenantID             *uint64   `gorm:"column:tenant_id"`
	UserID               uint64    `gorm:"column:user_id"`
	ReservationID        uint64    `gorm:"column:reservation_id"`
	Operation            string    `gorm:"column:operation"`
	ProviderKey          string    `gorm:"column:provider_key"`
	ModelKey             string    `gorm:"column:model_key"`
	ProviderCallSequence uint32    `gorm:"column:provider_call_seq"`
	InputTokens          uint64    `gorm:"column:input_tokens"`
	OutputTokens         uint64    `gorm:"column:output_tokens"`
	CachedInputTokens    uint64    `gorm:"column:cached_input_tokens"`
	SupplierCostMicros   uint64    `gorm:"column:supplier_cost_micros"`
	CreditsCharged       uint64    `gorm:"column:credits_charged"`
	UsageSource          string    `gorm:"column:usage_source"`
	ProviderRequestID    *string   `gorm:"column:provider_request_id"`
	OccurredAt           time.Time `gorm:"column:occurred_at"`
	Metadata             *string   `gorm:"column:metadata"`
	CreatedAt            time.Time `gorm:"column:created_at"`
}

func (usageEventRow) TableName() string { return "ai_usage_events" }

type ledgerRow struct {
	EntryID        string    `gorm:"column:entry_id"`
	OwnerType      string    `gorm:"column:owner_type"`
	OwnerID        uint64    `gorm:"column:owner_id"`
	GrantID        *uint64   `gorm:"column:grant_id"`
	ReservationID  *uint64   `gorm:"column:reservation_id"`
	UsageEventID   *uint64   `gorm:"column:usage_event_id"`
	EntryType      string    `gorm:"column:entry_type"`
	CreditsDelta   int64     `gorm:"column:credits_delta"`
	BalanceAfter   int64     `gorm:"column:balance_after"`
	IdempotencyKey string    `gorm:"column:idempotency_key"`
	Description    string    `gorm:"column:description"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (ledgerRow) TableName() string { return "ai_credit_ledger" }

type rateCardRow struct {
	InputMicrosPer1K       uint64 `gorm:"column:input_micros_per_1k_tokens"`
	OutputMicrosPer1K      uint64 `gorm:"column:output_micros_per_1k_tokens"`
	CachedInputMicrosPer1K uint64 `gorm:"column:cached_input_micros_per_1k_tokens"`
	CreditMicros           uint64 `gorm:"column:credit_micros"`
}

func (rateCardRow) TableName() string { return "ai_rate_cards" }

func (r *GormRepository) Balance(ctx context.Context, owner model.Owner, now time.Time) (model.Balance, error) {
	return balanceWithDB(ctx, r.db, owner, now, false)
}

func (r *GormRepository) EnsureMonthlyGrant(ctx context.Context, owner model.Owner, now time.Time) error {
	start, end := shanghaiMonth(now)
	type grantSource struct {
		SourceID   uint64 `gorm:"column:source_id"`
		Credits    uint64 `gorm:"column:credits"`
		SourceType string `gorm:"column:source_type"`
		GrantType  string `gorm:"column:grant_type"`
	}
	var source grantSource
	var err error
	if owner.Type == model.OwnerTenant {
		err = r.db.WithContext(ctx).Raw(`
			SELECT subscription.id source_id, price.included_credits credits,
			       'subscription' source_type, 'subscription_monthly' grant_type
			FROM billing_subscriptions subscription
			JOIN billing_price_versions price ON price.id = subscription.price_version_id
			WHERE subscription.owner_type = 'tenant' AND subscription.owner_id = ? AND subscription.status = 'active'
			  AND subscription.current_period_start < ? AND subscription.current_period_end > ?
			ORDER BY subscription.current_period_end DESC, subscription.id DESC LIMIT 1`, owner.ID, end, start).Scan(&source).Error
		if err == nil && source.SourceID == 0 {
			err = r.db.WithContext(ctx).Raw(`
			SELECT subscription.id source_id, CAST(JSON_UNQUOTE(entitlement.value_json) AS UNSIGNED) credits,
			       'system' source_type, 'subscription_monthly' grant_type
			FROM tenant_subscriptions subscription
			JOIN platform_plan_entitlements entitlement ON entitlement.plan_version_id = subscription.plan_version_id
			WHERE subscription.tenant_id = ? AND subscription.status = 'active'
			  AND subscription.starts_at < ? AND (subscription.ends_at IS NULL OR subscription.ends_at > ?)
			  AND entitlement.entitlement_key = 'ai.credits.monthly'
			ORDER BY subscription.starts_at DESC, subscription.id DESC LIMIT 1`, owner.ID, end, start).Scan(&source).Error
		}
	} else {
		err = r.db.WithContext(ctx).Raw(`
			SELECT subscription.id source_id, price.included_credits credits,
			       'subscription' source_type, 'subscription_monthly' grant_type
			FROM billing_subscriptions subscription
			JOIN billing_price_versions price ON price.id = subscription.price_version_id
			WHERE subscription.owner_type = 'user' AND subscription.owner_id = ? AND subscription.status = 'active'
			  AND subscription.current_period_start < ? AND subscription.current_period_end > ?
			ORDER BY subscription.current_period_end DESC, subscription.id DESC LIMIT 1`, owner.ID, end, start).Scan(&source).Error
		if err == nil && source.SourceID == 0 {
			err = r.db.WithContext(ctx).Raw(`
				SELECT price.id source_id, price.included_credits credits,
				       'system' source_type, 'free_monthly' grant_type
				FROM billing_products product JOIN billing_price_versions price ON price.product_id = product.id
				WHERE product.product_key = 'candidate_free' AND product.status = 'active' AND price.status = 'published'
				  AND price.effective_at <= ? ORDER BY price.version DESC LIMIT 1`, now).Scan(&source).Error
		}
	}
	if err != nil {
		return err
	}
	if source.SourceID == 0 || source.Credits == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Table("ai_credit_grants").Where("owner_type = ? AND owner_id = ? AND source_type = ? AND source_id = ? AND valid_from = ?", owner.Type, owner.ID, source.SourceType, source.SourceID, start).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		grant := map[string]any{
			"owner_type": owner.Type, "owner_id": owner.ID, "grant_type": source.GrantType,
			"source_type": source.SourceType, "source_id": source.SourceID, "total_credits": source.Credits,
			"remaining_credits": source.Credits, "valid_from": start, "expires_at": end,
			"status": "active", "created_at": now, "updated_at": now,
		}
		if err := tx.Table("ai_credit_grants").Create(grant).Error; err != nil {
			return err
		}
		var grantID uint64
		if err := tx.Raw("SELECT LAST_INSERT_ID()").Scan(&grantID).Error; err != nil {
			return err
		}
		gross, err := grossGrantBalance(ctx, tx, owner, now, false)
		if err != nil {
			return err
		}
		return tx.Create(&ledgerRow{
			EntryID: uuid.NewString(), OwnerType: string(owner.Type), OwnerID: owner.ID, GrantID: &grantID,
			EntryType: "grant", CreditsDelta: int64(source.Credits), BalanceAfter: int64(gross),
			IdempotencyKey: fmt.Sprintf("monthly:%s:%d:%s", source.SourceType, source.SourceID, start.Format(time.RFC3339)),
			Description:    "monthly AI credit grant", CreatedAt: now,
		}).Error
	})
}

func balanceWithDB(ctx context.Context, db *gorm.DB, owner model.Owner, now time.Time, lock bool) (model.Balance, error) {
	query := db.WithContext(ctx).Model(&creditGrantRow{}).
		Where("owner_type = ? AND owner_id = ? AND status = 'active' AND valid_from <= ? AND (expires_at IS NULL OR expires_at > ?)", owner.Type, owner.ID, now, now)
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var grants []creditGrantRow
	if err := query.Order("expires_at IS NULL, expires_at, valid_from, id").Find(&grants).Error; err != nil {
		return model.Balance{}, err
	}
	var total uint64
	var next *time.Time
	for _, grant := range grants {
		total += grant.RemainingCredits
		if grant.ExpiresAt != nil && (next == nil || grant.ExpiresAt.Before(*next)) {
			value := *grant.ExpiresAt
			next = &value
		}
	}
	var reserved uint64
	if err := db.WithContext(ctx).Model(&reservationRow{}).
		Where("owner_type = ? AND owner_id = ? AND status = 'active' AND enforcement_mode = 'enforce' AND expires_at > ?", owner.Type, owner.ID, now).
		Select("COALESCE(SUM(reserved_credits), 0)").Scan(&reserved).Error; err != nil {
		return model.Balance{}, err
	}
	available := uint64(0)
	if total > reserved {
		available = total - reserved
	}
	return model.Balance{AvailableCredits: available, ReservedCredits: reserved, NextExpiryAt: next}, nil
}

func (r *GormRepository) CurrentRate(ctx context.Context, provider, modelKey string, at time.Time) (model.RateCard, error) {
	var row rateCardRow
	err := r.db.WithContext(ctx).Where("provider_key = ? AND model_key = ? AND status = 'published' AND effective_at <= ? AND (retired_at IS NULL OR retired_at > ?)", provider, modelKey, at, at).
		Order("effective_at DESC, version DESC").First(&row).Error
	if err != nil {
		return model.RateCard{}, err
	}
	return model.RateCard{InputMicrosPer1K: row.InputMicrosPer1K, OutputMicrosPer1K: row.OutputMicrosPer1K, CachedInputMicrosPer1K: row.CachedInputMicrosPer1K, CreditMicros: row.CreditMicros}, nil
}

func (r *GormRepository) CreateReservation(ctx context.Context, reservation model.Reservation) (model.Reservation, model.Balance, bool, error) {
	var result model.Reservation
	var balance model.Balance
	var existing bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row reservationRow
		err := tx.Where("owner_type = ? AND owner_id = ? AND idempotency_key = ?", reservation.Owner.Type, reservation.Owner.ID, reservation.IdempotencyKey).First(&row).Error
		if err == nil {
			existing = true
			result = reservationFromRow(row)
			var balanceErr error
			balance, balanceErr = balanceWithDB(ctx, tx, reservation.Owner, reservation.CreatedAt, true)
			return balanceErr
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var balanceErr error
		balance, balanceErr = balanceWithDB(ctx, tx, reservation.Owner, reservation.CreatedAt, true)
		if balanceErr != nil {
			return balanceErr
		}
		if reservation.Mode == model.ModeEnforce && balance.AvailableCredits < reservation.ReservedCredits {
			return repository.ErrInsufficientCredits
		}
		row = reservationRow{
			ReservationNo: reservation.No, OwnerType: string(reservation.Owner.Type), OwnerID: reservation.Owner.ID,
			UserID: reservation.UserID, Capability: reservation.Capability, Operation: reservation.Operation,
			ProviderKey: reservation.ProviderKey, ModelKey: reservation.ModelKey, IdempotencyKey: reservation.IdempotencyKey,
			ReservedCredits: reservation.ReservedCredits, Status: "active", EnforcementMode: string(reservation.Mode),
			ExpiresAt: reservation.ExpiresAt, CreatedAt: reservation.CreatedAt, UpdatedAt: reservation.CreatedAt,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if reservation.Mode == model.ModeEnforce && reservation.ReservedCredits > 0 {
			balance.AvailableCredits -= reservation.ReservedCredits
			balance.ReservedCredits += reservation.ReservedCredits
			if err := tx.Create(&ledgerRow{
				EntryID: uuid.NewString(), OwnerType: row.OwnerType, OwnerID: row.OwnerID, ReservationID: &row.ID,
				EntryType: "reserve", CreditsDelta: -int64(row.ReservedCredits), BalanceAfter: int64(balance.AvailableCredits),
				IdempotencyKey: "reserve:" + reservation.IdempotencyKey, Description: "AI usage credit reservation", CreatedAt: reservation.CreatedAt,
			}).Error; err != nil {
				return err
			}
		}
		result = reservationFromRow(row)
		return nil
	})
	return result, balance, existing, err
}

func (r *GormRepository) SettleReservation(ctx context.Context, reservationNo string, usages []model.ProviderUsage, idempotencyKey string, now time.Time) (model.Settlement, error) {
	var result model.Settlement
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reservation reservationRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("reservation_no = ?", reservationNo).First(&reservation).Error; err != nil {
			return err
		}
		owner := model.Owner{Type: model.OwnerType(reservation.OwnerType), ID: reservation.OwnerID}
		if reservation.Status == "settled" {
			result.AlreadySettled = true
			result.ChargedCredits = reservation.SettledCredits
			if err := tx.Model(&usageEventRow{}).Where("reservation_id = ?", reservation.ID).Select("COALESCE(SUM(supplier_cost_micros), 0)").Scan(&result.SupplierCostMicros).Error; err != nil {
				return err
			}
			balance, err := balanceWithDB(ctx, tx, owner, now, false)
			result.AvailableCredits = balance.AvailableCredits
			return err
		}
		if reservation.Status != "active" {
			return fmt.Errorf("reservation status %s cannot be settled", reservation.Status)
		}
		sort.Slice(usages, func(i, j int) bool { return usages[i].CallSequence < usages[j].CallSequence })
		var predictedCredits uint64
		for _, usage := range usages {
			predictedCredits += usage.Credits
			result.SupplierCostMicros += usage.SupplierCostMicros
		}
		chargedCredits := predictedCredits
		if reservation.EnforcementMode == string(model.ModeEnforce) {
			// Provider calls may not create postpaid debt. The caller reserves the
			// maximum permitted run cost up front, so settlement is capped by that
			// reservation even when actual usage is unexpectedly larger.
			if chargedCredits > reservation.ReservedCredits {
				chargedCredits = reservation.ReservedCredits
			}
		}
		remainingCharge := chargedCredits
		for _, usage := range usages {
			usageCharge := usage.Credits
			if usageCharge > remainingCharge {
				usageCharge = remainingCharge
			}
			row := usageEventRow{
				EventID: uuid.NewString(), OwnerType: reservation.OwnerType, OwnerID: reservation.OwnerID,
				UserID: reservation.UserID, ReservationID: reservation.ID, Operation: reservation.Operation,
				ProviderKey: usage.ProviderKey, ModelKey: usage.ModelKey, ProviderCallSequence: usage.CallSequence,
				InputTokens: usage.InputTokens, OutputTokens: usage.OutputTokens, CachedInputTokens: usage.CachedInputTokens,
				SupplierCostMicros: usage.SupplierCostMicros, CreditsCharged: usageCharge, UsageSource: "provider",
				OccurredAt: usage.OccurredAt.UTC(), CreatedAt: now,
			}
			if owner.Type == model.OwnerTenant {
				tenantID := owner.ID
				row.TenantID = &tenantID
			}
			if usage.Estimated {
				row.UsageSource = "estimated"
			}
			if usage.ProviderRequestID != "" {
				value := usage.ProviderRequestID
				row.ProviderRequestID = &value
			}
			if usage.MetadataJSON != "" {
				value := usage.MetadataJSON
				row.Metadata = &value
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			remainingCharge -= usageCharge
		}
		if reservation.EnforcementMode == string(model.ModeEnforce) {
			if err := consumeGrants(ctx, tx, reservation, chargedCredits, idempotencyKey, now); err != nil {
				return err
			}
		}
		if err := tx.Model(&reservation).Updates(map[string]any{"status": "settled", "settled_credits": chargedCredits, "settled_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		balance, err := balanceWithDB(ctx, tx, owner, now, false)
		if err != nil {
			return err
		}
		result.ChargedCredits = chargedCredits
		result.AvailableCredits = balance.AvailableCredits
		return nil
	})
	return result, err
}

func (r *GormRepository) CancelReservation(ctx context.Context, reservationNo, reason, idempotencyKey string, now time.Time) (model.Cancellation, error) {
	var result model.Cancellation
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reservation reservationRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("reservation_no = ?", reservationNo).First(&reservation).Error; err != nil {
			return err
		}
		owner := model.Owner{Type: model.OwnerType(reservation.OwnerType), ID: reservation.OwnerID}
		if reservation.Status == "cancelled" || reservation.Status == "expired" {
			result.AlreadyCancelled = true
			balance, err := balanceWithDB(ctx, tx, owner, now, false)
			result.AvailableCredits = balance.AvailableCredits
			return err
		}
		if reservation.Status != "active" {
			return fmt.Errorf("reservation status %s cannot be cancelled", reservation.Status)
		}
		if err := tx.Model(&reservation).Updates(map[string]any{"status": "cancelled", "settled_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		if reservation.EnforcementMode == string(model.ModeEnforce) && reservation.ReservedCredits > 0 {
			balance, err := balanceWithDB(ctx, tx, owner, now, false)
			if err != nil {
				return err
			}
			if err := tx.Create(&ledgerRow{
				EntryID: uuid.NewString(), OwnerType: reservation.OwnerType, OwnerID: reservation.OwnerID, ReservationID: &reservation.ID,
				EntryType: "release", CreditsDelta: int64(reservation.ReservedCredits), BalanceAfter: int64(balance.AvailableCredits),
				IdempotencyKey: "cancel:" + idempotencyKey, Description: reason, CreatedAt: now,
			}).Error; err != nil {
				return err
			}
			result.ReleasedCredits = reservation.ReservedCredits
			result.AvailableCredits = balance.AvailableCredits
		} else {
			balance, err := balanceWithDB(ctx, tx, owner, now, false)
			if err != nil {
				return err
			}
			result.AvailableCredits = balance.AvailableCredits
		}
		return nil
	})
	return result, err
}

func grossGrantBalance(ctx context.Context, tx *gorm.DB, owner model.Owner, now time.Time, lock bool) (uint64, error) {
	query := tx.WithContext(ctx).Model(&creditGrantRow{}).Where("owner_type = ? AND owner_id = ? AND status = 'active' AND valid_from <= ? AND (expires_at IS NULL OR expires_at > ?)", owner.Type, owner.ID, now, now)
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var rows []creditGrantRow
	if err := query.Order("expires_at IS NULL, expires_at, id").Find(&rows).Error; err != nil {
		return 0, err
	}
	var total uint64
	for _, row := range rows {
		total += row.RemainingCredits
	}
	return total, nil
}

func consumeGrants(ctx context.Context, tx *gorm.DB, reservation reservationRow, credits uint64, idempotencyKey string, now time.Time) error {
	owner := model.Owner{Type: model.OwnerType(reservation.OwnerType), ID: reservation.OwnerID}
	var grants []creditGrantRow
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("owner_type = ? AND owner_id = ? AND status = 'active' AND remaining_credits > 0 AND valid_from <= ? AND (expires_at IS NULL OR expires_at > ?)", owner.Type, owner.ID, now, now).
		Order("expires_at IS NULL, expires_at, valid_from, id").Find(&grants).Error; err != nil {
		return err
	}
	remaining := credits
	var gross uint64
	for _, grant := range grants {
		gross += grant.RemainingCredits
	}
	availableAfter := gross
	for _, grant := range grants {
		if remaining == 0 {
			break
		}
		consume := grant.RemainingCredits
		if consume > remaining {
			consume = remaining
		}
		newRemaining := grant.RemainingCredits - consume
		status := "active"
		if newRemaining == 0 {
			status = "exhausted"
		}
		if err := tx.Model(&creditGrantRow{}).Where("id = ?", grant.ID).Updates(map[string]any{"remaining_credits": newRemaining, "status": status, "updated_at": now}).Error; err != nil {
			return err
		}
		availableAfter -= consume
		grantID := grant.ID
		if err := tx.Create(&ledgerRow{
			EntryID: uuid.NewString(), OwnerType: reservation.OwnerType, OwnerID: reservation.OwnerID,
			GrantID: &grantID, ReservationID: &reservation.ID, EntryType: "consume", CreditsDelta: -int64(consume),
			BalanceAfter: int64(availableAfter), IdempotencyKey: fmt.Sprintf("settle:%s:grant:%d", idempotencyKey, grant.ID),
			Description: "settled AI provider usage", CreatedAt: now,
		}).Error; err != nil {
			return err
		}
		remaining -= consume
	}
	return nil
}

func reservationFromRow(row reservationRow) model.Reservation {
	return model.Reservation{
		ID: row.ID, No: row.ReservationNo, Owner: model.Owner{Type: model.OwnerType(row.OwnerType), ID: row.OwnerID},
		UserID: row.UserID, Capability: row.Capability, Operation: row.Operation, ProviderKey: row.ProviderKey,
		ModelKey: row.ModelKey, IdempotencyKey: row.IdempotencyKey, ReservedCredits: row.ReservedCredits,
		SettledCredits: row.SettledCredits, Mode: model.EnforcementMode(row.EnforcementMode), Status: row.Status,
		ExpiresAt: row.ExpiresAt, CreatedAt: row.CreatedAt,
	}
}

// RunMaintenance expires stale reservations and credit buckets using append-only
// compensating ledger entries. It is safe to call concurrently from all replicas.
func (r *GormRepository) RunMaintenance(ctx context.Context, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("billing_subscriptions").Where("status = 'active' AND current_period_end <= ?", now).Updates(map[string]any{"status": "expired", "updated_at": now}).Error; err != nil {
			return err
		}
		type scheduledSubscription struct {
			ID        uint64
			OwnerID   uint64
			OwnerType string
		}
		var scheduled []scheduledSubscription
		if err := tx.Table("billing_subscriptions").Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("status = 'pending' AND current_period_start <= ? AND current_period_end > ?", now, now).Order("current_period_start, id").Limit(500).Find(&scheduled).Error; err != nil {
			return err
		}
		for _, subscription := range scheduled {
			var activeCount int64
			if err := tx.Table("billing_subscriptions").Where("owner_type = ? AND owner_id = ? AND status = 'active' AND current_period_end > ?", subscription.OwnerType, subscription.OwnerID, now).Count(&activeCount).Error; err != nil {
				return err
			}
			if activeCount > 0 {
				continue
			}
			if err := tx.Table("billing_subscriptions").Where("id = ? AND status = 'pending'", subscription.ID).Updates(map[string]any{"status": "active", "updated_at": now}).Error; err != nil {
				return err
			}
			payload := fmt.Sprintf(`{"subscription_id":%d,"owner_type":%q,"owner_id":%d}`, subscription.ID, subscription.OwnerType, subscription.OwnerID)
			if err := tx.Table("event_outbox").Create(map[string]any{"event_id": uuid.NewString(), "event_type": "billing.subscription.activated", "aggregate_type": "billing_subscription", "aggregate_id": subscription.ID, "routing_key": "billing.subscription.activated", "producer": "billing-service", "idempotency_key": fmt.Sprintf("billing-subscription:%d:activated", subscription.ID), "payload": payload, "status": 0, "created_at": now, "updated_at": now}).Error; err != nil {
				return err
			}
		}
		var reservations []reservationRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("status = 'active' AND expires_at <= ?", now).Limit(500).Find(&reservations).Error; err != nil {
			return err
		}
		for _, reservation := range reservations {
			if err := tx.Model(&reservation).Updates(map[string]any{"status": "expired", "settled_at": now, "updated_at": now}).Error; err != nil {
				return err
			}
			if reservation.EnforcementMode == string(model.ModeEnforce) && reservation.ReservedCredits > 0 {
				owner := model.Owner{Type: model.OwnerType(reservation.OwnerType), ID: reservation.OwnerID}
				balance, err := balanceWithDB(ctx, tx, owner, now, false)
				if err != nil {
					return err
				}
				if err := tx.Create(&ledgerRow{EntryID: uuid.NewString(), OwnerType: reservation.OwnerType, OwnerID: reservation.OwnerID, ReservationID: &reservation.ID, EntryType: "release", CreditsDelta: int64(reservation.ReservedCredits), BalanceAfter: int64(balance.AvailableCredits), IdempotencyKey: "expire-reservation:" + reservation.ReservationNo, Description: "expired AI credit reservation", CreatedAt: now}).Error; err != nil {
					return err
				}
			}
		}
		type expiringGrant struct {
			ID, OwnerID, RemainingCredits uint64
			OwnerType                     string
		}
		var grants []expiringGrant
		if err := tx.Table("ai_credit_grants").Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("status = 'active' AND expires_at IS NOT NULL AND expires_at <= ?", now).Limit(500).Find(&grants).Error; err != nil {
			return err
		}
		for _, grant := range grants {
			if err := tx.Table("ai_credit_grants").Where("id = ? AND status = 'active'", grant.ID).Updates(map[string]any{"status": "expired", "remaining_credits": 0, "updated_at": now}).Error; err != nil {
				return err
			}
			if grant.RemainingCredits == 0 {
				continue
			}
			owner := model.Owner{Type: model.OwnerType(grant.OwnerType), ID: grant.OwnerID}
			gross, err := grossGrantBalance(ctx, tx, owner, now, false)
			if err != nil {
				return err
			}
			grantID := grant.ID
			if err := tx.Create(&ledgerRow{EntryID: uuid.NewString(), OwnerType: grant.OwnerType, OwnerID: grant.OwnerID, GrantID: &grantID, EntryType: "expire", CreditsDelta: -int64(grant.RemainingCredits), BalanceAfter: int64(gross), IdempotencyKey: fmt.Sprintf("expire-grant:%d", grant.ID), Description: "expired AI credit grant", CreatedAt: now}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
