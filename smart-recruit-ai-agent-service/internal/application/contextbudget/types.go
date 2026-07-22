package contextbudget

import "context"

type ModelInfo struct {
	ContextWindowTokens int32
	MaxOutputTokens     int32
}

type MessageRow struct {
	ID      int64
	Role    string
	Content string
}

type SessionSummaryStore interface {
	GetSessionSummaryState(ctx context.Context, ownerID, sessionID int64) (summary string, coveredMessageID int64, messageCount int, found bool, err error)
	UpsertSessionSummaryIfNewer(ctx context.Context, ownerID, sessionID int64, summary string, coveredMessageID int64, messageCount int) (bool, error)
}

type SummaryGenerator interface {
	GenerateSessionSummary(ctx context.Context, oldSummary string, recentMessages []string) (string, error)
}

type AudienceSummaryGenerator interface {
	GenerateSessionSummaryForAudience(ctx context.Context, audience string, oldSummary string, recentMessages []string) (string, error)
}

type PrepareResult struct {
	SummaryApplied bool
	MemoryApplied  bool
	OmittedCount   int
}
