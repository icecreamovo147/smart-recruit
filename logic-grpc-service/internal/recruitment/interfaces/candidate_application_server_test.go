package interfaces

import (
	"context"
	"testing"

	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/recruitment/pb"
)

func TestCandidateServerForwardsRecruitmentOwnedMethods(t *testing.T) {
	t.Parallel()

	api := &recordingCandidateAPI{}
	server, err := NewCandidateServer(api)
	if err != nil {
		t.Fatalf("NewCandidateServer() error = %v", err)
	}

	if _, err := server.ConfirmResumeUpload(context.Background(), &pb.ConfirmResumeUploadRequest{}); err != nil {
		t.Fatalf("ConfirmResumeUpload() error = %v", err)
	}
	if api.confirmResumeUploadCalls != 1 {
		t.Fatalf("confirmResumeUploadCalls = %d, want 1", api.confirmResumeUploadCalls)
	}
}

func TestApplicationServerForwardsRecruitmentOwnedMethods(t *testing.T) {
	t.Parallel()

	api := &recordingApplicationAPI{}
	server, err := NewApplicationServer(api)
	if err != nil {
		t.Fatalf("NewApplicationServer() error = %v", err)
	}

	if _, err := server.UpdateApplicationStatus(context.Background(), &pb.UpdateApplicationStatusRequest{}); err != nil {
		t.Fatalf("UpdateApplicationStatus() error = %v", err)
	}
	if api.updateApplicationStatusCalls != 1 {
		t.Fatalf("updateApplicationStatusCalls = %d, want 1", api.updateApplicationStatusCalls)
	}
}

func TestCandidateAndApplicationServersRequireDependencies(t *testing.T) {
	t.Parallel()

	if _, err := NewCandidateServer(nil); err == nil {
		t.Fatal("NewCandidateServer(nil) error = nil")
	}
	if _, err := NewApplicationServer(nil); err == nil {
		t.Fatal("NewApplicationServer(nil) error = nil")
	}
}

type recordingCandidateAPI struct {
	confirmResumeUploadCalls int
}

func (a *recordingCandidateAPI) GetProfile(context.Context, *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	return &pb.GetProfileResponse{Code: errs.OK}, nil
}

func (a *recordingCandidateAPI) UpdateProfile(context.Context, *pb.UpdateProfileRequest) (*pb.GetProfileResponse, error) {
	return &pb.GetProfileResponse{Code: errs.OK}, nil
}

func (a *recordingCandidateAPI) GetResume(context.Context, *pb.GetResumeRequest) (*pb.GetResumeResponse, error) {
	return &pb.GetResumeResponse{Code: errs.OK}, nil
}

func (a *recordingCandidateAPI) PresignResumeUpload(context.Context, *pb.PresignResumeUploadRequest) (*pb.PresignResumeUploadResponse, error) {
	return &pb.PresignResumeUploadResponse{Code: errs.OK}, nil
}

func (a *recordingCandidateAPI) ConfirmResumeUpload(context.Context, *pb.ConfirmResumeUploadRequest) (*pb.ConfirmResumeUploadResponse, error) {
	a.confirmResumeUploadCalls++
	return &pb.ConfirmResumeUploadResponse{Code: errs.OK}, nil
}

type recordingApplicationAPI struct {
	updateApplicationStatusCalls int
}

func (a *recordingApplicationAPI) ApplyJob(context.Context, *pb.ApplyJobRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (a *recordingApplicationAPI) ListMyApplications(context.Context, *pb.ListMyApplicationsRequest) (*pb.ListMyApplicationsResponse, error) {
	return &pb.ListMyApplicationsResponse{Code: errs.OK}, nil
}

func (a *recordingApplicationAPI) ListJobApplications(context.Context, *pb.ListJobApplicationsRequest) (*pb.ListJobApplicationsResponse, error) {
	return &pb.ListJobApplicationsResponse{Code: errs.OK}, nil
}

func (a *recordingApplicationAPI) UpdateApplicationStatus(context.Context, *pb.UpdateApplicationStatusRequest) (*pb.CommonResponse, error) {
	a.updateApplicationStatusCalls++
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (a *recordingApplicationAPI) ListApplicationStatusTransitions(context.Context, *pb.ListApplicationStatusTransitionsRequest) (*pb.ListApplicationStatusTransitionsResponse, error) {
	return &pb.ListApplicationStatusTransitionsResponse{Code: errs.OK}, nil
}
