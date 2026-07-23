package model

import (
	"errors"
	"math"
	"testing"
)

func TestRateCardPriceRoundsSupplierCostAndCreditsUp(t *testing.T) {
	rate := RateCard{InputMicrosPer1K: 1001, OutputMicrosPer1K: 2001, CreditMicros: 1000}
	cost, credits, err := rate.Price(ProviderUsage{InputTokens: 1, OutputTokens: 1})
	if err != nil {
		t.Fatal(err)
	}
	if cost != 5 || credits != 1 {
		t.Fatalf("cost = %d, credits = %d", cost, credits)
	}
}

func TestRateCardPriceSeparatesCachedInputTokens(t *testing.T) {
	rate := RateCard{
		InputMicrosPer1K:       1000,
		CachedInputMicrosPer1K: 200,
		OutputMicrosPer1K:      2000,
		CreditMicros:           100,
	}
	cost, credits, err := rate.Price(ProviderUsage{
		InputTokens:       1000,
		CachedInputTokens: 1000,
		OutputTokens:      500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cost != 1200 || credits != 12 {
		t.Fatalf("cost = %d, credits = %d, want 1200/12", cost, credits)
	}
}

func TestRateCardPricePartialCache(t *testing.T) {
	rate := RateCard{
		InputMicrosPer1K:       1000,
		CachedInputMicrosPer1K: 200,
		OutputMicrosPer1K:      2000,
		CreditMicros:           100,
	}
	cost, _, err := rate.Price(ProviderUsage{InputTokens: 1000, CachedInputTokens: 400, OutputTokens: 500})
	if err != nil {
		t.Fatal(err)
	}
	if cost != 1680 {
		t.Fatalf("cost = %d, want 1680", cost)
	}
}

func TestRateCardPriceRejectsInvalidCachedTokens(t *testing.T) {
	_, _, err := (RateCard{InputMicrosPer1K: 1000, CachedInputMicrosPer1K: 200, CreditMicros: 100}).
		Price(ProviderUsage{InputTokens: 99, CachedInputTokens: 100})
	if !errors.Is(err, ErrInvalidProviderUsage) {
		t.Fatalf("error = %v, want ErrInvalidProviderUsage", err)
	}
}

func TestRateCardPriceRejectsArithmeticOverflow(t *testing.T) {
	rate := RateCard{InputMicrosPer1K: 2, CreditMicros: 1}
	if _, _, err := rate.Price(ProviderUsage{InputTokens: math.MaxUint64}); !errors.Is(err, ErrPriceOverflow) {
		t.Fatalf("multiplication error = %v, want ErrPriceOverflow", err)
	}
	if _, err := checkedAdd(math.MaxUint64, 1); !errors.Is(err, ErrPriceOverflow) {
		t.Fatalf("addition error = %v, want ErrPriceOverflow", err)
	}
}
