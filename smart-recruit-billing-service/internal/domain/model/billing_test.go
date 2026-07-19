package model

import "testing"

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
