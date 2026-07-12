package proto

import (
	"testing"

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
