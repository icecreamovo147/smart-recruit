package ai

import (
	"encoding/json"
	"testing"

	"github.com/cloudwego/eino/components/tool"
)

func TestQueryTodayInput_JSONTags(t *testing.T) {
	input := queryTodayInput{JobID: 42}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var got queryTodayInput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.JobID != 42 {
		t.Errorf("expected JobID=42, got %d", got.JobID)
	}
}

func TestProposeStatusUpdateInput_JSONTags(t *testing.T) {
	input := proposeStatusUpdateInput{ApplicationID: 1, Status: 2}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var got proposeStatusUpdateInput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.ApplicationID != 1 || got.Status != 2 {
		t.Errorf("got %+v", got)
	}
}

func TestSearchCandidatesOutput_Serialize(t *testing.T) {
	output := searchCandidatesOutput{
		Candidates: []candidateEntry{
			{ApplicationID: 1, RealName: "张三", JobTitle: "后端开发", Status: "待查看"},
		},
	}
	data, err := json.Marshal(output)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	candidates, ok := m["candidates"].([]any)
	if !ok || len(candidates) != 1 {
		t.Errorf("expected 1 candidate in output, got %v", m)
	}
}

func TestGetJobListOutput_Serialize(t *testing.T) {
	output := getJobListOutput{
		Jobs: []jobEntry{
			{JobID: 1, Title: "后端开发", StatusText: "招募中"},
		},
	}
	data, _ := json.Marshal(output)
	if !json.Valid(data) {
		t.Errorf("output is not valid JSON: %s", string(data))
	}
}

func TestToolCreation_NilExecutor(t *testing.T) {
	_, err := NewRecruitingADKTools(nil)
	if err == nil {
		t.Fatal("expected error for nil executor")
	}
}

func TestInferToolType(t *testing.T) {
	executor := &ToolExecutor{}
	tools, err := NewRecruitingADKTools(executor)
	if err != nil {
		t.Fatalf("NewRecruitingADKTools: %v", err)
	}
	if len(tools) != 14 {
		t.Errorf("expected 14 tools, got %d", len(tools))
	}
	for _, bt := range tools {
		if _, ok := bt.(tool.InvokableTool); !ok {
			t.Errorf("tool %T does not implement InvokableTool", bt)
		}
	}
}
