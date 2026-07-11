package interfaces

import (
	"context"
	"testing"

	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/recruitment/pb"
)

func TestInterviewServerForwardsInterviewOwnedMethods(t *testing.T) {
	t.Parallel()

	api := &recordingInterviewAPI{}
	server, err := NewInterviewServer(api)
	if err != nil {
		t.Fatalf("NewInterviewServer() error = %v", err)
	}

	if _, err := server.ScheduleInterview(context.Background(), &pb.ScheduleInterviewRequest{}); err != nil {
		t.Fatalf("ScheduleInterview() error = %v", err)
	}
	if _, err := server.SubmitFeedback(context.Background(), &pb.SubmitFeedbackRequest{}); err != nil {
		t.Fatalf("SubmitFeedback() error = %v", err)
	}
	if api.scheduleCalls != 1 {
		t.Fatalf("scheduleCalls = %d, want 1", api.scheduleCalls)
	}
	if api.feedbackCalls != 1 {
		t.Fatalf("feedbackCalls = %d, want 1", api.feedbackCalls)
	}
}

func TestInterviewServerRequiresDependency(t *testing.T) {
	t.Parallel()

	if _, err := NewInterviewServer(nil); err == nil {
		t.Fatal("NewInterviewServer(nil) error = nil")
	}
}

type recordingInterviewAPI struct {
	scheduleCalls int
	feedbackCalls int
}

func (a *recordingInterviewAPI) ScheduleInterview(context.Context, *pb.ScheduleInterviewRequest) (*pb.ScheduleInterviewResponse, error) {
	a.scheduleCalls++
	return &pb.ScheduleInterviewResponse{Code: errs.OK}, nil
}

func (a *recordingInterviewAPI) ListInterviewers(context.Context, *pb.ListInterviewersRequest) (*pb.ListInterviewersResponse, error) {
	return &pb.ListInterviewersResponse{Code: errs.OK}, nil
}

func (a *recordingInterviewAPI) UpdateInterview(context.Context, *pb.UpdateInterviewRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (a *recordingInterviewAPI) CancelInterview(context.Context, *pb.CancelInterviewRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (a *recordingInterviewAPI) BatchCancelInterviews(context.Context, *pb.BatchCancelInterviewsRequest) (*pb.BatchCancelInterviewsResponse, error) {
	return &pb.BatchCancelInterviewsResponse{Code: errs.OK}, nil
}

func (a *recordingInterviewAPI) GetInterview(context.Context, *pb.GetInterviewRequest) (*pb.GetInterviewResponse, error) {
	return &pb.GetInterviewResponse{Code: errs.OK}, nil
}

func (a *recordingInterviewAPI) ListApplicationInterviews(context.Context, *pb.ListApplicationInterviewsRequest) (*pb.ListApplicationInterviewsResponse, error) {
	return &pb.ListApplicationInterviewsResponse{Code: errs.OK}, nil
}

func (a *recordingInterviewAPI) ListMyInterviews(context.Context, *pb.ListMyInterviewsRequest) (*pb.ListMyInterviewsResponse, error) {
	return &pb.ListMyInterviewsResponse{Code: errs.OK}, nil
}

func (a *recordingInterviewAPI) ListCandidateInterviews(context.Context, *pb.ListCandidateInterviewsRequest) (*pb.ListCandidateInterviewsResponse, error) {
	return &pb.ListCandidateInterviewsResponse{Code: errs.OK}, nil
}

func (a *recordingInterviewAPI) SubmitFeedback(context.Context, *pb.SubmitFeedbackRequest) (*pb.CommonResponse, error) {
	a.feedbackCalls++
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (a *recordingInterviewAPI) GetFeedback(context.Context, *pb.GetFeedbackRequest) (*pb.GetFeedbackResponse, error) {
	return &pb.GetFeedbackResponse{Code: errs.OK}, nil
}
