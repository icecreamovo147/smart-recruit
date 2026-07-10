package ai

import (
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestToolMetadataBillingIsCumulativeAcrossTwoCalls(t *testing.T) {
	var m ToolMetadata

	m.recordModelUsage(&schema.TokenUsage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120})
	m.recordModelUsage(&schema.TokenUsage{PromptTokens: 200, CompletionTokens: 40, TotalTokens: 240})

	if m.BillingTokenUsage == nil {
		t.Fatal("BillingTokenUsage should not be nil after two calls")
	}
	if m.BillingTokenUsage.PromptTokens != 300 {
		t.Errorf("Billing cumulative prompt = %d, want 300", m.BillingTokenUsage.PromptTokens)
	}
	if m.BillingTokenUsage.CompletionTokens != 60 {
		t.Errorf("Billing cumulative completion = %d, want 60", m.BillingTokenUsage.CompletionTokens)
	}
	if m.BillingTokenUsage.TotalTokens != 360 {
		t.Errorf("Billing cumulative total = %d, want 360", m.BillingTokenUsage.TotalTokens)
	}
}

func TestToolMetadataContextIsLatestCall(t *testing.T) {
	var m ToolMetadata

	m.recordModelUsage(&schema.TokenUsage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120})
	m.recordModelUsage(&schema.TokenUsage{PromptTokens: 200, CompletionTokens: 40, TotalTokens: 240})

	if m.ContextTokenUsage == nil {
		t.Fatal("ContextTokenUsage should not be nil")
	}
	if m.ContextTokenUsage.PromptTokens != 200 {
		t.Errorf("Context prompt = %d, want 200 (latest call)", m.ContextTokenUsage.PromptTokens)
	}
	if m.ContextTokenUsage.CompletionTokens != 40 {
		t.Errorf("Context completion = %d, want 40 (latest call)", m.ContextTokenUsage.CompletionTokens)
	}
	if m.ContextTokenUsage.TotalTokens != 240 {
		t.Errorf("Context total = %d, want 240 (latest call)", m.ContextTokenUsage.TotalTokens)
	}
}

func TestToolMetadataNilUsageDoesNotPanic(t *testing.T) {
	var m ToolMetadata

	m.recordModelUsage(nil)
	m.addBillingTokenUsage(nil)
	m.setContextTokenUsage(nil)

	if m.BillingTokenUsage != nil {
		t.Error("BillingTokenUsage should remain nil after nil recordModelUsage")
	}
	if m.ContextTokenUsage != nil {
		t.Error("ContextTokenUsage should remain nil after nil recordModelUsage")
	}
}

func TestToolMetadataMergePreservesUsageSemantics(t *testing.T) {
	var m ToolMetadata

	m.recordModelUsage(&schema.TokenUsage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120})

	// merge ToolMetadata with billing only (as tools do)
	toolMeta := ToolMetadata{
		BillingTokenUsage: &schema.TokenUsage{PromptTokens: 50, CompletionTokens: 10, TotalTokens: 60},
	}
	m.merge(toolMeta)

	if m.BillingTokenUsage == nil {
		t.Fatal("BillingTokenUsage should not be nil")
	}
	if m.BillingTokenUsage.PromptTokens != 150 {
		t.Errorf("Billing cumulative prompt = %d, want 150", m.BillingTokenUsage.PromptTokens)
	}
	if m.BillingTokenUsage.TotalTokens != 180 {
		t.Errorf("Billing cumulative total = %d, want 180", m.BillingTokenUsage.TotalTokens)
	}
	// Context should remain the first model call's usage
	if m.ContextTokenUsage.PromptTokens != 100 {
		t.Errorf("Context prompt = %d, want 100 (first call, unchanged)", m.ContextTokenUsage.PromptTokens)
	}
}

func TestToolMetadataSetContextTokenUsageOverwrites(t *testing.T) {
	var m ToolMetadata

	m.setContextTokenUsage(&schema.TokenUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15})
	m.setContextTokenUsage(&schema.TokenUsage{PromptTokens: 99, CompletionTokens: 33, TotalTokens: 132})

	if m.ContextTokenUsage.PromptTokens != 99 {
		t.Errorf("Context prompt = %d, want 99 (last set)", m.ContextTokenUsage.PromptTokens)
	}
	if m.ContextTokenUsage.TotalTokens != 132 {
		t.Errorf("Context total = %d, want 132 (last set)", m.ContextTokenUsage.TotalTokens)
	}
}
