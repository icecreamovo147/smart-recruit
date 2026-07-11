package interfaces

import (
	"context"

	"logic-grpc-service/recruitment/pb"
)

// InterviewAPI is the Interview-owned gRPC contract implemented by the current InterviewService.
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
