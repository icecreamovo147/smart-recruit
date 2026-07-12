package runtime

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"smart-recruit-proto/recruitment/pb"
)

const ServiceName = "interview-service"

type InterviewAPI interface {
	ScheduleInterview(context.Context, *pb.ScheduleInterviewRequest) (*pb.ScheduleInterviewResponse, error)
	ListInterviewers(context.Context, *pb.ListInterviewersRequest) (*pb.ListInterviewersResponse, error)
	UpdateInterview(context.Context, *pb.UpdateInterviewRequest) (*pb.CommonResponse, error)
	CancelInterview(context.Context, *pb.CancelInterviewRequest) (*pb.CommonResponse, error)
	BatchCancelInterviews(context.Context, *pb.BatchCancelInterviewsRequest) (*pb.BatchCancelInterviewsResponse, error)
	GetInterview(context.Context, *pb.GetInterviewRequest) (*pb.GetInterviewResponse, error)
	ListApplicationInterviews(context.Context, *pb.ListApplicationInterviewsRequest) (*pb.ListApplicationInterviewsResponse, error)
	ListMyInterviews(context.Context, *pb.ListMyInterviewsRequest) (*pb.ListMyInterviewsResponse, error)
	ListCandidateInterviews(context.Context, *pb.ListCandidateInterviewsRequest) (*pb.ListCandidateInterviewsResponse, error)
	SubmitFeedback(context.Context, *pb.SubmitFeedbackRequest) (*pb.CommonResponse, error)
	GetFeedback(context.Context, *pb.GetFeedbackRequest) (*pb.GetFeedbackResponse, error)
}

type Deps struct {
	Interview InterviewAPI
}

type Runtime struct {
	Interview pb.InterviewServiceServer
}

func New(deps Deps) (*Runtime, error) {
	if deps.Interview == nil {
		return nil, fmt.Errorf("interview api is required")
	}
	return &Runtime{
		Interview: interviewServer{api: deps.Interview},
	}, nil
}

func (r *Runtime) RegisterGRPC(registrar grpc.ServiceRegistrar) error {
	if registrar == nil {
		return fmt.Errorf("grpc service registrar is required")
	}
	if r == nil || r.Interview == nil {
		return fmt.Errorf("interview runtime is not initialized")
	}
	pb.RegisterInterviewServiceServer(registrar, r.Interview)
	return nil
}

type interviewServer struct {
	pb.UnimplementedInterviewServiceServer
	api InterviewAPI
}

func (s interviewServer) ScheduleInterview(ctx context.Context, req *pb.ScheduleInterviewRequest) (*pb.ScheduleInterviewResponse, error) {
	return s.api.ScheduleInterview(ctx, req)
}

func (s interviewServer) ListInterviewers(ctx context.Context, req *pb.ListInterviewersRequest) (*pb.ListInterviewersResponse, error) {
	return s.api.ListInterviewers(ctx, req)
}

func (s interviewServer) UpdateInterview(ctx context.Context, req *pb.UpdateInterviewRequest) (*pb.CommonResponse, error) {
	return s.api.UpdateInterview(ctx, req)
}

func (s interviewServer) CancelInterview(ctx context.Context, req *pb.CancelInterviewRequest) (*pb.CommonResponse, error) {
	return s.api.CancelInterview(ctx, req)
}

func (s interviewServer) BatchCancelInterviews(ctx context.Context, req *pb.BatchCancelInterviewsRequest) (*pb.BatchCancelInterviewsResponse, error) {
	return s.api.BatchCancelInterviews(ctx, req)
}

func (s interviewServer) GetInterview(ctx context.Context, req *pb.GetInterviewRequest) (*pb.GetInterviewResponse, error) {
	return s.api.GetInterview(ctx, req)
}

func (s interviewServer) ListApplicationInterviews(ctx context.Context, req *pb.ListApplicationInterviewsRequest) (*pb.ListApplicationInterviewsResponse, error) {
	return s.api.ListApplicationInterviews(ctx, req)
}

func (s interviewServer) ListMyInterviews(ctx context.Context, req *pb.ListMyInterviewsRequest) (*pb.ListMyInterviewsResponse, error) {
	return s.api.ListMyInterviews(ctx, req)
}

func (s interviewServer) ListCandidateInterviews(ctx context.Context, req *pb.ListCandidateInterviewsRequest) (*pb.ListCandidateInterviewsResponse, error) {
	return s.api.ListCandidateInterviews(ctx, req)
}

func (s interviewServer) SubmitFeedback(ctx context.Context, req *pb.SubmitFeedbackRequest) (*pb.CommonResponse, error) {
	return s.api.SubmitFeedback(ctx, req)
}

func (s interviewServer) GetFeedback(ctx context.Context, req *pb.GetFeedbackRequest) (*pb.GetFeedbackResponse, error) {
	return s.api.GetFeedback(ctx, req)
}
