package memory

import (
	"context"
	"strings"

	domainmemory "smart-recruit-ai-agent-service/internal/domain/memory"
)

type Extractor struct {
	model ModelExtractor
}

func NewExtractor(model ModelExtractor) *Extractor {
	return &Extractor{model: model}
}

func (e *Extractor) Extract(ctx context.Context, input ExtractInput) ([]ExtractCandidate, error) {
	out := ruleExtract(input)
	if e != nil && e.model != nil {
		modelCandidates, err := e.model.Extract(ctx, input)
		if err == nil && len(modelCandidates) > 0 {
			out = append(out, modelCandidates...)
		}
	}
	return dedupeExtractCandidates(out), nil
}

func ruleExtract(input ExtractInput) []ExtractCandidate {
	text := strings.TrimSpace(input.UserText)
	if text == "" {
		return nil
	}
	lower := strings.ToLower(text)
	out := make([]ExtractCandidate, 0, 2)
	scope := defaultScope(input.Owner.Role, input.Owner.ID)

	if strings.Contains(text, "请记住") || strings.Contains(lower, "remember") {
		content := strings.TrimSpace(strings.TrimPrefix(text, "请记住"))
		content = strings.TrimSpace(strings.TrimPrefix(content, "remember"))
		if content == "" {
			content = text
		}
		out = append(out, ExtractCandidate{
			Scope:      scope,
			MemoryType: "preference",
			Content:    content,
			Confidence: 0.95,
			Importance: 0.9,
			Source:     "user",
		})
	}

	preferenceKeywords := []string{"偏好", "喜欢", "习惯", "prefer", "preference"}
	for _, keyword := range preferenceKeywords {
		if strings.Contains(lower, keyword) {
			out = append(out, ExtractCandidate{
				Scope:      scope,
				MemoryType: "preference",
				Content:    text,
				Confidence: 0.85,
				Importance: 0.8,
				Source:     "user",
			})
			break
		}
	}
	return out
}

func defaultScope(ownerRole domainmemory.OwnerRole, ownerID uint64) domainmemory.Scope {
	if ownerRole == domainmemory.OwnerRoleCandidate {
		return domainmemory.Scope{Type: domainmemory.ScopeUser, ID: ownerID}
	}
	return domainmemory.Scope{Type: domainmemory.ScopeHR, ID: 0}
}

func dedupeExtractCandidates(items []ExtractCandidate) []ExtractCandidate {
	seen := make(map[string]struct{}, len(items))
	out := make([]ExtractCandidate, 0, len(items))
	for _, item := range items {
		key := item.Scope.Type + ":" + strings.TrimSpace(item.Content)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if strings.TrimSpace(item.Content) == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}
