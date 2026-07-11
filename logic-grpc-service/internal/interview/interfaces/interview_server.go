package interfaces

import (
	"context"
	"fmt"

	"logic-grpc-service/recruitment/pb"
)

type InterviewServer struct {
	pb.UnimplementedInterviewServiceServer
	api InterviewAPI
}

func NewInterviewServer(api InterviewAPI) (*InterviewServer, error) {
	if api == nil {
		return nil, fmt.Errorf("interview api is required")
	}
	return &InterviewServer{api: api}, nil
}

func (s *InterviewServer) ScheduleInterview(ctx context.Context, req *pb.ScheduleInterviewRequest) (*pb.ScheduleInterviewResponse, error) {
	return s.api.ScheduleInterview(ctx, req)
}

func (s *InterviewServer) UpdateInterview(ctx context.Context, req *pb.UpdateInterviewRequest) (*pb.CommonResponse, error) {
	return s.api.UpdateInterview(ctx, req)
}

func (s *InterviewServer) CancelInterview(ctx context.Context, req *pb.CancelInterviewRequest) (*pb.CommonResponse, error) {
	return s.api.CancelInterview(ctx, req)
}

func (s *InterviewServer) GetInterview(ctx context.Context, req *pb.GetInterviewRequest) (*pb.GetInterviewResponse, error) {
	return s.api.GetInterview(ctx, req)
}

func (s *InterviewServer) ListInterviewers(ctx context.Context, req *pb.ListInterviewersRequest) (*pb.ListInterviewersResponse, error) {
	return s.api.ListInterviewers(ctx, req)
}

func (s *InterviewServer) ListApplicationInterviews(ctx context.Context, req *pb.ListApplicationInterviewsRequest) (*pb.ListApplicationInterviewsResponse, error) {
	return s.api.ListApplicationInterviews(ctx, req)
}

func (s *InterviewServer) ListMyInterviews(ctx context.Context, req *pb.ListMyInterviewsRequest) (*pb.ListMyInterviewsResponse, error) {
	return s.api.ListMyInterviews(ctx, req)
}

func (s *InterviewServer) ListCandidateInterviews(ctx context.Context, req *pb.ListCandidateInterviewsRequest) (*pb.ListCandidateInterviewsResponse, error) {
	return s.api.ListCandidateInterviews(ctx, req)
}

func (s *InterviewServer) SubmitFeedback(ctx context.Context, req *pb.SubmitFeedbackRequest) (*pb.CommonResponse, error) {
	return s.api.SubmitFeedback(ctx, req)
}

func (s *InterviewServer) GetFeedback(ctx context.Context, req *pb.GetFeedbackRequest) (*pb.GetFeedbackResponse, error) {
	return s.api.GetFeedback(ctx, req)
}

func (s *InterviewServer) BatchCancelInterviews(ctx context.Context, req *pb.BatchCancelInterviewsRequest) (*pb.BatchCancelInterviewsResponse, error) {
	return s.api.BatchCancelInterviews(ctx, req)
}
