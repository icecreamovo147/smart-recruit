package ai

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestGenerateRecruitingReplyWithUsageReturnsProviderUsage(t *testing.T) {
	model := &structuredTestModel{generate: func(context.Context, int) (*schema.Message, error) {
		message := schema.AssistantMessage("usage-aware reply", nil)
		message.ResponseMeta = &schema.ResponseMeta{Usage: &schema.TokenUsage{
			PromptTokens: 88, CompletionTokens: 12, TotalTokens: 100,
		}}
		return message, nil
	}}
	client := newStructuredTestClient(model)
	result, err := client.GenerateRecruitingReplyWithUsage(context.Background(), "question", RecruitingStats{}, nil)
	if err != nil {
		t.Fatalf("GenerateRecruitingReplyWithUsage: %v", err)
	}
	if result.Content != "usage-aware reply" || result.TokenUsage == nil || result.TokenUsage.PromptTokens != 88 || result.TokenUsage.TotalTokens != 100 {
		t.Fatalf("result = %+v", result)
	}
}

func TestGenerateRecruitingReplyCompatibilityWrapper(t *testing.T) {
	model := &structuredTestModel{generate: func(context.Context, int) (*schema.Message, error) {
		return schema.AssistantMessage("legacy reply", nil), nil
	}}
	client := newStructuredTestClient(model)
	reply, err := client.GenerateRecruitingReply(context.Background(), "question", RecruitingStats{}, nil)
	if err != nil || reply != "legacy reply" {
		t.Fatalf("reply = %q, err = %v", reply, err)
	}
}
