package persistence

import (
	"context"
	"os"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"smart-recruit-billing-service/internal/domain/model"
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
}

var billingLifecycleTestDDL = []string{
	`CREATE TABLE ai_credit_grants (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		owner_type VARCHAR(16) NOT NULL, owner_id BIGINT UNSIGNED NOT NULL,
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
}
