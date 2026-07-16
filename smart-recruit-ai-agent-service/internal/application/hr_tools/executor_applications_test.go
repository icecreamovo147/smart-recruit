package hr_tools

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	gogrpc "google.golang.org/grpc"

	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

type fakeAppListClient struct {
	byJob map[int64][]*pb.JobApplication
}

func (f *fakeAppListClient) ListJobApplications(_ context.Context, req *pb.ListJobApplicationsRequest, _ ...gogrpc.CallOption) (*pb.ListJobApplicationsResponse, error) {
	list := f.byJob[req.GetJobId()]
	return &pb.ListJobApplicationsResponse{Code: errs.OK, Total: int64(len(list)), List: list}, nil
}

type fakeSnapshotClient struct {
	resp *pb.GetApplicationSnapshotResponse
}

func (f *fakeSnapshotClient) GetApplicationSnapshot(context.Context, *pb.GetApplicationSnapshotRequest, ...gogrpc.CallOption) (*pb.GetApplicationSnapshotResponse, error) {
	if f.resp != nil {
		return f.resp, nil
	}
	return &pb.GetApplicationSnapshotResponse{Code: 404, Msg: "not found"}, nil
}

func TestSearchCandidatesAndStatusSummary(t *testing.T) {
	jobs := &fakeJobClient{listResp: &pb.ListJobsResponse{
		Code: errs.OK,
		List: []*pb.Job{
			{JobId: 10, Title: "Backend", Status: 1, ApplicationCount: 2},
			{JobId: 11, Title: "Frontend", Status: 1, ApplicationCount: 1},
		},
	}}
	apps := &fakeAppListClient{byJob: map[int64][]*pb.JobApplication{
		10: {
			{ApplicationId: 1001, RealName: "Ada Lovelace", Phone: "13800138000", Status: 0, IsCurrent: 1, AppliedAt: time.Now().Format("2006-01-02 15:04")},
			{ApplicationId: 1002, RealName: "Bob", Phone: "13900139000", Status: 2, IsCurrent: 1, AppliedAt: time.Now().AddDate(0, 0, -2).Format("2006-01-02 15:04")},
		},
		11: {
			{ApplicationId: 1101, RealName: "Carol", Phone: "13700137000", Status: 1, IsCurrent: 1, AppliedAt: time.Now().Format("2006-01-02 15:04")},
		},
	}}
	exec := &Executor{Jobs: jobs, Applications: apps}

	search, err := exec.Execute(context.Background(), 77, "search_candidates", map[string]any{"keyword": "Ada"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !jsonContains(search.Content, "Ada Lovelace") || !jsonContains(search.Content, "Backend") {
		t.Fatalf("search result = %s", search.Content)
	}

	summary, err := exec.Execute(context.Background(), 77, "get_application_status_summary", nil)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	var payload struct {
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal([]byte(summary.Content), &payload); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}
	if payload.Total != 3 {
		t.Fatalf("total = %d, want 3", payload.Total)
	}

	today, err := exec.Execute(context.Background(), 77, "query_today_applications", nil)
	if err != nil {
		t.Fatalf("today: %v", err)
	}
	if !jsonContains(today.Content, "today_applications") {
		t.Fatalf("today = %s", today.Content)
	}
}

func TestProposeStatusUpdateDoesNotMutate(t *testing.T) {
	jobs := &fakeJobClient{listResp: &pb.ListJobsResponse{
		Code: errs.OK,
		List: []*pb.Job{{JobId: 10, Title: "Backend", Status: 1}},
	}}
	apps := &fakeAppListClient{byJob: map[int64][]*pb.JobApplication{
		10: {{ApplicationId: 1001, RealName: "Ada", Status: 0, IsCurrent: 1}},
	}}
	snaps := &fakeSnapshotClient{resp: &pb.GetApplicationSnapshotResponse{
		Code: errs.OK, ApplicationId: 1001, CandidateName: "Ada", JobTitle: "Backend", JobId: 10, JobHrId: 77,
	}}
	exec := &Executor{Jobs: jobs, Applications: apps, Snapshots: snaps}
	result, err := exec.Execute(context.Background(), 77, "propose_application_status_update", map[string]any{
		"application_id": 1001,
		"status":         2,
	})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if !jsonContains(result.Content, "requires_confirm") || !jsonContains(result.Content, "Ada") {
		t.Fatalf("result = %s", result.Content)
	}
}
