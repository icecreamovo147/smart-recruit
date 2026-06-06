package model

// ── Application Status Keys ────────────────────────────────────────────
// Stable string keys for the recruiting pipeline state machine.
// These replace the legacy numeric status values (0-3) in new code.
// The legacy `status` column is preserved temporarily for migration compatibility.

const (
	StatusKeyApplied         = "applied"
	StatusKeyViewed          = "viewed"
	StatusKeyScreening       = "screening"
	StatusKeyScreenPassed    = "screen_passed"
	StatusKeyInterviewPending = "interview_pending"
	StatusKeyInterviewing    = "interviewing"
	StatusKeyInterviewPassed = "interview_passed"
	StatusKeyOfferPending    = "offer_pending"
	StatusKeyOfferSent       = "offer_sent"
	StatusKeyOfferAccepted   = "offer_accepted"
	StatusKeyOfferRejected   = "offer_rejected"
	StatusKeyHired           = "hired"
	StatusKeyRejected        = "rejected"
	StatusKeyWithdrawn       = "withdrawn"
)

// TerminalStatusKeys are statuses that close the current active application round
// and allow the candidate to create a new round. Rejected also has one staff-side
// exception in the transition validator: HR can re-pass it into screen_passed,
// which increments round_no and reopens the same application record as a new round.
var TerminalStatusKeys = map[string]bool{
	StatusKeyRejected:      true,
	StatusKeyWithdrawn:     true,
	StatusKeyOfferRejected: true,
	StatusKeyHired:         true,
}

// IsTerminal returns true if the given status key is a terminal state.
func IsTerminalStatusKey(key string) bool {
	return TerminalStatusKeys[key]
}

// TerminalStatusKeyList returns the terminal status keys as a slice for use in SQL queries.
func TerminalStatusKeyList() []string {
	keys := make([]string, 0, len(TerminalStatusKeys))
	for k := range TerminalStatusKeys {
		keys = append(keys, k)
	}
	return keys
}

// ── Candidate-Safe Labels ──────────────────────────────────────────────
// Status labels shown to candidates. These are intentionally less specific
// than internal statuses and never expose private HR reasons.

var CandidateStatusLabels = map[string]string{
	StatusKeyApplied:         "已投递",
	StatusKeyViewed:          "简历被查看",
	StatusKeyScreening:       "筛选中",
	StatusKeyScreenPassed:    "筛选通过",
	StatusKeyInterviewPending: "待面试",
	StatusKeyInterviewing:    "面试中",
	StatusKeyInterviewPassed: "面试通过",
	StatusKeyOfferPending:    "待发offer",
	StatusKeyOfferSent:       "Offer已发",
	StatusKeyOfferAccepted:   "Offer已接受",
	StatusKeyOfferRejected:   "Offer已拒绝",
	StatusKeyHired:           "已入职",
	StatusKeyRejected:        "未通过",
	StatusKeyWithdrawn:       "已撤回",
}

// ── HR-Facing Labels ───────────────────────────────────────────────────
// Internal status labels shown to HR users.

var HRStatusLabels = map[string]string{
	StatusKeyApplied:         "待查看",
	StatusKeyViewed:          "已查看",
	StatusKeyScreening:       "筛选中",
	StatusKeyScreenPassed:    "筛选通过",
	StatusKeyInterviewPending: "待安排面试",
	StatusKeyInterviewing:    "面试中",
	StatusKeyInterviewPassed: "面试通过",
	StatusKeyOfferPending:    "待发Offer",
	StatusKeyOfferSent:       "Offer已发",
	StatusKeyOfferAccepted:   "Offer已接受",
	StatusKeyOfferRejected:   "Offer被拒",
	StatusKeyHired:           "已入职",
	StatusKeyRejected:        "淘汰",
	StatusKeyWithdrawn:       "候选人撤回",
}

// ── Legacy Numeric Mapping ─────────────────────────────────────────────
// Maps legacy numeric status values to new string status keys.

var LegacyStatusToKey = map[int32]string{
	0: StatusKeyApplied,
	1: StatusKeyViewed,
	2: StatusKeyScreenPassed,
	3: StatusKeyRejected,
}

// StatusKeyToLegacy maps string status keys to legacy numeric values.
var StatusKeyToLegacy = map[string]int32{
	StatusKeyApplied:         0,
	StatusKeyViewed:          1,
	StatusKeyScreening:       1,
	StatusKeyScreenPassed:    2,
	StatusKeyInterviewPending: 2,
	StatusKeyInterviewing:    2,
	StatusKeyInterviewPassed: 2,
	StatusKeyOfferPending:    2,
	StatusKeyOfferSent:       2,
	StatusKeyOfferAccepted:   2,
	StatusKeyHired:           2,
	StatusKeyOfferRejected:   3,
	StatusKeyRejected:        3,
	StatusKeyWithdrawn:       3,
}

// DefaultStatusKey returns the default status key for a newly created application.
func DefaultStatusKey() string {
	return StatusKeyApplied
}

// ── Reapplication Rules ────────────────────────────────────────────────

// CanReapply returns true if the current status key closes the current round.
// Candidates can reapply only from these configured closeout states.
func CanReapply(statusKey string) bool {
	return IsTerminalStatusKey(statusKey)
}
