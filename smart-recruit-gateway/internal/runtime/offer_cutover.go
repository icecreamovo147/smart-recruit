package runtime

import (
	"context"
	"fmt"
)

var OfferSmokeMethods = []string{
	"OfferService.CreateOffer",
	"OfferService.SendOffer",
	"OfferService.AcceptOffer",
	"OfferService.RejectOffer",
	"OfferService.WithdrawOffer",
	"OfferService.ListOffersByApplication",
	"OfferService.ListMyOffers",
	"OfferService.ListOfferEvents",
}

type OfferCutoverPlan struct {
	Mode         string
	Target       string
	ReadyTargets []RouteEntry
	SmokeMethods []string
}

func BuildOfferCutoverPlan(ctx context.Context, table RouteTable, logicTarget string, resolver TargetResolver) (OfferCutoverPlan, error) {
	resolved, err := table.ResolveTargets(ctx, logicTarget, resolver)
	if err != nil {
		return OfferCutoverPlan{}, err
	}
	entry, ok := resolved.Entry("offer")
	if !ok {
		return OfferCutoverPlan{}, fmt.Errorf("offer route entry is required")
	}
	return OfferCutoverPlan{
		Mode:         entry.Mode,
		Target:       entry.Target,
		ReadyTargets: resolved.ReadyTargets(logicTarget),
		SmokeMethods: append([]string(nil), OfferSmokeMethods...),
	}, nil
}

func (plan OfferCutoverPlan) ValidateCutover(offerTarget string) error {
	if plan.Mode != "offer" {
		return fmt.Errorf("offer route mode = %q, want offer", plan.Mode)
	}
	if plan.Target != offerTarget {
		return fmt.Errorf("offer route target = %q, want %q", plan.Target, offerTarget)
	}
	if len(plan.SmokeMethods) == 0 {
		return fmt.Errorf("offer smoke methods are required")
	}
	for _, target := range plan.ReadyTargets {
		if target.Service == "offer" && target.Target == offerTarget {
			return nil
		}
	}
	return fmt.Errorf("offer ready target %q is missing", offerTarget)
}

func (plan OfferCutoverPlan) ValidateRollback(logicTarget string) error {
	if plan.Mode != LogicService {
		return fmt.Errorf("offer rollback mode = %q, want logic", plan.Mode)
	}
	if plan.Target != logicTarget {
		return fmt.Errorf("offer rollback target = %q, want %q", plan.Target, logicTarget)
	}
	for _, target := range plan.ReadyTargets {
		if target.Service == "offer" {
			return fmt.Errorf("offer should not be a separate ready target during logic rollback")
		}
	}
	return nil
}
