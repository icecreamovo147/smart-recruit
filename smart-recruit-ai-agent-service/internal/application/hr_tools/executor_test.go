package hr_tools

import (
	"context"
	"encoding/json"
	"testing"

	gogrpc "google.golang.org/grpc"

	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

type fakeJobClient struct {
	listResp   *pb.ListJobsResponse
	detailResp *pb.GetJobDetailResponse
	listCalls  int
	detailID   int64
}

func (f *fakeJobClient) ListHRJobs(context.Context, *pb.ListHRJobsRequest, ...gogrpc.CallOption) (*pb.ListJobsResponse, error) {
	f.listCalls++
	if f.listResp != nil {
		return f.listResp, nil
	}
	return &pb.ListJobsResponse{Code: errs.OK}, nil
}

func (f *fakeJobClient) GetJobDetail(_ context.Context, req *pb.GetJobDetailRequest, _ ...gogrpc.CallOption) (*pb.GetJobDetailResponse, error) {
	f.detailID = req.GetJobId()
	if f.detailResp != nil {
		return f.detailResp, nil
	}
	return &pb.GetJobDetailResponse{Code: 404, Msg: "not found"}, nil
}

func TestGetJobListReturnsOpenJobsOnly(t *testing.T) {
	client := &fakeJobClient{listResp: &pb.ListJobsResponse{
		Code:  errs.OK,
		Total: 3,
		List: []*pb.Job{
			{JobId: 1, Title: "Backend", Status: 1, Department: "Eng"},
			{JobId: 2, Title: "Frontend", Status: 1, Location: "Shanghai"},
			{JobId: 3, Title: "Closed", Status: 0},
		},
	}}
	exec := &Executor{Jobs: client}
	result, err := exec.Execute(context.Background(), 77, "get_job_list", nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var payload struct {
		Total int              `json:"total"`
		Jobs  []map[string]any `json:"jobs"`
	}
	if err := json.Unmarshal([]byte(result.Content), &payload); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, result.Content)
	}
	if payload.Total != 2 || len(payload.Jobs) != 2 {
		t.Fatalf("payload = %#v, want 2 open jobs", payload)
	}
}

func TestSearchJobsFiltersByKeyword(t *testing.T) {
	client := &fakeJobClient{listResp: &pb.ListJobsResponse{
		Code: errs.OK,
		List: []*pb.Job{
			{JobId: 1, Title: "Go Backend", Status: 1},
			{JobId: 2, Title: "Java Backend", Status: 1},
			{JobId: 3, Title: "Product Manager", Status: 1},
		},
	}}
	exec := &Executor{Jobs: client}
	result, err := exec.Execute(context.Background(), 77, "search_jobs", map[string]any{"keyword": "backend"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var payload struct {
		Total int `json:"total"`
		Jobs  []struct {
			Title string `json:"title"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal([]byte(result.Content), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.Total != 2 {
		t.Fatalf("total = %d, want 2", payload.Total)
	}
}

func TestGetJobDetailRequiresOwnership(t *testing.T) {
	client := &fakeJobClient{
		listResp: &pb.ListJobsResponse{
			Code: errs.OK,
			List: []*pb.Job{{JobId: 9, Title: "Owned", Status: 1, HrId: 77}},
		},
		detailResp: &pb.GetJobDetailResponse{
			Code: errs.OK,
			Job:  &pb.Job{JobId: 9, Title: "Owned", Status: 1, Description: "desc", Requirements: "req"},
		},
	}
	exec := &Executor{Jobs: client}
	result, err := exec.Execute(context.Background(), 77, "get_job_detail", map[string]any{"job_id": 9})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !jsonContains(result.Content, "Owned") {
		t.Fatalf("result = %s", result.Content)
	}
	// Not owned
	result, err = exec.Execute(context.Background(), 77, "get_job_detail", map[string]any{"job_id": 99})
	if err != nil {
		t.Fatalf("Execute unowned: %v", err)
	}
	if !jsonContains(result.Content, "无权限") && !jsonContains(result.Content, "error") {
		t.Fatalf("unowned result = %s", result.Content)
	}
}

func TestResolveBuiltinToolNamesDefaultsAndNormalizes(t *testing.T) {
	got := ResolveBuiltinToolNames(nil, nil)
	if len(got) != len(DefaultJobToolNames) {
		t.Fatalf("default tools = %v, want %v", got, DefaultJobToolNames)
	}
	got = ResolveBuiltinToolNames([]string{"builtin:get_job_list"}, []string{"candidate_search"})
	if len(got) != 1 || got[0] != "get_job_list" {
		t.Fatalf("resolved = %v", got)
	}
	// Only abstract caps → fall back to default job tools
	got = ResolveBuiltinToolNames(nil, []string{"candidate_search", "resume_intelligence"})
	if len(got) != len(DefaultJobToolNames) {
		t.Fatalf("abstract fallback = %v", got)
	}
}

func jsonContains(raw, needle string) bool {
	return len(raw) > 0 && (needle == "" || contains(raw, needle))
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || len(s) > 0 && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()))
}
