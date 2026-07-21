package service

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"smart-recruit-billing-service/internal/infrastructure/payment"
)

func TestProratedCeil(t *testing.T) {
	start := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(30 * 24 * time.Hour)

	tests := []struct {
		name  string
		value uint64
		now   time.Time
		want  uint64
	}{
		{name: "full period", value: 100, now: start, want: 100},
		{name: "half period", value: 101, now: start.Add(15 * 24 * time.Hour), want: 51},
		{name: "expired", value: 100, now: end, want: 0},
		{name: "zero value", value: 0, now: start, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := proratedCeil(tt.value, start, end, tt.now); got != tt.want {
				t.Fatalf("proratedCeil() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCommerceOrderEnvironmentMapsPaymentEnvironmentColumn(t *testing.T) {
	parsed, err := schema.Parse(&commerceOrderRow{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	field := parsed.LookUpField("Environment")
	if field == nil || field.DBName != "payment_environment" {
		t.Fatalf("Environment DB column = %#v, want payment_environment", field)
	}
}

func TestShanghaiBillingMonthUsesCalendarBoundary(t *testing.T) {
	now := time.Date(2026, time.July, 31, 18, 30, 0, 0, time.UTC)
	start, end := shanghaiBillingMonth(now)

	wantStart := time.Date(2026, time.July, 31, 16, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, time.August, 31, 16, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) {
		t.Fatalf("start = %s, want %s", start, wantStart)
	}
	if !end.Equal(wantEnd) {
		t.Fatalf("end = %s, want %s", end, wantEnd)
	}
}

func TestResolveAlipayCloseFailureTreatsMissingTradeAsClosed(t *testing.T) {
	closeErr := errors.New("sandbox close returned HTML")
	queryErr := &payment.APIError{Operation: "query", SubCode: "ACQ.TRADE_NOT_EXIST", Message: "Business Failed", SubMessage: "交易不存在"}

	resolution, err := resolveAlipayCloseFailure(closeErr, payment.QueryResult{}, queryErr)
	if err != nil {
		t.Fatalf("resolve missing trade: %v", err)
	}
	if resolution != alipayTradeClosed {
		t.Fatalf("resolution = %v, want closed", resolution)
	}
}

func TestResolveAlipayCloseFailurePreservesPaidTrade(t *testing.T) {
	result := payment.QueryResult{TradeNo: "20260720001", TradeStatus: "TRADE_SUCCESS", AmountFen: 3900}

	resolution, err := resolveAlipayCloseFailure(errors.New("close rejected"), result, nil)
	if err != nil {
		t.Fatalf("resolve paid trade: %v", err)
	}
	if resolution != alipayTradePaid {
		t.Fatalf("resolution = %v, want paid", resolution)
	}
}

func TestResolveAlipayCloseFailureRejectsLiveUnclosedTrade(t *testing.T) {
	result := payment.QueryResult{TradeStatus: "WAIT_BUYER_PAY"}

	if _, err := resolveAlipayCloseFailure(errors.New("close rejected"), result, nil); err == nil {
		t.Fatal("expected a live unclosed trade to block replacement")
	}
}

func TestPaymentSettlementAcceptsRecoveryStates(t *testing.T) {
	for _, status := range []string{"created", "pending", "closing", "unknown", "closed", "succeeded"} {
		if !isPaymentSettleable(status) {
			t.Fatalf("status %q must remain settleable after Alipay confirms payment", status)
		}
	}
	for _, status := range []string{"failed", "refunded", ""} {
		if isPaymentSettleable(status) {
			t.Fatalf("status %q must not be settleable", status)
		}
	}
}

func TestNoOpConflictProducesValidMySQLAssignmentForMapCreate(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8mb4&parseTime=True&loc=Local",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("open dry-run database: %v", err)
	}

	statement := db.Table("billing_webhook_events").
		Clauses(noOpConflict("event_key", "channel", "payment_environment", "event_key")).
		Create(map[string]any{"channel": "alipay", "payment_environment": "sandbox", "event_key": "query:1"})
	if statement.Error != nil {
		t.Fatalf("build insert: %v", statement.Error)
	}
	sql := statement.Statement.SQL.String()
	if strings.HasSuffix(strings.TrimSpace(sql), "UPDATE") {
		t.Fatalf("generated invalid dangling duplicate-key update: %s", sql)
	}
	if !strings.Contains(sql, "`event_key`=VALUES(`event_key`)") {
		t.Fatalf("generated SQL lacks deterministic no-op assignment: %s", sql)
	}
}
