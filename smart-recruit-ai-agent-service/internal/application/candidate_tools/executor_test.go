package candidate_tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type fakeCandidateStore struct {
	apps     []ApplicationListItem
	detail   ApplicationDetail
	foundApp bool
	resume   ResumeText
	jobs     []JobListItem
	job      JobDetail
	foundJob bool
}

func (f *fakeCandidateStore) ListMyApplicationsForAI(context.Context, int64, int32) ([]ApplicationListItem, error) {
	return f.apps, nil
}
func (f *fakeCandidateStore) GetMyApplicationDetailForAI(_ context.Context, _, applicationID int64) (ApplicationDetail, bool, error) {
	if !f.foundApp || f.detail.ApplicationID != applicationID {
		return ApplicationDetail{}, false, nil
	}
	return f.detail, true, nil
}
func (f *fakeCandidateStore) GetMyResumeTextForAI(context.Context, int64) (ResumeText, error) {
	return f.resume, nil
}
func (f *fakeCandidateStore) ListJobsForCandidateAI(context.Context, int64, int32) ([]JobListItem, error) {
	return f.jobs, nil
}
func (f *fakeCandidateStore) GetJobDetailForCandidateAI(context.Context, int64, int64) (JobDetail, bool, error) {
	return f.job, f.foundJob, nil
}

func TestExecutorListMyApplicationsEmpty(t *testing.T) {
	exec := NewExecutor(&fakeCandidateStore{})
	result, err := exec.Execute(context.Background(), 55, "list_my_applications", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Content, "还没有投递记录") {
		t.Fatalf("content = %s", result.Content)
	}
}

func TestExecutorGetMyApplicationDetailOwnership(t *testing.T) {
	exec := NewExecutor(&fakeCandidateStore{foundApp: false})
	result, err := exec.Execute(context.Background(), 55, "get_my_application_detail", map[string]any{"application_id": 99})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Content, "不存在或无权限") {
		t.Fatalf("content = %s", result.Content)
	}
}

func TestExecutorGetMyApplicationDetailSuccess(t *testing.T) {
	exec := NewExecutor(&fakeCandidateStore{
		foundApp: true,
		detail:   ApplicationDetail{ApplicationID: 99, JobTitle: "后端", StatusText: "待查看"},
	})
	result, err := exec.Execute(context.Background(), 55, "get_my_application_detail", map[string]any{"application_id": float64(99)})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(result.Content), &m); err != nil {
		t.Fatal(err)
	}
	if m["job_title"] != "后端" {
		t.Fatalf("got %#v", m)
	}
}

func TestExecutorRecommendJobsRequiresResume(t *testing.T) {
	exec := NewExecutor(&fakeCandidateStore{
		resume: ResumeText{Available: false, Message: "no resume"},
	})
	result, err := exec.Execute(context.Background(), 55, "recommend_jobs_by_resume", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Content, "no_resume") {
		t.Fatalf("content = %s", result.Content)
	}
}

func TestExecutorRecommendJobsSuccess(t *testing.T) {
	exec := NewExecutor(&fakeCandidateStore{
		resume: ResumeText{
			Available:  true,
			FileName:   "r.pdf",
			TextLength: 100,
			ResumeText: strings.Repeat("Go backend engineer experience ", 5),
		},
		jobs: []JobListItem{{JobID: 1, Title: "后端开发", HasApplied: false}},
	})
	result, err := exec.Execute(context.Background(), 55, "recommend_jobs_by_resume", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Content, "instruction") || !strings.Contains(result.Content, "后端开发") {
		t.Fatalf("content = %s", result.Content)
	}
}

func TestExecutorRequiresUserID(t *testing.T) {
	exec := NewExecutor(&fakeCandidateStore{})
	if _, err := exec.Execute(context.Background(), 0, "list_my_applications", nil); err == nil {
		t.Fatal("expected error for missing user id")
	}
}

func TestExecutorUnknownTool(t *testing.T) {
	exec := NewExecutor(&fakeCandidateStore{})
	if _, err := exec.Execute(context.Background(), 1, "not_a_tool", nil); err == nil {
		t.Fatal("expected unknown tool error")
	}
}
