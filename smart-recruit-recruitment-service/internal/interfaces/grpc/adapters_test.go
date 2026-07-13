package grpc

import (
	"context"
	"testing"

	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

func TestAdaptersDelegateRecruitmentRuntimeAPIs(t *testing.T) {
	ctx := context.Background()
	delegate := &fakeRecruitmentDelegate{}
	job, err := NewJobAdapter(delegate)
	if err != nil {
		t.Fatalf("NewJobAdapter() error = %v", err)
	}
	if resp, err := job.CreateJob(ctx, &pb.CreateJobRequest{}); err != nil || resp.Code != errs.OK {
		t.Fatalf("CreateJob() = %+v, %v", resp, err)
	}
	candidate, err := NewCandidateAdapter(delegate)
	if err != nil {
		t.Fatalf("NewCandidateAdapter() error = %v", err)
	}
	if resp, err := candidate.UpdateProfile(ctx, &pb.UpdateProfileRequest{}); err != nil || resp.Code != errs.OK {
		t.Fatalf("UpdateProfile() = %+v, %v", resp, err)
	}
	application, err := NewApplicationAdapter(delegate)
	if err != nil {
		t.Fatalf("NewApplicationAdapter() error = %v", err)
	}
	if resp, err := application.UpdateApplicationStatus(ctx, &pb.UpdateApplicationStatusRequest{}); err != nil || resp.Code != errs.OK {
		t.Fatalf("UpdateApplicationStatus() = %+v, %v", resp, err)
	}
	if delegate.calls["CreateJob"] != 1 || delegate.calls["UpdateProfile"] != 1 || delegate.calls["UpdateApplicationStatus"] != 1 {
		t.Fatalf("delegate calls = %+v", delegate.calls)
	}
}

func TestAdaptersRejectMissingDelegates(t *testing.T) {
	if _, err := NewJobAdapter(nil); err == nil {
		t.Fatal("NewJobAdapter(nil) expected error")
	}
	if _, err := NewCollaborationAdapter(nil); err == nil {
		t.Fatal("NewCollaborationAdapter(nil) expected error")
	}
}

type fakeRecruitmentDelegate struct {
	calls map[string]int
}

func (d *fakeRecruitmentDelegate) mark(name string) {
	if d.calls == nil {
		d.calls = map[string]int{}
	}
	d.calls[name]++
}

func (d *fakeRecruitmentDelegate) CreateJob(context.Context, *pb.CreateJobRequest) (*pb.CreateJobResponse, error) {
	d.mark("CreateJob")
	return &pb.CreateJobResponse{Code: errs.OK}, nil
}
func (d *fakeRecruitmentDelegate) UpdateJob(context.Context, *pb.UpdateJobRequest) (*pb.CommonResponse, error) {
	d.mark("UpdateJob")
	return &pb.CommonResponse{Code: errs.OK}, nil
}
func (d *fakeRecruitmentDelegate) OfflineJob(context.Context, *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	d.mark("OfflineJob")
	return &pb.CommonResponse{Code: errs.OK}, nil
}
func (d *fakeRecruitmentDelegate) OnlineJob(context.Context, *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	d.mark("OnlineJob")
	return &pb.CommonResponse{Code: errs.OK}, nil
}
func (d *fakeRecruitmentDelegate) ListHRJobs(context.Context, *pb.ListHRJobsRequest) (*pb.ListJobsResponse, error) {
	d.mark("ListHRJobs")
	return &pb.ListJobsResponse{Code: errs.OK}, nil
}
func (d *fakeRecruitmentDelegate) ListPublicJobs(context.Context, *pb.ListPublicJobsRequest) (*pb.ListJobsResponse, error) {
	d.mark("ListPublicJobs")
	return &pb.ListJobsResponse{Code: errs.OK}, nil
}
func (d *fakeRecruitmentDelegate) GetJobDetail(context.Context, *pb.GetJobDetailRequest) (*pb.GetJobDetailResponse, error) {
	d.mark("GetJobDetail")
	return &pb.GetJobDetailResponse{Code: errs.OK}, nil
}
func (d *fakeRecruitmentDelegate) GetProfile(context.Context, *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	d.mark("GetProfile")
	return &pb.GetProfileResponse{Code: errs.OK}, nil
}
func (d *fakeRecruitmentDelegate) UpdateProfile(context.Context, *pb.UpdateProfileRequest) (*pb.GetProfileResponse, error) {
	d.mark("UpdateProfile")
	return &pb.GetProfileResponse{Code: errs.OK}, nil
}
func (d *fakeRecruitmentDelegate) GetResume(context.Context, *pb.GetResumeRequest) (*pb.GetResumeResponse, error) {
	d.mark("GetResume")
	return &pb.GetResumeResponse{Code: errs.OK}, nil
}
func (d *fakeRecruitmentDelegate) PresignResumeUpload(context.Context, *pb.PresignResumeUploadRequest) (*pb.PresignResumeUploadResponse, error) {
	d.mark("PresignResumeUpload")
	return &pb.PresignResumeUploadResponse{Code: errs.OK}, nil
}
func (d *fakeRecruitmentDelegate) ConfirmResumeUpload(context.Context, *pb.ConfirmResumeUploadRequest) (*pb.ConfirmResumeUploadResponse, error) {
	d.mark("ConfirmResumeUpload")
	return &pb.ConfirmResumeUploadResponse{Code: errs.OK}, nil
}
func (d *fakeRecruitmentDelegate) ApplyJob(context.Context, *pb.ApplyJobRequest) (*pb.CommonResponse, error) {
	d.mark("ApplyJob")
	return &pb.CommonResponse{Code: errs.OK}, nil
}
func (d *fakeRecruitmentDelegate) ListMyApplications(context.Context, *pb.ListMyApplicationsRequest) (*pb.ListMyApplicationsResponse, error) {
	d.mark("ListMyApplications")
	return &pb.ListMyApplicationsResponse{Code: errs.OK}, nil
}
func (d *fakeRecruitmentDelegate) ListJobApplications(context.Context, *pb.ListJobApplicationsRequest) (*pb.ListJobApplicationsResponse, error) {
	d.mark("ListJobApplications")
	return &pb.ListJobApplicationsResponse{Code: errs.OK}, nil
}
func (d *fakeRecruitmentDelegate) UpdateApplicationStatus(context.Context, *pb.UpdateApplicationStatusRequest) (*pb.CommonResponse, error) {
	d.mark("UpdateApplicationStatus")
	return &pb.CommonResponse{Code: errs.OK}, nil
}
func (d *fakeRecruitmentDelegate) ListApplicationStatusTransitions(context.Context, *pb.ListApplicationStatusTransitionsRequest) (*pb.ListApplicationStatusTransitionsResponse, error) {
	d.mark("ListApplicationStatusTransitions")
	return &pb.ListApplicationStatusTransitionsResponse{Code: errs.OK}, nil
}
