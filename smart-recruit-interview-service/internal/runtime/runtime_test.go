package runtime

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

func TestRuntimeRegistersInterviewGRPCService(t *testing.T) {
	runtime, err := New(Deps{Interview: fakeInterviewAPI{}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	server := grpc.NewServer()
	t.Cleanup(server.Stop)
	if err := runtime.RegisterGRPC(server); err != nil {
		t.Fatalf("RegisterGRPC returned error: %v", err)
	}
	services := server.GetServiceInfo()
	if _, ok := services[pb.InterviewService_ServiceDesc.ServiceName]; !ok {
		t.Fatalf("missing registered service %s", pb.InterviewService_ServiceDesc.ServiceName)
	}
}

func TestRuntimeRequiresInterviewDependency(t *testing.T) {
	if _, err := New(Deps{}); err == nil {
		t.Fatal("expected missing interview dependency error")
	}
}

type fakeInterviewAPI struct{}

func (fakeInterviewAPI) ScheduleInterview(context.Context, *pb.ScheduleInterviewRequest) (*pb.ScheduleInterviewResponse, error) {
	return &pb.ScheduleInterviewResponse{Code: errs.OK}, nil
}

func (fakeInterviewAPI) ListInterviewers(context.Context, *pb.ListInterviewersRequest) (*pb.ListInterviewersResponse, error) {
	return &pb.ListInterviewersResponse{Code: errs.OK}, nil
}

func (fakeInterviewAPI) UpdateInterview(context.Context, *pb.UpdateInterviewRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeInterviewAPI) CancelInterview(context.Context, *pb.CancelInterviewRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeInterviewAPI) BatchCancelInterviews(context.Context, *pb.BatchCancelInterviewsRequest) (*pb.BatchCancelInterviewsResponse, error) {
	return &pb.BatchCancelInterviewsResponse{Code: errs.OK}, nil
}

func (fakeInterviewAPI) GetInterview(context.Context, *pb.GetInterviewRequest) (*pb.GetInterviewResponse, error) {
	return &pb.GetInterviewResponse{Code: errs.OK}, nil
}

func (fakeInterviewAPI) ListApplicationInterviews(context.Context, *pb.ListApplicationInterviewsRequest) (*pb.ListApplicationInterviewsResponse, error) {
	return &pb.ListApplicationInterviewsResponse{Code: errs.OK}, nil
}

func (fakeInterviewAPI) ListMyInterviews(context.Context, *pb.ListMyInterviewsRequest) (*pb.ListMyInterviewsResponse, error) {
	return &pb.ListMyInterviewsResponse{Code: errs.OK}, nil
}

func (fakeInterviewAPI) ListCandidateInterviews(context.Context, *pb.ListCandidateInterviewsRequest) (*pb.ListCandidateInterviewsResponse, error) {
	return &pb.ListCandidateInterviewsResponse{Code: errs.OK}, nil
}

func (fakeInterviewAPI) SubmitFeedback(context.Context, *pb.SubmitFeedbackRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeInterviewAPI) GetFeedback(context.Context, *pb.GetFeedbackRequest) (*pb.GetFeedbackResponse, error) {
	return &pb.GetFeedbackResponse{Code: errs.OK}, nil
}
