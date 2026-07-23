package observability

import (
	"strings"
	"testing"
)

func TestBillingMetricsUseLowCardinalityLabels(t *testing.T) {
	registry := NewRegistry("billing")
	registry.RecordBillingEvent("alipay_webhook", "processed")
	registry.SetBillingGauge("payment", "unknown", 2)
	output := registry.Prometheus()
	if !strings.Contains(output, `smart_recruit_billing_events_total{service="billing",operation="alipay_webhook",outcome="processed"} 1`) {
		t.Fatalf("missing billing counter: %s", output)
	}
	if !strings.Contains(output, `smart_recruit_billing_state_count{service="billing",resource="payment",state="unknown"} 2`) {
		t.Fatalf("missing billing gauge: %s", output)
	}
}
