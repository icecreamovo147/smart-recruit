package service

import (
	"strings"
	"testing"
)

func TestBuildAgentSkillEmbeddingText(t *testing.T) {
	tests := []struct {
		name       string
		skillName  string
		desc       string
		bodyMD     string
		tags       []string
		wantPrefix string
		wantEmpty  bool
	}{
		{
			name:       "full fields",
			skillName:  "TestSkill",
			desc:       "A test skill description",
			bodyMD:     "## Content\nsome body",
			tags:       []string{"tag1", "tag2"},
			wantPrefix: "Skill Name: TestSkill",
		},
		{
			name:       "empty name and description",
			skillName:  "",
			desc:       "",
			bodyMD:     "",
			tags:       nil,
			wantEmpty:  true,
		},
		{
			name:       "only tags",
			skillName:  "",
			desc:       "",
			bodyMD:     "",
			tags:       []string{"tag1"},
			wantPrefix: "Tags: tag1",
		},
		{
			name:       "long body truncated",
			skillName:  "S",
			desc:       "D",
			bodyMD:     strings.Repeat("x", 3000),
			tags:       nil,
			wantPrefix: "Skill Name: S",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildAgentSkillEmbeddingText(tt.skillName, tt.desc, tt.bodyMD, tt.tags)
			if tt.wantEmpty && got != "" {
				t.Errorf("expected empty, got %q", got)
			}
			if !tt.wantEmpty && !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("expected prefix %q, got %q", tt.wantPrefix, got)
			}
			if len(got) > maxEmbeddingTextLen {
				t.Errorf("text too long: %d > %d", len(got), maxEmbeddingTextLen)
			}
		})
	}
}

func TestBuildMemoryEmbeddingText(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		memoryType       string
		scopeDescription string
		wantPrefix       string
		wantEmpty        bool
	}{
		{
			name:             "full fields",
			content:          "candidate is skilled in Go",
			memoryType:       "conclusion",
			scopeDescription: "application:42",
			wantPrefix:       "Type: conclusion",
		},
		{
			name:      "empty content",
			content:   "",
			memoryType: "conclusion",
			wantPrefix: "Type: conclusion",
		},
		{
			name:       "only content",
			content:    "some memory content",
			memoryType: "",
			wantPrefix: "Content: some memory content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildMemoryEmbeddingText(tt.content, tt.memoryType, tt.scopeDescription)
			if tt.wantEmpty && got != "" {
				t.Errorf("expected empty, got %q", got)
			}
			if !tt.wantEmpty && !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("expected prefix %q, got %q", tt.wantPrefix, got)
			}
			if len(got) > maxEmbeddingTextLen {
				t.Errorf("text too long: %d > %d", len(got), maxEmbeddingTextLen)
			}
		})
	}
}

func TestStripMarkdown(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "# Title", want: "Title"},
		{input: "**bold**", want: "bold"},
		{input: "_italic_", want: "italic"},
		{input: "plain text", want: "plain text"},
		{input: "```\ncode\n```", want: "code"},
		{input: "> quote", want: "quote"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripMarkdown(tt.input)
			if got != tt.want {
				t.Errorf("stripMarkdown(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildAgentSkillEmbeddingTextMaxLen(t *testing.T) {
	longBody := strings.Repeat("A", 10000)
	result := BuildAgentSkillEmbeddingText("Test", "Desc", longBody, []string{"tag"})
	if len(result) > maxEmbeddingTextLen {
		t.Errorf("exceeded max length: %d > %d", len(result), maxEmbeddingTextLen)
	}
}
