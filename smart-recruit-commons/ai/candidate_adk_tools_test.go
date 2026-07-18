package ai

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/eino/components/tool"
)

type stubCandidateRunner struct {
	lastOwner int64
	lastTool  string
	content   string
	err       error
}

func (s *stubCandidateRunner) Execute(_ context.Context, ownerID int64, toolName string, _ map[string]any) (ToolResult, error) {
	s.lastOwner = ownerID
	s.lastTool = toolName
	if s.err != nil {
		return ToolResult{}, s.err
	}
	if s.content != "" {
		return ToolResult{Content: s.content}, nil
	}
	return ToolResult{Content: `{}`}, nil
}

func TestCandidateToolCreation_NilExecutor(t *testing.T) {
	_, err := NewCandidateADKTools(nil)
	if err == nil {
		t.Fatal("expected error for nil executor")
	}
}

func TestCandidateToolCount(t *testing.T) {
	tools, err := NewCandidateADKTools(&stubCandidateRunner{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 6 {
		t.Errorf("expected 6 candidate tools, got %d", len(tools))
	}
	for _, bt := range tools {
		if _, ok := bt.(tool.InvokableTool); !ok {
			t.Errorf("tool %T does not implement InvokableTool", bt)
		}
	}
}

func TestCandidateADKToolsInvokeWithOwnerContext(t *testing.T) {
	runner := &stubCandidateRunner{content: `{"total":0,"applications":[]}`}
	tools, err := NewCandidateADKTools(runner)
	if err != nil {
		t.Fatal(err)
	}
	var listTool tool.InvokableTool
	for _, bt := range tools {
		info, infoErr := bt.Info(context.Background())
		if infoErr != nil {
			t.Fatal(infoErr)
		}
		if info.Name == "list_my_applications" {
			var ok bool
			listTool, ok = bt.(tool.InvokableTool)
			if !ok {
				t.Fatal("list_my_applications is not InvokableTool")
			}
			break
		}
	}
	if listTool == nil {
		t.Fatal("list_my_applications tool not found")
	}
	ctx := WithOwnerID(context.Background(), 55)
	if _, err := listTool.InvokableRun(ctx, `{}`); err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if runner.lastOwner != 55 || runner.lastTool != "list_my_applications" {
		t.Fatalf("executor called with owner=%d tool=%q", runner.lastOwner, runner.lastTool)
	}
}

func TestFilterCandidateADKToolsByName_EmptyAllowlistKeepsAll(t *testing.T) {
	tools, err := NewCandidateADKTools(&stubCandidateRunner{})
	if err != nil {
		t.Fatal(err)
	}
	filtered := FilterCandidateADKToolsByName(tools, nil)
	if len(filtered) != len(tools) {
		t.Fatalf("empty allowlist should keep all tools, got %d want %d", len(filtered), len(tools))
	}
	filtered = FilterCandidateADKToolsByName(tools, []string{"list_my_applications"})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(filtered))
	}
}

func TestListMyApplicationsOutput_Serialize(t *testing.T) {
	output := listMyApplicationsOutput{
		Total: 2,
		Applications: []myAppEntry{
			{ApplicationID: 1, JobTitle: "前端开发", StatusText: "已查看"},
		},
	}
	data, _ := json.Marshal(output)
	if !json.Valid(data) {
		t.Errorf("output is not valid JSON: %s", string(data))
	}
}

func TestGetMyResumeOutput_Serialize(t *testing.T) {
	output := getMyResumeOutput{
		ResumeAvailable: true,
		FileName:        "resume.pdf",
		TextLength:      100,
		ResumeText:      "test content",
	}
	data, err := json.Marshal(output)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) {
		t.Errorf("output is not valid JSON: %s", string(data))
	}
}

func TestRecommendJobsByResumeOutput_Serialize(t *testing.T) {
	output := recommendJobsByResumeOutput{
		ResumeText:  "resume content",
		ResumeFile:  "resume.pdf",
		TotalJobs:   3,
		Jobs:        []candidateJobEntry{{JobID: 1, Title: "后端", HasApplied: false}},
		Instruction: "请推荐岗位",
	}
	data, _ := json.Marshal(output)
	if !json.Valid(data) {
		t.Errorf("output is not valid JSON: %s", string(data))
	}
}

func TestMyApplicationDetailOutput_RequiredFields(t *testing.T) {
	output := myApplicationDetailOutput{
		ApplicationID: 1,
		JobID:         2,
		JobTitle:      "测试岗位",
		Status:        1,
		StatusText:    "已查看",
		Department:    "技术部",
	}
	data, _ := json.Marshal(output)
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	required := []string{"application_id", "job_id", "job_title", "status", "status_text", "department"}
	for _, key := range required {
		if _, ok := m[key]; !ok {
			t.Errorf("required field %q missing from output", key)
		}
	}
}

func TestJobDetailForCandidateOutput_Serialize(t *testing.T) {
	output := jobDetailForCandidateOutput{
		JobID:       1,
		Title:       "测试",
		Department:  "技术",
		Status:      1,
		StatusText:  "招募中",
		HasApplied:  true,
		Description: "岗位描述",
	}
	data, _ := json.Marshal(output)
	if !json.Valid(data) {
		t.Errorf("output is not valid JSON: %s", string(data))
	}
}
