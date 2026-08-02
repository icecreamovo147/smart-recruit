package hr_tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

type fakeJobClient struct {
	listResp   *pb.ListJobsResponse
	detailResp *pb.GetJobDetailResponse
	listCalls  int
	detailID   int64
}

func (f *fakeJobClient) ListHRJobs(context.Context, *pb.ListHRJobsRequest) (*pb.ListJobsResponse, error) {
	f.listCalls++
	if f.listResp != nil {
		return f.listResp, nil
	}
	return &pb.ListJobsResponse{Code: errs.OK}, nil
}

func (f *fakeJobClient) GetJobDetail(_ context.Context, req *pb.GetJobDetailRequest) (*pb.GetJobDetailResponse, error) {
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
	assertToolErrorKind(t, err, "forbidden_or_not_found")
	if !jsonContains(result.Content, "无权限") && !jsonContains(result.Content, "error") {
		t.Fatalf("unowned result = %s", result.Content)
	}
}

func TestResolveBuiltinToolNamesFailsClosedAndNormalizes(t *testing.T) {
	got := ResolveBuiltinToolNames(nil, nil)
	if len(got) != 0 {
		t.Fatalf("empty bindings resolved = %v, want none", got)
	}
	got = ResolveBuiltinToolNames([]string{"builtin:get_job_list", "builtin:disabled_unknown"}, []string{"candidate_search"})
	if len(got) != 1 || got[0] != "get_job_list" {
		t.Fatalf("resolved = %v", got)
	}
	for name, input := range map[string][]string{
		"abstract":         {"candidate_search", "resume_intelligence"},
		"unknown":          {"unknown_tool"},
		"platform_context": {"get_application_snapshot"},
	} {
		if got := ResolveBuiltinToolNames(nil, input); len(got) != 0 {
			t.Fatalf("%s bindings resolved = %v, want none", name, got)
		}
	}
}

func TestExecuteReturnsClassifiedErrors(t *testing.T) {
	exec := &Executor{}
	result, err := exec.Execute(context.Background(), 0, "get_job_list", nil)
	assertToolErrorKind(t, err, "invalid_argument")
	if !jsonContains(result.Content, "error_type") {
		t.Fatalf("invalid argument result = %s", result.Content)
	}

	_, err = exec.Execute(context.Background(), 77, "does_not_exist", nil)
	assertToolErrorKind(t, err, "unsupported")

	_, err = exec.Execute(context.Background(), 77, "get_job_list", nil)
	assertToolErrorKind(t, err, "downstream")
}

func assertToolErrorKind(t *testing.T, err error, want string) {
	t.Helper()
	var toolErr *ToolExecutionError
	if !errors.As(err, &toolErr) {
		t.Fatalf("error = %v, want ToolExecutionError", err)
	}
	if toolErr.Kind != want {
		t.Fatalf("error kind = %q, want %q", toolErr.Kind, want)
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
