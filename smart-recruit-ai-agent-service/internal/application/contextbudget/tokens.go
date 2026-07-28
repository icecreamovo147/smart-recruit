package contextbudget

import (
	"encoding/json"

	"github.com/cloudwego/eino/schema"

	domaintokenbudget "smart-recruit-ai-agent-service/internal/domain/tokenbudget"
)

func EstimateTokensConservative(value string) int {
	return domaintokenbudget.EstimateConservative(value)
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
