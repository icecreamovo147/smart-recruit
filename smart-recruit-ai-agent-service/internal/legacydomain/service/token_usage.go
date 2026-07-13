package service

import "github.com/cloudwego/eino/schema"

func tokenUsageTotal(usage *schema.TokenUsage) int {
	if usage == nil {
		return 0
	}
	return usage.TotalTokens
}
