package runtime

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/recruitment/pb"
)

func TestRuntimeRegistersInterviewGRPCService(t *testing.T) {
	t.Parallel()

	runtime, err := New(Deps{Interview: &runtimeInterviewAPI{}})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	server := grpc.NewServer()
	t.Cleanup(server.Stop)

	if err := runtime.RegisterGRPC(server); err != nil {
		t.Fatalf("RegisterGRPC() error = %v", err)
	}

	services := server.GetServiceInfo()
	if _, ok := services[pb.InterviewService_ServiceDesc.ServiceName]; !ok {
		t.Fatalf("registered services missing %s: %v", pb.InterviewService_ServiceDesc.ServiceName, services)
	}
}

func TestRuntimeRequiresInterviewDependency(t *testing.T) {
	t.Parallel()

	if _, err := New(Deps{}); err == nil {
		t.Fatal("New(Deps{}) error = nil")
	}
}

func TestRuntimeRejectsNilRegistrar(t *testing.T) {
	t.Parallel()

	runtime, err := New(Deps{Interview: &runtimeInterviewAPI{}})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := runtime.RegisterGRPC(nil); err == nil {
		t.Fatal("RegisterGRPC(nil) error = nil")
	}
}

type runtimeInterviewAPI struct{}

func (runtimeInterviewAPI) ScheduleInterview(context.Context, *pb.ScheduleInterviewRequest) (*pb.ScheduleInterviewResponse, error) {
	return &pb.ScheduleInterviewResponse{Code: errs.OK}, nil
}

func (runtimeInterviewAPI) ListInterviewers(context.Context, *pb.ListInterviewersRequest) (*pb.ListInterviewersResponse, error) {
	return &pb.ListInterviewersResponse{Code: errs.OK}, nil
}

func (runtimeInterviewAPI) UpdateInterview(context.Context, *pb.UpdateInterviewRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeInterviewAPI) CancelInterview(context.Context, *pb.CancelInterviewRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeInterviewAPI) BatchCancelInterviews(context.Context, *pb.BatchCancelInterviewsRequest) (*pb.BatchCancelInterviewsResponse, error) {
	return &pb.BatchCancelInterviewsResponse{Code: errs.OK}, nil
}

func (runtimeInterviewAPI) GetInterview(context.Context, *pb.GetInterviewRequest) (*pb.GetInterviewResponse, error) {
	return &pb.GetInterviewResponse{Code: errs.OK}, nil
}

func (runtimeInterviewAPI) ListApplicationInterviews(context.Context, *pb.ListApplicationInterviewsRequest) (*pb.ListApplicationInterviewsResponse, error) {
	return &pb.ListApplicationInterviewsResponse{Code: errs.OK}, nil
}

func (runtimeInterviewAPI) ListMyInterviews(context.Context, *pb.ListMyInterviewsRequest) (*pb.ListMyInterviewsResponse, error) {
	return &pb.ListMyInterviewsResponse{Code: errs.OK}, nil
}

func (runtimeInterviewAPI) ListCandidateInterviews(context.Context, *pb.ListCandidateInterviewsRequest) (*pb.ListCandidateInterviewsResponse, error) {
	return &pb.ListCandidateInterviewsResponse{Code: errs.OK}, nil
}

func (runtimeInterviewAPI) SubmitFeedback(context.Context, *pb.SubmitFeedbackRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeInterviewAPI) GetFeedback(context.Context, *pb.GetFeedbackRequest) (*pb.GetFeedbackResponse, error) {
	return &pb.GetFeedbackResponse{Code: errs.OK}, nil
}
