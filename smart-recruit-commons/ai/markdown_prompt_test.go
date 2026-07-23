package ai

import (
	"strings"
	"testing"
)

func TestRecruitingMessagesIncludeStandardMarkdownRules(t *testing.T) {
	messages := buildRecruitingMessages("今天有多少投递？", RecruitingStats{})
	if len(messages) == 0 {
		t.Fatal("expected system message")
	}
	system := messages[0].Content
	if !containsAll(system, []string{"Markdown 输出硬性规范", "禁止写成 \"-内容\"", "2026-05-25 **淘汰**"}) {
		t.Fatalf("recruiting system prompt missing strict markdown rules: %s", system)
	}
}

func TestApplicationAnalysisMessagesIncludeStandardMarkdownRules(t *testing.T) {
	messages := buildApplicationAnalysisMessages(ApplicationAnalysisInput{Question: "分析简历"})
	if len(messages) == 0 {
		t.Fatal("expected system message")
	}
	system := messages[0].Content
	if !containsAll(system, []string{"Markdown 输出硬性规范", "禁止写成 \"-内容\"", "**薪资：** 10000 元"}) {
		t.Fatalf("application analysis system prompt missing strict markdown rules: %s", system)
	}
}

func containsAll(text string, needles []string) bool {
	for _, needle := range needles {
		if !strings.Contains(text, needle) {
			return false
		}
	}
	return true
}
