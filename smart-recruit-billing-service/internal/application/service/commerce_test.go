package service

import (
	"sync"
	"testing"
	"time"

	"gorm.io/gorm/schema"
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
