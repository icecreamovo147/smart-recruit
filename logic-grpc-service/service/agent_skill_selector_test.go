package service

import (
	"context"
	"strings"
	"testing"

	"logic-grpc-service/repository"
)

type fakeAgentSkillLister struct {
	rows []repository.AgentSkillRuntimeRecord
}

func (f fakeAgentSkillLister) ListEnabled(context.Context) ([]repository.AgentSkillRuntimeRecord, error) {
	return f.rows, nil
}

func TestSelectAgentSkillsManualPriorityAndAutoMatch(t *testing.T) {
	repo := fakeAgentSkillLister{rows: []repository.AgentSkillRuntimeRecord{
		{ID: 1, Name: "screening", DisplayName: "Screening", Description: "简历筛选", BodyMarkdown: "筛选候选人", IsManualInvocable: 1},
		{ID: 2, Name: "offer", DisplayName: "Offer", Description: "offer negotiation", BodyMarkdown: "薪资 沟通 offer", IsManualInvocable: 1},
		{ID: 3, Name: "interview", DisplayName: "Interview", Description: "面试安排", BodyMarkdown: "安排面试", IsManualInvocable: 1},
		{ID: 4, Name: "analytics", DisplayName: "Analytics", Description: "招聘趋势", BodyMarkdown: "数据分析", IsManualInvocable: 1},
	}}

	selected, err := selectAgentSkills(context.Background(), repo, "帮我安排面试并准备 offer 沟通", []int64{3})
	if err != nil {
		t.Fatalf("selectAgentSkills returned error: %v", err)
	}
	if len(selected) != 2 {
		t.Fatalf("len(selected) = %d, want 2", len(selected))
	}
	if selected[0].ID != 3 || !selected[0].Manual {
		t.Fatalf("first selected = %+v, want manual skill 3 first", selected[0])
	}
	if selected[1].ID != 2 || selected[1].Manual {
		t.Fatalf("second selected = %+v, want auto skill 2", selected[1])
	}
}

func TestSelectAgentSkillsSkipsNonManualSkillForManualIDs(t *testing.T) {
	repo := fakeAgentSkillLister{rows: []repository.AgentSkillRuntimeRecord{
		{ID: 7, Name: "internal", DisplayName: "Internal", Description: "内部流程", BodyMarkdown: "内部流程", IsManualInvocable: 0},
	}}

	selected, err := selectAgentSkills(context.Background(), repo, "内部流程", []int64{7})
	if err != nil {
		t.Fatalf("selectAgentSkills returned error: %v", err)
	}
	if len(selected) != 1 || selected[0].Manual {
		t.Fatalf("selected = %+v, want only auto-selected non-manual skill", selected)
	}
}

func TestRenderAgentSkillInstructionBlock(t *testing.T) {
	block := renderAgentSkillInstructionBlock([]selectedAgentSkill{{
		ID:          9,
		DisplayName: "Follow-up",
		Content:     "Use concise next steps.",
	}})
	if block == "" || !strings.Contains(block, "Agent Skill instructions") || !strings.Contains(block, "Use concise next steps.") {
		t.Fatalf("unexpected block: %q", block)
	}
}
