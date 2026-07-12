package interfaces

import (
	"context"
	"fmt"

	"logic-grpc-service/recruitment/pb"
)

type CandidateServer struct {
	pb.UnimplementedCandidateServiceServer
	api CandidateProfileAPI
}

func NewCandidateServer(api CandidateProfileAPI) (*CandidateServer, error) {
	if api == nil {
		return nil, fmt.Errorf("recruitment candidate profile api is required")
	}
	return &CandidateServer{api: api}, nil
}

func (s *CandidateServer) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	return s.api.GetProfile(ctx, req)
}

func (s *CandidateServer) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.GetProfileResponse, error) {
	return s.api.UpdateProfile(ctx, req)
}

func (s *CandidateServer) GetResume(ctx context.Context, req *pb.GetResumeRequest) (*pb.GetResumeResponse, error) {
	return s.api.GetResume(ctx, req)
}

func (s *CandidateServer) PresignResumeUpload(ctx context.Context, req *pb.PresignResumeUploadRequest) (*pb.PresignResumeUploadResponse, error) {
	return s.api.PresignResumeUpload(ctx, req)
}

func (s *CandidateServer) ConfirmResumeUpload(ctx context.Context, req *pb.ConfirmResumeUploadRequest) (*pb.ConfirmResumeUploadResponse, error) {
	return s.api.ConfirmResumeUpload(ctx, req)
}
