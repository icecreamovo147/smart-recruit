package runtime

import (
	"context"
	"testing"
)

func TestOfferCutoverPlanCoversOfferLifecycleSmoke(t *testing.T) {
	table, err := NewRouteTable(map[string]string{"offer": "offer"})
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	plan, err := BuildOfferCutoverPlan(context.Background(), table, "logic:50051", StaticTargetResolver{
		"offer": "offer:50064",
	})
	if err != nil {
		t.Fatalf("BuildOfferCutoverPlan returned error: %v", err)
	}
	if err := plan.ValidateCutover("offer:50064"); err != nil {
		t.Fatalf("ValidateCutover returned error: %v", err)
	}
	required := map[string]bool{
		"OfferService.CreateOffer":             false,
		"OfferService.SendOffer":               false,
		"OfferService.AcceptOffer":             false,
		"OfferService.RejectOffer":             false,
		"OfferService.WithdrawOffer":           false,
		"OfferService.ListOffersByApplication": false,
		"OfferService.ListMyOffers":            false,
		"OfferService.ListOfferEvents":         false,
	}
	for _, method := range plan.SmokeMethods {
		if _, ok := required[method]; ok {
			required[method] = true
		}
	}
	for method, seen := range required {
		if !seen {
			t.Fatalf("offer smoke plan missing %s", method)
		}
	}
}

func TestOfferRollbackPlanUsesLogicOnly(t *testing.T) {
	table, err := NewRouteTable(map[string]string{"offer": "logic"})
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	plan, err := BuildOfferCutoverPlan(context.Background(), table, "logic:50051", nil)
	if err != nil {
		t.Fatalf("BuildOfferCutoverPlan returned error: %v", err)
	}
	if err := plan.ValidateRollback("logic:50051"); err != nil {
		t.Fatalf("ValidateRollback returned error: %v", err)
	}
}
