package contextbudget

const (
	SummaryMessagePrefix  = "[Rolling conversation summary]\n"
	MemoryMessagePrefix   = "[Long-term memories]\n"
	UnknownContextHistory = 20

	BudgetExceededCode = "AI_CONTEXT_BUDGET_EXCEEDED"
	ConfigInvalidCode  = "AI_CONTEXT_CONFIGURATION_INVALID"

	AudienceHR        = "hr"
	AudienceCandidate = "candidate"
)
