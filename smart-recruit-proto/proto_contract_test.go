package proto

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	pb "smart-recruit-proto/recruitment/pb"
)

func TestGeneratedContractsAreImportable(t *testing.T) {
	if pb.AuthService_ServiceDesc.ServiceName != "recruitment.AuthService" {
		t.Fatalf("unexpected auth service name: %s", pb.AuthService_ServiceDesc.ServiceName)
	}
	if pb.JobService_ServiceDesc.ServiceName != "recruitment.JobService" {
		t.Fatalf("unexpected job service name: %s", pb.JobService_ServiceDesc.ServiceName)
	}
}

func TestContextUsageAdditiveFieldNumbers(t *testing.T) {
	breakdown := (&pb.ContextUsageBreakdown{}).ProtoReflect().Descriptor().Fields()
	if got := breakdown.ByName("tool_schema_tokens").Number(); got != 8 {
		t.Fatalf("tool_schema_tokens number = %d, want 8", got)
	}
	if got := breakdown.ByName("protocol_overhead_tokens").Number(); got != 9 {
		t.Fatalf("protocol_overhead_tokens number = %d, want 9", got)
	}
	usage := (&pb.ContextUsageInfo{}).ProtoReflect().Descriptor().Fields()
	want := map[string]int32{
		"input_budget_tokens": 15, "safety_margin_tokens": 16, "budget_usage_ratio": 17,
		"budget_status": 18, "included_message_count": 19, "omitted_message_count": 20,
		"summary_applied": 21,
	}
	for name, number := range want {
		field := usage.ByName(protoreflect.Name(name))
		if field == nil || int32(field.Number()) != number {
			t.Fatalf("%s number = %v, want %d", name, field, number)
		}
	}
}
