package contextbudget

import (
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/schema"
)

func EstimateTokensConservative(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	ascii := 0
	nonASCII := 0
	for _, r := range trimmed {
		if r <= 0x7f {
			ascii++
		} else {
			nonASCII++
		}
	}
	return maxInt((ascii+3)/4+nonASCII, 1)
}

func EstimateToolSchemaTokens(tools []*schema.ToolInfo) int {
	total := 0
	for _, tool := range tools {
		if tool == nil {
			continue
		}
		payload, err := json.Marshal(tool)
		if err != nil {
			continue
		}
		total += EstimateTokensConservative(string(payload))
	}
	return total
}

func EstimateMessagesEnvelopeTokens(messages []*schema.Message, tools []*schema.ToolInfo) int {
	total := EstimateToolSchemaTokens(tools) + len(messages)*4 + 2
	for _, message := range messages {
		if message != nil {
			total += EstimateTokensConservative(message.Content)
		}
	}
	return total
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
