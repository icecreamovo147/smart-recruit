package profile

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	RefreshReasonMissing      = "missing"
	RefreshReasonHeuristic    = "heuristic"
	RefreshReasonInputChanged = "input_changed"
	RefreshReasonForced       = "forced"
	RefreshReasonReused       = "reused"
	RefreshReasonNoParsedText = "no_parsed_text"
	RefreshReasonNoResume     = "no_resume"

	HeuristicParserVersionMarker = "heuristic"
)

// ResumeFillRefreshInput is the evidence used to decide whether to re-parse.
type ResumeFillRefreshInput struct {
	ForceRefresh   bool
	HasResume      bool
	ParsedText     string
	HasProfile     bool
	ParserVersion  string
	StoredInputHash string
}

// EvaluateResumeFillRefresh decides whether gateway should trigger AI parse.
func EvaluateResumeFillRefresh(input ResumeFillRefreshInput) (needsRefresh bool, reason string) {
	if !input.HasResume {
		return false, RefreshReasonNoResume
	}
	if strings.TrimSpace(input.ParsedText) == "" {
		return false, RefreshReasonNoParsedText
	}
	if input.ForceRefresh {
		return true, RefreshReasonForced
	}
	if !input.HasProfile {
		return true, RefreshReasonMissing
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(input.ParserVersion)), HeuristicParserVersionMarker) {
		return true, RefreshReasonHeuristic
	}
	currentHash := HashResumeParsedText(input.ParsedText)
	stored := strings.TrimSpace(input.StoredInputHash)
	if stored == "" || !strings.EqualFold(stored, currentHash) {
		return true, RefreshReasonInputChanged
	}
	return false, RefreshReasonReused
}

// HashResumeParsedText mirrors ai-agent resumeInputHash (sha256 hex).
func HashResumeParsedText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
