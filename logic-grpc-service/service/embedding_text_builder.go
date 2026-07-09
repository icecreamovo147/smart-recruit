package service

import (
	"strings"
)

const maxEmbeddingTextLen = 8192

func BuildAgentSkillEmbeddingText(skillName, description, bodyMarkdown string, semanticTags []string) string {
	parts := make([]string, 0, 4)
	if skillName != "" {
		parts = append(parts, "Skill Name: "+skillName)
	}
	if description != "" {
		parts = append(parts, "Description: "+description)
	}
	if len(semanticTags) > 0 {
		parts = append(parts, "Tags: "+strings.Join(semanticTags, ", "))
	}
	if bodyMarkdown != "" {
		body := stripMarkdown(bodyMarkdown)
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
