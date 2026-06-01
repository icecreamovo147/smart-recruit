package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
)

const maxCandidateContextMessages = 20

// buildCandidateAgentMessages assembles the message list for candidate ADK
// agent execution. It loads the most recent messages from the current
// session (only the candidate's own session) so multi-turn follow-up
// questions benefit from conversation context.
//
// The system message is always included first, followed by historical
// messages in time-ascending order, and finally the current user message.
// If the current message was just persisted as the last history entry,
// that copy is skipped in the history loop to avoid duplication.
func (s *CandidateAIService) buildCandidateAgentMessages(
	ctx context.Context,
	userID int64,
	sessionID int64,
	currentMessage string,
) ([]*schema.Message, error) {
	session, err := s.chats.GetSessionOwnedBy(ctx, ownerRoleCandidate, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("会话不存在或无权限访问")
	}

	messages := []*schema.Message{
		schema.SystemMessage(candidateSystemPrompt),
	}

	history, err := s.chats.ListRecentBySessionOwned(ctx, ownerRoleCandidate, userID, sessionID, maxCandidateContextMessages)
	if err != nil {
		return nil, fmt.Errorf("list recent session messages: %w", err)
	}

	currentMsgTrimmed := strings.TrimSpace(currentMessage)

	// Find whether the last message is a user message matching the current
	// one — that means it was just persisted and should not be duplicated.
	skipLastMatch := false
	if len(history) > 0 {
		last := history[len(history)-1]
		if last.Role == "user" && strings.TrimSpace(last.Content) == currentMsgTrimmed {
			skipLastMatch = true
		}
	}

	for i, h := range history {
		content := strings.TrimSpace(h.Content)
		if content == "" {
			continue
		}
		// Only skip the most recent matching user message to avoid
		// duplication of the just-persisted current message. Earlier
		// occurrences of the same text are preserved as valid context.
		if skipLastMatch && i == len(history)-1 && h.Role == "user" && content == currentMsgTrimmed {
			continue
		}
		role := schema.Assistant
		if h.Role == "user" {
			role = schema.User
		}
		messages = append(messages, &schema.Message{Role: role, Content: h.Content})
	}

	// Always append the current user message. If it was also the last
	// history entry, the loop above skipped that copy to avoid duplication.
	if currentMsgTrimmed != "" {
		messages = append(messages, schema.UserMessage(currentMessage))
	}

	return messages, nil
}
