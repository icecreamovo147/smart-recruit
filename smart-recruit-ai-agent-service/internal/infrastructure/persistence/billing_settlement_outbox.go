package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm/clause"

	aiagentgrpc "smart-recruit-ai-agent-service/internal/interfaces/grpc"
	"smart-recruit-platform-go/businessclock"
	platformlogger "smart-recruit-platform-go/logger"
	platformobservability "smart-recruit-platform-go/observability"
	"smart-recruit-proto/recruitment/pb"
)

const (
	billingOutboxReserved         = "reserved"
	billingOutboxPendingSettle    = "pending_settle"
	billingOutboxProcessingSettle = "processing_settle"
	billingOutboxPendingCancel    = "pending_cancel"
	billingOutboxProcessingCancel = "processing_cancel"
	billingOutboxSettled          = "settled"
	billingOutboxCancelled        = "cancelled"
	billingOutboxDead             = "dead"
	billingOutboxMaxRetries       = 12
	billingOutboxInvocationLease  = 5 * time.Minute
)

type billingSettlementOutboxRow struct {
	ID             uint64     `gorm:"column:id;primaryKey"`
	ReservationNo  string     `gorm:"column:reservation_no;uniqueIndex:uk_ai_billing_outbox_reservation"`
	OwnerType      string     `gorm:"column:owner_type"`
	OwnerID        uint64     `gorm:"column:owner_id"`
	UserID         uint64     `gorm:"column:user_id"`
	Capability     string     `gorm:"column:capability"`
	Operation      string     `gorm:"column:operation"`
	ProviderKey    string     `gorm:"column:provider_key"`
	ModelKey       string     `gorm:"column:model_key"`
	Status         string     `gorm:"column:status"`
	RequestPayload *string    `gorm:"column:request_payload"`
	IdempotencyKey *string    `gorm:"column:idempotency_key"`
	RetryCount     uint32     `gorm:"column:retry_count"`
	NextAttemptAt  *time.Time `gorm:"column:next_attempt_at"`
	LockedAt       *time.Time `gorm:"column:locked_at"`
	LastError      *string    `gorm:"column:last_error"`
	CompletedAt    *time.Time `gorm:"column:completed_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

func (billingSettlementOutboxRow) TableName() string { return "ai_billing_settlement_outbox" }

func (s *NativeStore) CreateBillingOutboxReservation(ctx context.Context, record aiagentgrpc.BillingOutboxReservation) error {
	if s == nil || s.db == nil {
		return errors.New("AI billing settlement outbox database is unavailable")
	}
	row := billingSettlementOutboxRow{
		ReservationNo: strings.TrimSpace(record.ReservationNo), OwnerType: record.OwnerType,
		OwnerID: uint64(record.OwnerID), UserID: uint64(record.UserID), Capability: record.Capability,
		Operation: record.Operation, ProviderKey: record.ProviderKey, ModelKey: record.ModelKey,
		Status: billingOutboxReserved, CreatedAt: businessclock.Now(), UpdatedAt: businessclock.Now(),
	}
	now := row.CreatedAt
	row.LockedAt = &now
	if row.ReservationNo == "" || row.OwnerID == 0 || row.UserID == 0 {
		return errors.New("AI billing reservation outbox identity is invalid")
	}
	result := s.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "reservation_no"}}, DoNothing: true}).Create(&row)
	if result.Error != nil || result.RowsAffected > 0 {
		return result.Error
	}
	var existing billingSettlementOutboxRow
	if err := s.db.WithContext(ctx).Select("reservation_no", "status", "locked_at").Where("reservation_no = ?", row.ReservationNo).First(&existing).Error; err != nil {
		return err
	}
	if existing.Status != billingOutboxReserved {
		return fmt.Errorf("AI billing reservation already has delivery status %s", existing.Status)
	}
	if existing.LockedAt != nil && existing.LockedAt.After(now.Add(-billingOutboxInvocationLease)) {
		return errors.New("AI billing reservation invocation is already in progress")
	}
	claim := s.db.WithContext(ctx).Model(&billingSettlementOutboxRow{}).
		Where("reservation_no = ? AND status = ? AND (locked_at IS NULL OR locked_at <= ?)", row.ReservationNo, billingOutboxReserved, now.Add(-billingOutboxInvocationLease)).
		Updates(map[string]any{"locked_at": now, "updated_at": now})
	if claim.Error != nil {
		return claim.Error
	}
	if claim.RowsAffected == 0 {
		return errors.New("AI billing reservation invocation is already in progress")
	}
	return nil
}

func (s *NativeStore) QueueBillingOutboxSettlement(ctx context.Context, request *pb.SettleAIUsageRequest) error {
	if request == nil || strings.TrimSpace(request.GetReservationNo()) == "" {
		return errors.New("AI billing settlement request is invalid")
	}
	payload, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(request)
	if err != nil {
		return err
	}
	return s.queueBillingOutbox(ctx, request.GetReservationNo(), request.GetIdempotencyKey(), string(payload), billingOutboxPendingSettle)
}

func (s *NativeStore) QueueBillingOutboxCancellation(ctx context.Context, request *pb.CancelAIUsageRequest) error {
	if request == nil || strings.TrimSpace(request.GetReservationNo()) == "" {
		return errors.New("AI billing cancellation request is invalid")
	}
	payload, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(request)
	if err != nil {
		return err
	}
	return s.queueBillingOutbox(ctx, request.GetReservationNo(), request.GetIdempotencyKey(), string(payload), billingOutboxPendingCancel)
}

func (s *NativeStore) queueBillingOutbox(ctx context.Context, reservationNo, idempotencyKey, payload, status string) error {
	if s == nil || s.db == nil {
		return errors.New("AI billing settlement outbox database is unavailable")
	}
	allowedStatuses := []string{billingOutboxReserved, billingOutboxPendingSettle, billingOutboxProcessingSettle}
	completedStatus := billingOutboxSettled
	if status == billingOutboxPendingCancel {
		allowedStatuses = []string{billingOutboxReserved, billingOutboxPendingCancel, billingOutboxProcessingCancel}
		completedStatus = billingOutboxCancelled
	}
	now := businessclock.Now()
	result := s.db.WithContext(ctx).Model(&billingSettlementOutboxRow{}).
		Where("reservation_no = ? AND status IN ?", reservationNo, allowedStatuses).
		Updates(map[string]any{
			"status": status, "request_payload": payload, "idempotency_key": idempotencyKey,
			"next_attempt_at": now, "locked_at": nil, "last_error": nil, "updated_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}
	var row billingSettlementOutboxRow
	if err := s.db.WithContext(ctx).Where("reservation_no = ?", reservationNo).First(&row).Error; err != nil {
		return err
	}
	if row.Status == completedStatus {
		return nil
	}
	return errors.New("AI billing settlement outbox cannot be queued from current state")
}

func (s *NativeStore) MarkBillingOutboxSettled(ctx context.Context, reservationNo string) error {
	return s.completeBillingOutbox(ctx, reservationNo, billingOutboxSettled)
}

func (s *NativeStore) MarkBillingOutboxCancelled(ctx context.Context, reservationNo string) error {
	return s.completeBillingOutbox(ctx, reservationNo, billingOutboxCancelled)
}

func (s *NativeStore) completeBillingOutbox(ctx context.Context, reservationNo, status string) error {
	if s == nil || s.db == nil {
		return errors.New("AI billing settlement outbox database is unavailable")
	}
	now := businessclock.Now()
	return s.db.WithContext(ctx).Model(&billingSettlementOutboxRow{}).Where("reservation_no = ?", reservationNo).Updates(map[string]any{
		"status": status, "completed_at": now, "next_attempt_at": nil, "locked_at": nil,
		"last_error": nil, "updated_at": now,
	}).Error
}

// RunBillingSettlementOutbox continuously retries requests that were durably
// recorded before their synchronous Billing RPC failed or the process exited.
func (s *NativeStore) RunBillingSettlementOutbox(ctx context.Context, billing pb.BillingServiceClient) {
	if s == nil || s.db == nil || billing == nil {
		return
	}
	process := func() {
		if err := s.processBillingSettlementOutbox(ctx, billing, 100); err != nil && ctx.Err() == nil {
			platformlogger.L().Warn("process AI billing settlement outbox failed", zap.Error(err))
		}
	}
	process()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			process()
		}
	}
}

func (s *NativeStore) processBillingSettlementOutbox(ctx context.Context, billing pb.BillingServiceClient, limit int) error {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	now := businessclock.Now()
	stale := now.Add(-2 * time.Minute)
	if err := s.db.WithContext(ctx).Model(&billingSettlementOutboxRow{}).
		Where("status = ? AND locked_at < ?", billingOutboxProcessingSettle, stale).
		Updates(map[string]any{"status": billingOutboxPendingSettle, "locked_at": nil, "next_attempt_at": now, "updated_at": now}).Error; err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Model(&billingSettlementOutboxRow{}).
		Where("status = ? AND locked_at < ?", billingOutboxProcessingCancel, stale).
		Updates(map[string]any{"status": billingOutboxPendingCancel, "locked_at": nil, "next_attempt_at": now, "updated_at": now}).Error; err != nil {
		return err
	}
	var rows []billingSettlementOutboxRow
	if err := s.db.WithContext(ctx).
		Where("status IN ? AND next_attempt_at <= ?", []string{billingOutboxPendingSettle, billingOutboxPendingCancel}, now).
		Order("next_attempt_at, id").Limit(limit).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := s.deliverBillingOutboxRow(ctx, billing, row, now); err != nil && ctx.Err() != nil {
			return ctx.Err()
		}
	}
	s.updateBillingOutboxMetrics(ctx)
	return nil
}

func (s *NativeStore) deliverBillingOutboxRow(ctx context.Context, billing pb.BillingServiceClient, row billingSettlementOutboxRow, now time.Time) error {
	processing := billingOutboxProcessingSettle
	if row.Status == billingOutboxPendingCancel {
		processing = billingOutboxProcessingCancel
	}
	claim := s.db.WithContext(ctx).Model(&billingSettlementOutboxRow{}).
		Where("id = ? AND status = ?", row.ID, row.Status).
		Updates(map[string]any{"status": processing, "locked_at": now, "updated_at": now})
	if claim.Error != nil || claim.RowsAffected == 0 {
		return claim.Error
	}
	if row.RequestPayload == nil {
		return s.failBillingOutboxRow(ctx, row, errors.New("billing outbox payload is missing"), now)
	}
	callCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 8*time.Second)
	defer cancel()
	var callErr error
	if processing == billingOutboxProcessingSettle {
		request := &pb.SettleAIUsageRequest{}
		if err := protojson.Unmarshal([]byte(*row.RequestPayload), request); err != nil {
			callErr = err
		} else {
			_, callErr = billing.SettleAIUsage(callCtx, request)
		}
		if callErr == nil {
			callErr = s.MarkBillingOutboxSettled(callCtx, row.ReservationNo)
		}
	} else {
		request := &pb.CancelAIUsageRequest{}
		if err := protojson.Unmarshal([]byte(*row.RequestPayload), request); err != nil {
			callErr = err
		} else {
			_, callErr = billing.CancelAIUsage(callCtx, request)
		}
		if callErr == nil {
			callErr = s.MarkBillingOutboxCancelled(callCtx, row.ReservationNo)
		}
	}
	if callErr != nil {
		return s.failBillingOutboxRow(context.WithoutCancel(ctx), row, callErr, now)
	}
	platformobservability.DefaultMetrics.RecordBillingEvent("settlement_outbox_retry", "delivered")
	return nil
}

func (s *NativeStore) failBillingOutboxRow(ctx context.Context, row billingSettlementOutboxRow, cause error, now time.Time) error {
	retries := row.RetryCount + 1
	status := billingOutboxPendingSettle
	if row.Status == billingOutboxPendingCancel {
		status = billingOutboxPendingCancel
	}
	if retries >= billingOutboxMaxRetries {
		status = billingOutboxDead
	}
	delay := time.Second * time.Duration(1<<min(int(retries), 8))
	message := cause.Error()
	if len(message) > 500 {
		message = message[:500]
	}
	err := s.db.WithContext(ctx).Model(&billingSettlementOutboxRow{}).Where("id = ?", row.ID).Updates(map[string]any{
		"status": status, "retry_count": retries, "next_attempt_at": now.Add(delay),
		"locked_at": nil, "last_error": message, "updated_at": now,
	}).Error
	platformobservability.DefaultMetrics.RecordBillingEvent("settlement_outbox_retry", status)
	if status == billingOutboxDead {
		platformlogger.L().Error("AI billing settlement outbox entered dead state",
			zap.String("reservation_no", row.ReservationNo), zap.Uint32("retry_count", retries), zap.Error(cause))
	}
	return err
}

func (s *NativeStore) updateBillingOutboxMetrics(ctx context.Context) {
	type countRow struct {
		Status string
		Count  int64
	}
	var counts []countRow
	if err := s.db.WithContext(ctx).Model(&billingSettlementOutboxRow{}).Select("status, COUNT(*) count").Group("status").Scan(&counts).Error; err != nil {
		return
	}
	for _, status := range []string{
		billingOutboxReserved, billingOutboxPendingSettle, billingOutboxProcessingSettle,
		billingOutboxPendingCancel, billingOutboxProcessingCancel, billingOutboxSettled,
		billingOutboxCancelled, billingOutboxDead,
	} {
		platformobservability.DefaultMetrics.SetBillingGauge("ai_settlement_outbox", status, 0)
	}
	for _, item := range counts {
		platformobservability.DefaultMetrics.SetBillingGauge("ai_settlement_outbox", item.Status, float64(item.Count))
	}
}
