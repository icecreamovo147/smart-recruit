package ai

import (
	"encoding/json"
	"testing"

	"github.com/cloudwego/eino/components/tool"
)

func TestCandidateToolCreation_NilExecutor(t *testing.T) {
	_, err := NewCandidateADKTools(nil)
	if err == nil {
		t.Fatal("expected error for nil executor")
	}
}

func TestCandidateToolCount(t *testing.T) {
	// Pass a non-nil executor for tool creation validation only;
	// tool closures delegate to Execute() which won't be called in tests.
	executor := &CandidateToolExecutor{}
	tools, err := NewCandidateADKTools(executor)
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
