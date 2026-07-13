package ai

import (
	"errors"
	"strings"
	"testing"
)

func TestBuildHRFallbackReplyEmpty(t *testing.T) {
	got := BuildHRFallbackReply(nil)
	if !strings.Contains(got, "没有查询到") {
		t.Errorf("expected no-data message, got: %s", got)
	}
}

func TestBuildHRFallbackReplyScalarTotal(t *testing.T) {
	traces := []ToolTrace{{
		ToolName: "query_total_applications",
		Result:   `{"total_applications":42}`,
	}}
	got := BuildHRFallbackReply(traces)
	if !strings.Contains(got, "累计投递总数") || !strings.Contains(got, "42") {
		t.Errorf("expected scalar summary, got: %s", got)
	}
	if !strings.Contains(got, "AI 模型回答失败") {
		t.Error("expected fallback prefix")
	}
}

func TestBuildHRFallbackReplyHotJobs(t *testing.T) {
	traces := []ToolTrace{{
		ToolName: "get_job_heat_ranking",
		Result:   `{"hot_jobs":[{"title":"后端","total":10},{"title":"前端","total":7}]}`,
	}}
	got := BuildHRFallbackReply(traces)
	if !strings.Contains(got, "后端") || !strings.Contains(got, "10") || !strings.Contains(got, "前端") {
		t.Errorf("expected hot jobs, got: %s", got)
	}
}

func TestBuildHRFallbackReplyStatusSummary(t *testing.T) {
	traces := []ToolTrace{{
		ToolName: "get_application_status_summary",
		Result:   `{"job_id":3,"counts":[{"status_text":"待查看","total":5},{"status_text":"通过","total":2}]}`,
	}}
	got := BuildHRFallbackReply(traces)
	if !strings.Contains(got, "待查看") || !strings.Contains(got, "5") || !strings.Contains(got, "通过") {
		t.Errorf("expected status summary, got: %s", got)
	}
	if !strings.Contains(got, "岗位 3") {
		t.Error("expected job-scoped prefix")
	}
}

func TestBuildHRFallbackReplyApplicationList(t *testing.T) {
	traces := []ToolTrace{{
		ToolName: "list_all_applications",
		Result:   `{"total":2,"candidates":[{"real_name":"张三","job_title":"后端","status_text":"通过"},{"real_name":"李四","job_title":"前端","status_text":"淘汰"}]}`,
	}}
	got := BuildHRFallbackReply(traces)
	for _, want := range []string{"张三", "李四", "后端", "前端", "通过", "淘汰"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in: %s", want, got)
		}
	}
}

func TestBuildHRFallbackReplySearchCandidatesEmpty(t *testing.T) {
	traces := []ToolTrace{{
		ToolName: "search_candidates",
		Result:   `{"candidates":[],"message":"未找到匹配的候选人"}`,
	}}
	got := BuildHRFallbackReply(traces)
	if !strings.Contains(got, "未找到匹配的候选人") {
		t.Errorf("expected message passthrough, got: %s", got)
	}
}

func TestBuildHRFallbackReplyProposeAction(t *testing.T) {
	traces := []ToolTrace{{
		ToolName: "propose_application_status_update",
		Result:   `{"action":"update_status","application_id":12,"action_status":2,"candidate_name":"王五","job_title":"算法"}`,
	}}
	got := BuildHRFallbackReply(traces)
	if !strings.Contains(got, "12") || !strings.Contains(got, "王五") || !strings.Contains(got, "通过") {
		t.Errorf("expected propose action summary, got: %s", got)
	}
	if !strings.Contains(got, "二次确认") {
		t.Error("expected confirmation reminder")
	}
}

func TestBuildHRFallbackReplyToolError(t *testing.T) {
	traces := []ToolTrace{{
		ToolName: "query_total_applications",
		Error:    errors.New("db down"),
	}}
	got := BuildHRFallbackReply(traces)
	if !strings.Contains(got, "工具调用失败") || !strings.Contains(got, "db down") {
		t.Errorf("expected error trace summary, got: %s", got)
	}
}

func TestBuildHRFallbackReplyMultipleTools(t *testing.T) {
	traces := []ToolTrace{
		{ToolName: "query_total_applications", Result: `{"total_applications":100}`},
		{ToolName: "query_today_applications", Result: `{"today_applications":7}`},
	}
	got := BuildHRFallbackReply(traces)
	if !strings.Contains(got, "100") || !strings.Contains(got, "7") {
		t.Errorf("expected both scalars, got: %s", got)
	}
}

func TestBuildCandidateFallbackReplyEmpty(t *testing.T) {
	got := BuildCandidateFallbackReply(nil)
	if !strings.Contains(got, "没有查询到") {
		t.Errorf("expected no-data message, got: %s", got)
	}
}

func TestBuildCandidateFallbackReplyMyApplications(t *testing.T) {
	traces := []ToolTrace{{
		ToolName: "list_my_applications",
		Result:   `{"applications":[{"job_title":"后端","status_text":"通过","round_no":2,"applied_at":"2026-05-20"}]}`,
	}}
	got := BuildCandidateFallbackReply(traces)
	for _, want := range []string{"后端", "通过", "2026-05-20", "第 2 轮"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in: %s", want, got)
		}
	}
}

func TestBuildCandidateFallbackReplyResumeUnavailable(t *testing.T) {
	traces := []ToolTrace{{
		ToolName: "get_my_resume_text",
		Result:   `{"resume_available":false,"message":"请先上传简历"}`,
	}}
	got := BuildCandidateFallbackReply(traces)
	if !strings.Contains(got, "请先上传简历") {
		t.Errorf("expected resume message, got: %s", got)
	}
}

func TestBuildCandidateFallbackReplyResumeAvailable(t *testing.T) {
	traces := []ToolTrace{{
		ToolName: "get_my_resume_text",
		Result:   `{"resume_available":true,"file_name":"resume.pdf"}`,
	}}
	got := BuildCandidateFallbackReply(traces)
	if !strings.Contains(got, "resume.pdf") {
		t.Errorf("expected file name, got: %s", got)
	}
}

func TestBuildCandidateFallbackReplyRecommend(t *testing.T) {
	traces := []ToolTrace{{
		ToolName: "recommend_jobs_by_resume",
		Result:   `{"jobs":[]}`,
	}}
	got := BuildCandidateFallbackReply(traces)
	if !strings.Contains(got, "AI 推荐生成失败") {
		t.Errorf("expected recommend fallback note, got: %s", got)
	}
}

func TestFormatJobListTruncates(t *testing.T) {
	// Construct 7 jobs to verify the truncation note is shown.
	raw := `{"jobs":[
		{"title":"A","department":"D"},
		{"title":"B","department":"D"},
		{"title":"C","department":"D"},
		{"title":"D","department":"D"},
		{"title":"E","department":"D"},
		{"title":"F","department":"D"},
		{"title":"G","department":"D"}
	]}`
	got := formatJobList(raw, "jobs")
	if !strings.Contains(got, "共 7 条") {
		t.Errorf("expected total count note, got: %s", got)
	}
}
