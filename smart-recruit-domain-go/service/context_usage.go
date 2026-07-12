package service

import (
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"

	pb "smart-recruit-proto/recruitment/pb"
)

// TokenEstimator provides a rough token count estimation for text.
// This is a heuristic placeholder; exact tokenizer integration can replace it later.
type TokenEstimator struct{}

func NewTokenEstimator() *TokenEstimator {
	return &TokenEstimator{}
}

// EstimateText returns an estimated token count for the given text.
// ASCII-heavy text: chars / 4. CJK/mixed text: chars / 2. Always >= 1 for non-empty.
func (e *TokenEstimator) EstimateText(text string) int {
	if text == "" {
		return 0
	}
	n := utf8.RuneCountInString(text)
	cjkCount := 0
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF || r >= 0x3000 && r <= 0x303F || r >= 0xFF00 && r <= 0xFFEF {
			cjkCount++
		}
	}
	asciiCount := n - cjkCount
	estimated := cjkCount/2 + asciiCount/4
	if estimated < 1 {
		estimated = 1
	}
	return estimated
}

// EstimateMessages returns a ContextUsageBreakdown by categorizing the given messages.
func (e *TokenEstimator) EstimateMessages(messages []*schema.Message) *pb.ContextUsageBreakdown {
	bd := &pb.ContextUsageBreakdown{}
	lastUserIndex := -1
	for i, msg := range messages {
		if msg.Role == schema.User {
			lastUserIndex = i
		}
	}
	for i, msg := range messages {
		tokens := e.EstimateText(msg.Content)
		switch msg.Role {
		case schema.System:
			bd.SystemPromptTokens += int32(tokens)
		case schema.User:
			if i == lastUserIndex {
				bd.CurrentMessageTokens += int32(tokens)
			} else {
				bd.RecentMessageTokens += int32(tokens)
			}
		case schema.Assistant:
			bd.RecentMessageTokens += int32(tokens)
		case schema.Tool:
			bd.ToolResultTokens += int32(tokens)
		default:
			bd.RecentMessageTokens += int32(tokens)
		}
	}
	return bd
}

// ContextUsageBuilder constructs ContextUsageInfo from model config and context data.
type ContextUsageBuilder struct {
	estimator *TokenEstimator
}

func NewContextUsageBuilder() *ContextUsageBuilder {
	return &ContextUsageBuilder{
		estimator: NewTokenEstimator(),
	}
}

type UsageBuildInput struct {
	ModelID              int64
	ModelName            string
	ContextWindowTokens  int32
	MaxOutputTokens      int32
	SystemPromptTokens   int32
	RecentMessageTokens  int32
	SummaryTokens        int32
	MemoryTokens         int32
	CurrentMessageTokens int32
	SkillTokens          int32
	ToolResultTokens     int32
	Stage                string // initial | tool_result | final
	Source               string // estimator | provider | unknown_config
}

// Build constructs a ContextUsageInfo from the input.
// If ContextWindowTokens is 0, ratio/remaining are set to 0. Estimated describes
// whether token counts come from a local estimator rather than provider usage.
func (b *ContextUsageBuilder) Build(input UsageBuildInput) *pb.ContextUsageInfo {
	source := input.Source
	if source == "" {
		source = "estimator"
	}
	info := &pb.ContextUsageInfo{
		ModelId:             input.ModelID,
		ModelName:           input.ModelName,
		ContextWindowTokens: input.ContextWindowTokens,
		MaxOutputTokens:     input.MaxOutputTokens,
		Source:              source,
		Stage:               input.Stage,
		Estimated:           source != "provider",
		Breakdown: &pb.ContextUsageBreakdown{
			SystemPromptTokens:   input.SystemPromptTokens,
			RecentMessageTokens:  input.RecentMessageTokens,
			SummaryTokens:        input.SummaryTokens,
			MemoryTokens:         input.MemoryTokens,
			CurrentMessageTokens: input.CurrentMessageTokens,
			SkillTokens:          input.SkillTokens,
			ToolResultTokens:     input.ToolResultTokens,
		},
	}

	promptEstimated := input.SystemPromptTokens + input.RecentMessageTokens +
		input.SummaryTokens + input.MemoryTokens +
		input.CurrentMessageTokens + input.SkillTokens + input.ToolResultTokens
	info.PromptTokensEstimated = promptEstimated

	if input.ContextWindowTokens > 0 {
		remaining := input.ContextWindowTokens - promptEstimated - input.MaxOutputTokens
		if remaining < 0 {
			remaining = 0
		}
		info.RemainingTokensEstimated = remaining

		ratio := float64(promptEstimated) / float64(input.ContextWindowTokens)
		if ratio > 1.0 {
			ratio = 1.0
		}
		info.UsageRatio = ratio
	} else {
		info.RemainingTokensEstimated = 0
		info.UsageRatio = 0
	}

	return info
}
