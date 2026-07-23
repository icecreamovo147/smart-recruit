package persistence

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	appservice "smart-recruit-billing-service/internal/application/service"
	"smart-recruit-billing-service/internal/domain/model"
	"smart-recruit-billing-service/internal/infrastructure/payment"
	"smart-recruit-platform-go/businessclock"
)

// This test is opt-in because it verifies MySQL row locks and transactional
// ledger behavior that SQLite cannot represent. The acceptance workflow runs
// it against a disposable database.
func TestMySQLCreditReservationSettlementLifecycle(t *testing.T) {
	dsn := os.Getenv("BILLING_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("BILLING_TEST_MYSQL_DSN is not configured")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, ddl := range billingLifecycleTestDDL {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo, err := NewGormRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 21, 8, 0, 0, 0, time.UTC)
	if err := db.Exec(`INSERT INTO ai_credit_grants
		(owner_type, owner_id, remaining_credits, valid_from, expires_at, status, created_at, updated_at)
		VALUES ('tenant', 9, 100, ?, ?, 'active', ?, ?)`, now.Add(-time.Hour), now.Add(time.Hour), now, now).Error; err != nil {
		t.Fatal(err)
	}
	reservation := model.Reservation{
		No: "11111111-1111-1111-1111-111111111111", Owner: model.Owner{Type: model.OwnerTenant, ID: 9},
		UserID: 7, Capability: "ai.chat", Operation: "hr_chat", ProviderKey: "provider", ModelKey: "model",
		IdempotencyKey: "reserve-1", ReservedCredits: 20, Mode: model.ModeEnforce,
		Status: "active", ExpiresAt: now.Add(10 * time.Minute), CreatedAt: now,
	}
	created, balance, existing, err := repo.CreateReservation(context.Background(), reservation)
	if err != nil {
		t.Fatal(err)
	}
	if existing || created.ID == 0 || balance.AvailableCredits != 80 || balance.ReservedCredits != 20 {
		t.Fatalf("reservation=%+v balance=%+v existing=%v", created, balance, existing)
	}

	type settlementResult struct {
		value model.Settlement
		err   error
	}
	results := make(chan settlementResult, 2)
	settle := func() {
		value, settleErr := repo.SettleReservation(context.Background(), reservation.No, []model.ProviderUsage{{
			CallSequence: 1, ProviderKey: "provider", ModelKey: "model", InputTokens: 100,
			OutputTokens: 20, Credits: 3, SupplierCostMicros: 2500, OccurredAt: now,
		}}, "settle-1", now)
		results <- settlementResult{value: value, err: settleErr}
	}
	go settle()
	go settle()
	first, second := <-results, <-results
	if first.err != nil || second.err != nil {
		t.Fatalf("settlement errors: %v / %v", first.err, second.err)
	}
	if first.value.AlreadySettled == second.value.AlreadySettled {
		t.Fatalf("exactly one concurrent settlement must be idempotent: %+v / %+v", first.value, second.value)
	}

	var remaining uint64
	if err := db.Table("ai_credit_grants").Select("remaining_credits").Where("owner_type = 'tenant' AND owner_id = 9").Scan(&remaining).Error; err != nil {
		t.Fatal(err)
	}
	var usageCount int64
	if err := db.Table("ai_usage_events").Where("reservation_id = ?", created.ID).Count(&usageCount).Error; err != nil {
		t.Fatal(err)
	}
	var ledgerDelta int64
	if err := db.Table("ai_credit_ledger").Select("COALESCE(SUM(credits_delta), 0)").Where("reservation_id = ?", created.ID).Scan(&ledgerDelta).Error; err != nil {
		t.Fatal(err)
	}
	finalBalance, err := repo.Balance(context.Background(), reservation.Owner, now)
	if err != nil {
		t.Fatal(err)
	}
	if remaining != 97 || usageCount != 1 || ledgerDelta != -3 || finalBalance.AvailableCredits != 97 || finalBalance.ReservedCredits != 0 {
		t.Fatalf("remaining=%d usage=%d ledger_delta=%d balance=%+v", remaining, usageCount, ledgerDelta, finalBalance)
	}
	var allocation reservationAllocationRow
	if err := db.Where("reservation_id = ?", created.ID).First(&allocation).Error; err != nil {
		t.Fatal(err)
	}
	if allocation.ReservedCredits != 20 || allocation.ConsumedCredits != 3 || allocation.ReleasedCredits != 17 {
		t.Fatalf("allocation=%+v", allocation)
	}

	type reserveResult struct {
		reservation model.Reservation
		err         error
	}
	concurrent := make(chan reserveResult, 2)
	for index := 0; index < 2; index++ {
		index := index
		go func() {
			value := allocationReservationForMySQL(now, index, 60)
			createdReservation, _, _, createErr := repo.CreateReservation(context.Background(), value)
			concurrent <- reserveResult{reservation: createdReservation, err: createErr}
		}()
	}
	reserveOne, reserveTwo := <-concurrent, <-concurrent
	successes := 0
	var successful model.Reservation
	for _, item := range []reserveResult{reserveOne, reserveTwo} {
		if item.err == nil {
			successes++
			successful = item.reservation
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent reservations = %+v / %+v, want exactly one success", reserveOne, reserveTwo)
	}
	cancelled, err := repo.CancelReservation(context.Background(), successful.No, "concurrency test cleanup", "cancel-concurrent", now)
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.ReleasedCredits != 60 || cancelled.AvailableCredits != 97 {
		t.Fatalf("cancellation=%+v, want released=60 available=97", cancelled)
	}

	expiredReservation := allocationReservationForMySQL(now, 2, 10)
	expiredReservation.ExpiresAt = now.Add(time.Minute)
	expiredCreated, _, _, err := repo.CreateReservation(context.Background(), expiredReservation)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.RunMaintenance(context.Background(), now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	var expiredStatus string
	if err := db.Table("ai_credit_reservations").Where("id = ?", expiredCreated.ID).Select("status").Scan(&expiredStatus).Error; err != nil {
		t.Fatal(err)
	}
	var expiredAllocation reservationAllocationRow
	if err := db.Where("reservation_id = ?", expiredCreated.ID).First(&expiredAllocation).Error; err != nil {
		t.Fatal(err)
	}
	if expiredStatus != "expired" || expiredAllocation.ReleasedCredits != 10 {
		t.Fatalf("expired status=%s allocation=%+v", expiredStatus, expiredAllocation)
	}

	if err := db.Exec(`INSERT INTO ai_credit_grants
		(owner_type, owner_id, remaining_credits, valid_from, expires_at, status, created_at, updated_at)
		VALUES ('tenant', 10, 30, ?, ?, 'active', ?, ?)`, now.Add(-time.Hour), now.Add(time.Minute), now, now).Error; err != nil {
		t.Fatal(err)
	}
	expiringReservation := model.Reservation{
		No: "33333333-3333-3333-3333-333333333333", Owner: model.Owner{Type: model.OwnerTenant, ID: 10},
		UserID: 8, Capability: "ai.chat", Operation: "hr_chat", ProviderKey: "provider", ModelKey: "model",
		IdempotencyKey: "reserve-expiring", ReservedCredits: 20, Mode: model.ModeEnforce,
		Status: "active", ExpiresAt: now.Add(10 * time.Minute), CreatedAt: now,
	}
	expiringCreated, _, _, err := repo.CreateReservation(context.Background(), expiringReservation)
	if err != nil {
		t.Fatal(err)
	}
	var expiringAllocation reservationAllocationRow
	if err := db.Where("reservation_id = ?", expiringCreated.ID).First(&expiringAllocation).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Table("ai_credit_grants").Where("id = ?", expiringAllocation.GrantID).Update("expires_at", now.Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SettleReservation(context.Background(), expiringReservation.No, []model.ProviderUsage{{
		CallSequence: 1, ProviderKey: "provider", ModelKey: "model", Credits: 3, OccurredAt: now,
	}}, "settle-expiring", now); err != nil {
		t.Fatalf("settle reservation backed before grant expiry: %v", err)
	}
	var expiringRemaining uint64
	if err := db.Table("ai_credit_grants").Where("id = ?", expiringAllocation.GrantID).Select("remaining_credits").Scan(&expiringRemaining).Error; err != nil {
		t.Fatal(err)
	}
	if expiringRemaining != 27 {
		t.Fatalf("expiring grant remaining=%d, want 27", expiringRemaining)
	}

	corrupt := allocationReservationForMySQL(now, 3, 5)
	corruptCreated, _, _, err := repo.CreateReservation(context.Background(), corrupt)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Where("reservation_id = ?", corruptCreated.ID).Delete(&reservationAllocationRow{}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SettleReservation(context.Background(), corrupt.No, []model.ProviderUsage{{
		CallSequence: 1, ProviderKey: "provider", ModelKey: "model", Credits: 2, OccurredAt: now,
	}}, "settle-corrupt", now); err == nil {
		t.Fatal("settlement unexpectedly succeeded without allocation backing")
	}
	var corruptUsageCount int64
	if err := db.Table("ai_usage_events").Where("reservation_id = ?", corruptCreated.ID).Count(&corruptUsageCount).Error; err != nil {
		t.Fatal(err)
	}
	if corruptUsageCount != 0 {
		t.Fatalf("corrupt settlement persisted %d usage rows", corruptUsageCount)
	}

	refundNow := businessclock.Now()
	if err := db.Exec(`INSERT INTO billing_orders
		(order_no, owner_type, owner_id, order_type, status, amount_fen, paid_at, created_at, updated_at)
		VALUES ('ORDER-REFUND-RACE', 'tenant', 11, 'purchase', 'paid', 3900, ?, ?, ?)`,
		refundNow, refundNow, refundNow).Error; err != nil {
		t.Fatal(err)
	}
	var refundOrderID uint64
	if err := db.Raw("SELECT LAST_INSERT_ID()").Scan(&refundOrderID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO billing_payments
		(order_id, merchant_order_no, status, created_at, updated_at)
		VALUES (?, 'MERCHANT-REFUND-RACE', 'succeeded', ?, ?)`, refundOrderID, refundNow, refundNow).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO ai_credit_grants
		(owner_type, owner_id, grant_type, source_type, source_id, total_credits, remaining_credits,
		 valid_from, expires_at, status, created_at, updated_at)
		VALUES ('tenant', 11, 'credit_pack', 'order', ?, 30, 30, ?, ?, 'active', ?, ?)`,
		refundOrderID, refundNow.Add(-time.Hour), refundNow.Add(time.Hour), refundNow, refundNow).Error; err != nil {
		t.Fatal(err)
	}
	refundReservation := model.Reservation{
		No: "44444444-4444-4444-4444-444444444444", Owner: model.Owner{Type: model.OwnerTenant, ID: 11},
		UserID: 9, Capability: "ai.chat", Operation: "hr_chat", ProviderKey: "provider", ModelKey: "model",
		IdempotencyKey: "reserve-refund-race", ReservedCredits: 20, Mode: model.ModeEnforce,
		Status: "active", ExpiresAt: refundNow.Add(10 * time.Minute), CreatedAt: refundNow,
	}
	refundCreated, _, _, err := repo.CreateReservation(context.Background(), refundReservation)
	if err != nil {
		t.Fatal(err)
	}
	alipay := &refundRaceAlipay{}
	commerce, err := appservice.NewCommerce(db, alipay)
	if err != nil {
		t.Fatal(err)
	}
	blocker := db.Begin()
	defer blocker.Rollback()
	var lockedAllocation reservationAllocationRow
	if err := blocker.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("reservation_id = ?", refundCreated.ID).
		First(&lockedAllocation).Error; err != nil {
		t.Fatal(err)
	}
	startRace := make(chan struct{})
	type refundResult struct {
		no, status string
		err        error
	}
	refundDone := make(chan refundResult, 1)
	settlementDone := make(chan error, 1)
	go func() {
		<-startRace
		refundNo, refundStatus, _, requestErr := commerce.RequestRefund(
			context.Background(),
			model.Owner{Type: model.OwnerTenant, ID: 11},
			9,
			"ORDER-REFUND-RACE",
			"race test",
			"refund-race",
		)
		refundDone <- refundResult{no: refundNo, status: refundStatus, err: requestErr}
	}()
	go func() {
		<-startRace
		_, settleErr := repo.SettleReservation(context.Background(), refundReservation.No, []model.ProviderUsage{{
			CallSequence: 1, ProviderKey: "provider", ModelKey: "model", Credits: 3, OccurredAt: refundNow,
		}}, "settle-refund-race", refundNow)
		settlementDone <- settleErr
	}()
	close(startRace)
	refundOutcome := <-refundDone
	if refundOutcome.err != nil {
		t.Fatal(refundOutcome.err)
	}
	if refundOutcome.status != "waiting_usage" || alipay.refundCallCount() != 0 {
		t.Fatalf("refund status=%s calls=%d", refundOutcome.status, alipay.refundCallCount())
	}
	if err := blocker.Commit().Error; err != nil {
		t.Fatal(err)
	}
	if err := <-settlementDone; err != nil {
		t.Fatal(err)
	}
	var frozenStatus string
	if err := db.Table("ai_credit_grants").Where("owner_type = 'tenant' AND owner_id = 11").Select("status").Scan(&frozenStatus).Error; err != nil {
		t.Fatal(err)
	}
	if frozenStatus != "refund_frozen" {
		t.Fatalf("grant status=%s, want refund_frozen", frozenStatus)
	}
	if err := db.Table("billing_refunds").Where("refund_no = ?", refundOutcome.no).Update("next_reconcile_at", refundNow.Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	if err := commerce.ReconcilePendingRefunds(context.Background(), 10); err != nil {
		t.Fatal(err)
	}
	var reconciled struct{ Status, ReviewMode string }
	if err := db.Table("billing_refunds").Where("refund_no = ?", refundOutcome.no).Select("status, review_mode").Scan(&reconciled).Error; err != nil {
		t.Fatal(err)
	}
	if reconciled.Status != "reviewing" || reconciled.ReviewMode != "manual" || alipay.refundCallCount() != 0 {
		t.Fatalf("refund after settlement=%+v calls=%d", reconciled, alipay.refundCallCount())
	}
	if _, statusValue, err := commerce.ReviewRefund(context.Background(), refundOutcome.no, 99, "reject", "consumed credits require rejection"); err != nil || statusValue != "rejected" {
		t.Fatalf("reject refund status=%s err=%v", statusValue, err)
	}
	var restored struct {
		Status           string
		RemainingCredits uint64
	}
	if err := db.Table("ai_credit_grants").Where("owner_type = 'tenant' AND owner_id = 11").
		Select("status, remaining_credits").Scan(&restored).Error; err != nil {
		t.Fatal(err)
	}
	if restored.Status != "active" || restored.RemainingCredits != 27 {
		t.Fatalf("restored grant=%+v", restored)
	}

	successOrderNo, _ := seedRefundOrder(t, db, refundNow, 12, "SUCCESS", 25)
	successAlipay := &refundRaceAlipay{}
	successCommerce, err := appservice.NewCommerce(db, successAlipay)
	if err != nil {
		t.Fatal(err)
	}
	successRefundNo, successStatus, _, err := successCommerce.RequestRefund(
		context.Background(), model.Owner{Type: model.OwnerTenant, ID: 12}, 12,
		successOrderNo, "immediate success", "refund-success",
	)
	if err != nil || successStatus != "succeeded" || successAlipay.refundCallCount() != 1 {
		t.Fatalf("immediate refund no=%s status=%s calls=%d err=%v", successRefundNo, successStatus, successAlipay.refundCallCount(), err)
	}
	if _, duplicateStatus, _, err := successCommerce.RequestRefund(
		context.Background(), model.Owner{Type: model.OwnerTenant, ID: 12}, 12,
		successOrderNo, "immediate success", "refund-success",
	); err != nil || duplicateStatus != "succeeded" || successAlipay.refundCallCount() != 1 {
		t.Fatalf("duplicate refund status=%s calls=%d err=%v", duplicateStatus, successAlipay.refundCallCount(), err)
	}

	failedOrderNo, _ := seedRefundOrder(t, db, refundNow, 13, "FAILED", 25)
	failedAlipay := &refundRaceAlipay{refundErr: &payment.APIError{
		Operation: "refund", Code: "40004", SubCode: "ACQ.INVALID_PARAMETER", Message: "Business Failed",
	}}
	failedCommerce, err := appservice.NewCommerce(db, failedAlipay)
	if err != nil {
		t.Fatal(err)
	}
	_, failedStatus, _, err := failedCommerce.RequestRefund(
		context.Background(), model.Owner{Type: model.OwnerTenant, ID: 13}, 13,
		failedOrderNo, "rejected by channel", "refund-failed",
	)
	if err != nil || failedStatus != "failed" {
		t.Fatalf("failed refund status=%s err=%v", failedStatus, err)
	}
	var failedGrantStatus string
	if err := db.Table("ai_credit_grants").Where("owner_type = 'tenant' AND owner_id = 13").
		Select("status").Scan(&failedGrantStatus).Error; err != nil {
		t.Fatal(err)
	}
	if failedGrantStatus != "active" {
		t.Fatalf("failed refund restored grant status=%s", failedGrantStatus)
	}

	unknownOrderNo, _ := seedRefundOrder(t, db, refundNow, 14, "UNKNOWN", 25)
	unknownAlipay := &refundRaceAlipay{refundErr: errors.New("temporary network failure")}
	unknownCommerce, err := appservice.NewCommerce(db, unknownAlipay)
	if err != nil {
		t.Fatal(err)
	}
	unknownRefundNo, unknownStatus, _, err := unknownCommerce.RequestRefund(
		context.Background(), model.Owner{Type: model.OwnerTenant, ID: 14}, 14,
		unknownOrderNo, "unknown then reconciled", "refund-unknown",
	)
	if err != nil || unknownStatus != "unknown" {
		t.Fatalf("unknown refund status=%s err=%v", unknownStatus, err)
	}
	var unknownFrozenStatus string
	if err := db.Table("ai_credit_grants").Where("owner_type = 'tenant' AND owner_id = 14").
		Select("status").Scan(&unknownFrozenStatus).Error; err != nil {
		t.Fatal(err)
	}
	if unknownFrozenStatus != "refund_frozen" {
		t.Fatalf("unknown refund grant status=%s", unknownFrozenStatus)
	}
	unknownAlipay.setRefundBehavior(nil, payment.RefundQueryResult{
		TradeNo: "CHANNEL-UNKNOWN", RefundStatus: "REFUND_SUCCESS", AmountFen: 3900,
	}, nil)
	if err := db.Table("billing_refunds").Where("refund_no = ?", unknownRefundNo).
		Update("next_reconcile_at", refundNow.Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	if err := unknownCommerce.ReconcilePendingRefunds(context.Background(), 10); err != nil {
		t.Fatal(err)
	}
	var unknownFinalStatus string
	if err := db.Table("billing_refunds").Where("refund_no = ?", unknownRefundNo).
		Select("status").Scan(&unknownFinalStatus).Error; err != nil {
		t.Fatal(err)
	}
	var unknownRefundLedgerCount int64
	if err := db.Table("ai_credit_ledger").Where("idempotency_key LIKE ?", "refund:"+unknownRefundNo+":grant:%").
		Count(&unknownRefundLedgerCount).Error; err != nil {
		t.Fatal(err)
	}
	if unknownFinalStatus != "succeeded" || unknownRefundLedgerCount != 1 {
		t.Fatalf("reconciled status=%s refund_ledgers=%d", unknownFinalStatus, unknownRefundLedgerCount)
	}
}

type refundRaceAlipay struct {
	mu          sync.Mutex
	refundCalls int
	refundErr   error
	queryResult payment.RefundQueryResult
	queryErr    error
}

func (f *refundRaceAlipay) Environment() string { return "sandbox" }
func (f *refundRaceAlipay) PayURL(payment.PayRequest, time.Time) (string, error) {
	return "", nil
}
func (f *refundRaceAlipay) VerifyNotification(map[string]string) error { return nil }
func (f *refundRaceAlipay) VerifyReturn(map[string]string) error       { return nil }
func (f *refundRaceAlipay) Refund(context.Context, string, string, string, uint64, time.Time) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refundCalls++
	if f.refundErr != nil {
		return "", f.refundErr
	}
	return "CHANNEL-REFUND", nil
}
func (f *refundRaceAlipay) QueryRefund(_ context.Context, _ string, refundNo string, _ time.Time) (payment.RefundQueryResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := f.queryResult
	if result.RefundNo == "" {
		result.RefundNo = refundNo
	}
	return result, f.queryErr
}
func (f *refundRaceAlipay) Query(context.Context, string, time.Time) (payment.QueryResult, error) {
	return payment.QueryResult{}, nil
}
func (f *refundRaceAlipay) Close(context.Context, string, time.Time) error { return nil }
func (f *refundRaceAlipay) refundCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.refundCalls
}

func (f *refundRaceAlipay) setRefundBehavior(refundErr error, queryResult payment.RefundQueryResult, queryErr error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refundErr = refundErr
	f.queryResult = queryResult
	f.queryErr = queryErr
}

func seedRefundOrder(t *testing.T, db *gorm.DB, now time.Time, ownerID uint64, suffix string, credits uint64) (string, uint64) {
	t.Helper()
	orderNo := "ORDER-REFUND-" + suffix
	merchantOrderNo := "MERCHANT-REFUND-" + suffix
	if err := db.Exec(`INSERT INTO billing_orders
		(order_no, owner_type, owner_id, order_type, status, amount_fen, paid_at, created_at, updated_at)
		VALUES (?, 'tenant', ?, 'purchase', 'paid', 3900, ?, ?, ?)`,
		orderNo, ownerID, now, now, now).Error; err != nil {
		t.Fatal(err)
	}
	var orderID uint64
	if err := db.Raw("SELECT LAST_INSERT_ID()").Scan(&orderID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO billing_payments
		(order_id, merchant_order_no, status, created_at, updated_at)
		VALUES (?, ?, 'succeeded', ?, ?)`, orderID, merchantOrderNo, now, now).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO ai_credit_grants
		(owner_type, owner_id, grant_type, source_type, source_id, total_credits, remaining_credits,
		 valid_from, expires_at, status, created_at, updated_at)
		VALUES ('tenant', ?, 'credit_pack', 'order', ?, ?, ?, ?, ?, 'active', ?, ?)`,
		ownerID, orderID, credits, credits, now.Add(-time.Hour), now.Add(time.Hour), now, now).Error; err != nil {
		t.Fatal(err)
	}
	return orderNo, orderID
}

func allocationReservationForMySQL(now time.Time, index int, credits uint64) model.Reservation {
	return model.Reservation{
		No:     fmt.Sprintf("22222222-2222-2222-2222-%012d", index),
		Owner:  model.Owner{Type: model.OwnerTenant, ID: 9},
		UserID: 7, Capability: "ai.chat", Operation: "hr_chat",
		ProviderKey: "provider", ModelKey: "model",
		IdempotencyKey:  fmt.Sprintf("reserve-concurrent-%d", index),
		ReservedCredits: credits, Mode: model.ModeEnforce,
		Status: "active", ExpiresAt: now.Add(10 * time.Minute), CreatedAt: now,
	}
}

var billingLifecycleTestDDL = []string{
	`CREATE TABLE ai_credit_grants (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		owner_type VARCHAR(16) NOT NULL, owner_id BIGINT UNSIGNED NOT NULL,
		grant_type VARCHAR(24) NOT NULL DEFAULT 'manual_adjustment',
		source_type VARCHAR(24) NOT NULL DEFAULT 'manual', source_id BIGINT UNSIGNED NULL,
		total_credits BIGINT UNSIGNED NOT NULL DEFAULT 0,
		remaining_credits BIGINT UNSIGNED NOT NULL, valid_from DATETIME(3) NOT NULL,
		expires_at DATETIME(3) NULL, status VARCHAR(16) NOT NULL,
		created_at DATETIME(3) NOT NULL, updated_at DATETIME(3) NOT NULL
	) ENGINE=InnoDB`,
	`CREATE TABLE ai_credit_reservations (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, reservation_no CHAR(36) NOT NULL,
		owner_type VARCHAR(16) NOT NULL, owner_id BIGINT UNSIGNED NOT NULL, user_id BIGINT UNSIGNED NOT NULL,
		capability VARCHAR(96) NOT NULL, operation VARCHAR(64) NOT NULL, provider_key VARCHAR(64) NOT NULL,
		model_key VARCHAR(128) NOT NULL, idempotency_key VARCHAR(191) NOT NULL,
		reserved_credits BIGINT UNSIGNED NOT NULL, settled_credits BIGINT UNSIGNED NOT NULL DEFAULT 0,
		status VARCHAR(16) NOT NULL, enforcement_mode VARCHAR(16) NOT NULL,
		expires_at DATETIME(3) NOT NULL, settled_at DATETIME(3) NULL,
		created_at DATETIME(3) NOT NULL, updated_at DATETIME(3) NOT NULL,
		UNIQUE KEY uk_reservation_no (reservation_no),
		UNIQUE KEY uk_owner_idempotency (owner_type, owner_id, idempotency_key)
	) ENGINE=InnoDB`,
	`CREATE TABLE ai_billing_settlement_outbox (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		reservation_no CHAR(36) NOT NULL, status VARCHAR(32) NOT NULL,
		created_at DATETIME(3) NOT NULL, updated_at DATETIME(3) NOT NULL,
		KEY idx_settlement_reservation_status (reservation_no, status)
	) ENGINE=InnoDB`,
	`CREATE TABLE ai_credit_reservation_allocations (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		reservation_id BIGINT UNSIGNED NOT NULL, grant_id BIGINT UNSIGNED NOT NULL,
		reserved_credits BIGINT UNSIGNED NOT NULL, consumed_credits BIGINT UNSIGNED NOT NULL DEFAULT 0,
		released_credits BIGINT UNSIGNED NOT NULL DEFAULT 0,
		created_at DATETIME(3) NOT NULL, updated_at DATETIME(3) NOT NULL,
		UNIQUE KEY uk_allocation_reservation_grant (reservation_id, grant_id)
	) ENGINE=InnoDB`,
	`CREATE TABLE ai_usage_events (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, event_id CHAR(36) NOT NULL,
		owner_type VARCHAR(16) NOT NULL, owner_id BIGINT UNSIGNED NOT NULL, tenant_id BIGINT UNSIGNED NULL,
		user_id BIGINT UNSIGNED NOT NULL, reservation_id BIGINT UNSIGNED NOT NULL, operation VARCHAR(64) NOT NULL,
		provider_key VARCHAR(64) NOT NULL, model_key VARCHAR(128) NOT NULL, provider_call_seq INT UNSIGNED NOT NULL,
		input_tokens BIGINT UNSIGNED NOT NULL, output_tokens BIGINT UNSIGNED NOT NULL,
		cached_input_tokens BIGINT UNSIGNED NOT NULL, supplier_cost_micros BIGINT UNSIGNED NOT NULL,
		credits_charged BIGINT UNSIGNED NOT NULL, usage_source VARCHAR(16) NOT NULL,
		provider_request_id VARCHAR(191) NULL, occurred_at DATETIME(3) NOT NULL, metadata JSON NULL,
		created_at DATETIME(3) NOT NULL,
		UNIQUE KEY uk_provider_call (reservation_id, provider_call_seq)
	) ENGINE=InnoDB`,
	`CREATE TABLE ai_credit_ledger (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, entry_id CHAR(36) NOT NULL,
		owner_type VARCHAR(16) NOT NULL, owner_id BIGINT UNSIGNED NOT NULL, grant_id BIGINT UNSIGNED NULL,
		reservation_id BIGINT UNSIGNED NULL, usage_event_id BIGINT UNSIGNED NULL, entry_type VARCHAR(24) NOT NULL,
		credits_delta BIGINT NOT NULL, balance_after BIGINT NOT NULL, idempotency_key VARCHAR(191) NOT NULL,
		description VARCHAR(500) NULL, created_at DATETIME(3) NOT NULL,
		UNIQUE KEY uk_ledger_owner_idempotency (owner_type, owner_id, idempotency_key)
	) ENGINE=InnoDB`,
	`CREATE TABLE billing_orders (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, order_no VARCHAR(64) NOT NULL,
		owner_type VARCHAR(16) NOT NULL, owner_id BIGINT UNSIGNED NOT NULL, order_type VARCHAR(24) NOT NULL,
		status VARCHAR(24) NOT NULL, amount_fen BIGINT UNSIGNED NOT NULL, paid_at DATETIME(3) NOT NULL,
		created_at DATETIME(3) NOT NULL, updated_at DATETIME(3) NOT NULL,
		UNIQUE KEY uk_test_order_no (order_no)
	) ENGINE=InnoDB`,
	`CREATE TABLE billing_payments (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, order_id BIGINT UNSIGNED NOT NULL,
		merchant_order_no VARCHAR(64) NOT NULL, status VARCHAR(24) NOT NULL,
		created_at DATETIME(3) NOT NULL, updated_at DATETIME(3) NOT NULL
	) ENGINE=InnoDB`,
	`CREATE TABLE billing_refunds (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, refund_no VARCHAR(64) NOT NULL,
		idempotency_key VARCHAR(128) NOT NULL, order_id BIGINT UNSIGNED NOT NULL, payment_id BIGINT UNSIGNED NOT NULL,
		amount_fen BIGINT UNSIGNED NOT NULL, reason VARCHAR(500) NOT NULL, status VARCHAR(24) NOT NULL,
		active_slot TINYINT UNSIGNED NULL, review_mode VARCHAR(16) NOT NULL, requested_by BIGINT UNSIGNED NOT NULL,
		reviewed_by BIGINT UNSIGNED NULL, reviewed_at DATETIME(3) NULL, rejected_reason VARCHAR(500) NULL,
		channel_refund_no VARCHAR(128) NULL, succeeded_at DATETIME(3) NULL, next_reconcile_at DATETIME(3) NULL,
		reconcile_attempts INT UNSIGNED NOT NULL DEFAULT 0, last_error VARCHAR(500) NULL,
		created_at DATETIME(3) NOT NULL, updated_at DATETIME(3) NOT NULL,
		UNIQUE KEY uk_test_refund_no (refund_no),
		UNIQUE KEY uk_test_refund_idempotency (order_id, idempotency_key),
		UNIQUE KEY uk_test_refund_active (payment_id, active_slot)
	) ENGINE=InnoDB`,
	`CREATE TABLE billing_subscriptions (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		owner_type VARCHAR(16) NOT NULL DEFAULT 'tenant', owner_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
		activated_by_order_id BIGINT UNSIGNED NULL, status VARCHAR(24) NOT NULL,
		current_period_start DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
		current_period_end DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
		updated_at DATETIME(3) NOT NULL
	) ENGINE=InnoDB`,
}
