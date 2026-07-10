package service

import (
	"strings"
)

const maxEmbeddingTextLen = 8192

func BuildAgentSkillEmbeddingText(skillName, description, bodyMarkdown string, semanticTags []string) string {
	return BuildAgentSkillEmbeddingTextWithMetadata(AgentSkillEmbeddingTextInput{
		SkillName:    skillName,
		Description:  description,
		BodyMarkdown: bodyMarkdown,
		SemanticTags: semanticTags,
	})
}

type AgentSkillEmbeddingTextInput struct {
	SkillName          string
	Description        string
	BodyMarkdown       string
	Category           string
	Scenario           string
	RiskLevel          string
	TriggerKeywords    []string
	SemanticTags       []string
	EvaluationCriteria string
	OutputSchema       string
}

func BuildAgentSkillEmbeddingTextWithMetadata(input AgentSkillEmbeddingTextInput) string {
	parts := make([]string, 0, 4)
	if input.SkillName != "" {
		parts = append(parts, "Skill Name: "+input.SkillName)
	}
	if input.Description != "" {
		parts = append(parts, "Description: "+input.Description)
	}
	if input.Category != "" {
		parts = append(parts, "Category: "+input.Category)
	}
	if input.Scenario != "" {
		parts = append(parts, "Scenario: "+input.Scenario)
	}
	if input.RiskLevel != "" {
		parts = append(parts, "Risk Level: "+input.RiskLevel)
	}
	if len(input.TriggerKeywords) > 0 {
		parts = append(parts, "Trigger Keywords: "+strings.Join(input.TriggerKeywords, ", "))
	}
	if len(input.SemanticTags) > 0 {
		parts = append(parts, "Tags: "+strings.Join(input.SemanticTags, ", "))
	}
	if input.EvaluationCriteria != "" {
		parts = append(parts, "Evaluation Criteria: "+truncateEmbeddingSection(input.EvaluationCriteria, 1000))
	}
	if input.OutputSchema != "" {
		parts = append(parts, "Output Schema: "+truncateEmbeddingSection(input.OutputSchema, 1000))
	}
	if input.BodyMarkdown != "" {
		body := stripMarkdown(input.BodyMarkdown)
		if len(body) > 2000 {
			body = body[:2000]
		}
		parts = append(parts, "Content: "+body)
	}
	text := strings.Join(parts, "\n\n")
	if len(text) > maxEmbeddingTextLen {
		text = text[:maxEmbeddingTextLen]
	}
	return text
}

func truncateEmbeddingSection(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if len(text) > maxLen {
		return text[:maxLen]
	}
	return text
}

func BuildMemoryEmbeddingText(content string, memoryType string, scopeDescription string) string {
	parts := make([]string, 0, 3)
	if memoryType != "" {
		parts = append(parts, "Type: "+memoryType)
	}
	if scopeDescription != "" {
		parts = append(parts, "Scope: "+scopeDescription)
	}
	if content != "" {
		if len(content) > 2000 {
			parts = append(parts, "Content: "+content[:2000])
		} else {
			parts = append(parts, "Content: "+content)
		}
	}
	text := strings.Join(parts, "\n\n")
	if len(text) > maxEmbeddingTextLen {
		text = text[:maxEmbeddingTextLen]
	}
	return text
}

func stripMarkdown(md string) string {
	var buf strings.Builder
	inCodeBlock := false
	runes := []rune(md)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if strings.HasPrefix(md[i:], "```") {
			inCodeBlock = !inCodeBlock
			i += 2
			continue
		}
		if inCodeBlock {
			buf.WriteRune(ch)
			continue
		}
		switch ch {
		case '#', '>', '-':
			if i+1 < len(runes) && (runes[i+1] == ' ' || runes[i+1] == '#') {
				continue
			}
			buf.WriteRune(ch)
		case '*', '_', '`':
			continue
		default:
			buf.WriteRune(ch)
		}
	}
	return strings.TrimSpace(buf.String())
}
