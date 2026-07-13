package runtime

import (
	"fmt"
	"sort"
	"strings"
)

const (
	OwnerPlatform     = "platform"
	OwnerNotification = "notification"
	OwnerRecruitment  = "recruitment"
	OwnerAIAgent      = "ai-agent"
	OwnerAnalytics    = "analytics"
)

type WorkloadToggle struct {
	EnableEnv  string
	DisableEnv string
	DefaultOn  bool
}

type OwnerContract struct {
	OwnerContext string
	OwnedTables  []string
	ReadTables   []string
	Writes       string
	Idempotency  string
	DLQ          string
	Notes        string
}

type WorkloadProfile struct {
	Name        string
	Queue       string
	Category    string
	Toggle      WorkloadToggle
	Contract    OwnerContract
	Description string
}

var DefaultWorkloadProfiles = []WorkloadProfile{
	{
		Name:        "outbox-dispatcher",
		Queue:       "event_outbox",
		Category:    "outbox",
		Toggle:      defaultToggle(),
		Description: "Publishes pending outbox events to RabbitMQ without owning source-domain tables.",
		Contract: OwnerContract{
			OwnerContext: OwnerPlatform,
			OwnedTables:  []string{"event_outbox"},
			Writes:       "Claims pending outbox rows and marks published/dead through outbox status transitions.",
			Idempotency:  "outbox claim/mark-published status transition",
			DLQ:          "event_outbox dead status with dead_lettered_at and retry diagnostics",
			Notes:        "Must not mutate aggregate owner tables while publishing events.",
		},
	},
	{
		Name:        "notification-consumer",
		Queue:       "RABBITMQ_NOTIFICATION_QUEUE",
		Category:    "notification",
		Toggle:      defaultToggle(),
		Description: "Consumes notification.create events and writes Notification-owned rows/cache effects.",
		Contract: OwnerContract{
			OwnerContext: OwnerNotification,
			OwnedTables:  []string{"notifications", "event_inbox"},
			ReadTables:   []string{"users"},
			Writes:       "Creates notification records and consumer inbox claims only.",
			Idempotency:  "notification inbox business key",
			DLQ:          "notification queue DLQ plus event_inbox dead status",
			Notes:        "Account type and actor scope must remain notification-owned.",
		},
	},
	{
		Name:        "email-consumer",
		Queue:       "RABBITMQ_EMAIL_QUEUE",
		Category:    "email",
		Toggle:      defaultToggle(),
		Description: "Consumes email.send events and records email delivery attempts.",
		Contract: OwnerContract{
			OwnerContext: OwnerNotification,
			OwnedTables:  []string{"email_logs", "event_inbox"},
			ReadTables:   []string{"users"},
			Writes:       "Records email log status and inbox claims; sends email through configured sender.",
			Idempotency:  "email inbox and email log uniqueness",
			DLQ:          "email queue DLQ plus event_inbox dead status",
			Notes:        "Reports must never include raw email body or full recipient addresses.",
		},
	},
	{
		Name:        "resume-parse-consumer",
		Queue:       "RABBITMQ_RESUME_PARSE_QUEUE",
		Category:    "resume",
		Toggle:      defaultToggle(),
		Description: "Consumes resume parse events and updates resume parsed-text state.",
		Contract: OwnerContract{
			OwnerContext: OwnerRecruitment,
			OwnedTables:  []string{"resumes", "event_inbox"},
			ReadTables:   []string{"users"},
			Writes:       "Updates resume parse status/text only for the resume owner context.",
			Idempotency:  "resume parse run and resume idempotency",
			DLQ:          "resume parse queue DLQ plus event_inbox dead status",
			Notes:        "Do not log raw resume text, object keys, or parsed content.",
		},
	},
	{
		Name:        "embedding-consumer",
		Queue:       "RABBITMQ_EMBEDDING_QUEUE",
		Category:    "embedding",
		Toggle:      defaultToggle(),
		Description: "Consumes embedding upsert events for AI-owned semantic retrieval objects.",
		Contract: OwnerContract{
			OwnerContext: OwnerAIAgent,
			OwnedTables:  []string{"ai_embeddings", "event_inbox"},
			ReadTables:   []string{"agent_skills", "memories", "embedding_model_configs", "embedding_provider_configs"},
			Writes:       "Upserts embedding rows and inbox claims only.",
			Idempotency:  "embedding content hash and model key",
			DLQ:          "embedding queue DLQ plus event_inbox dead status",
			Notes:        "Provider credentials must stay encrypted and out of logs.",
		},
	},
	{
		Name:        "agent-run-consumer",
		Queue:       "RABBITMQ_AGENT_RUN_QUEUE",
		Category:    "agent-run",
		Toggle:      defaultToggle(),
		Description: "Consumes durable agent-run execution requests.",
		Contract: OwnerContract{
			OwnerContext: OwnerAIAgent,
			OwnedTables:  []string{"agent_runs", "agent_run_events", "ai_chat_messages", "event_inbox"},
			ReadTables:   []string{"ai_chat_sessions", "agent_configs", "prompt_templates"},
			Writes:       "Transitions agent run status, appends run events, and persists final assistant message.",
			Idempotency:  "agent run status transition guard",
			DLQ:          "agent run queue DLQ plus event_inbox dead status",
			Notes:        "Cancel/confirm state machine must remain AI Agent-owned.",
		},
	},
	{
		Name:        "analytics-projection-consumer",
		Queue:       "domain-event projections",
		Category:    "analytics",
		Toggle:      defaultToggle(),
		Description: "Consumes domain events into Analytics-owned projection ledgers.",
		Contract: OwnerContract{
			OwnerContext: OwnerAnalytics,
			OwnedTables:  []string{"analytics_projection_events", "analytics_projection_checkpoints"},
			ReadTables:   []string{"event_outbox"},
			Writes:       "Upserts projection event ledger and checkpoints only.",
			Idempotency:  "analytics projection event id ledger",
			DLQ:          "projection event status/dead-letter metadata",
			Notes:        "Must not write recruitment owner tables while projecting.",
		},
	},
}

func ProfileByName(name string) (WorkloadProfile, bool) {
	for _, profile := range DefaultWorkloadProfiles {
		if profile.Name == name {
			return profile, true
		}
	}
	return WorkloadProfile{}, false
}

func ValidateWorkloadProfiles() error {
	seen := map[string]struct{}{}
	defaults := knownWorkloadSet()
	for _, profile := range DefaultWorkloadProfiles {
		if strings.TrimSpace(profile.Name) == "" {
			return fmt.Errorf("worker workload profile name is required")
		}
		if _, ok := defaults[profile.Name]; !ok {
			return fmt.Errorf("worker workload profile %q has no runtime workload descriptor", profile.Name)
		}
		if _, exists := seen[profile.Name]; exists {
			return fmt.Errorf("duplicate worker workload profile %q", profile.Name)
		}
		seen[profile.Name] = struct{}{}
		if strings.TrimSpace(profile.Contract.OwnerContext) == "" {
			return fmt.Errorf("worker workload %q owner context is required", profile.Name)
		}
		if strings.TrimSpace(profile.Contract.Idempotency) == "" {
			return fmt.Errorf("worker workload %q idempotency contract is required", profile.Name)
		}
		if strings.TrimSpace(profile.Contract.DLQ) == "" {
			return fmt.Errorf("worker workload %q DLQ contract is required", profile.Name)
		}
	}
	for name := range defaults {
		if _, ok := seen[name]; !ok {
			return fmt.Errorf("worker workload %q has no profile", name)
		}
	}
	return nil
}

func WorkloadProfileNames() []string {
	names := make([]string, 0, len(DefaultWorkloadProfiles))
	for _, profile := range DefaultWorkloadProfiles {
		names = append(names, profile.Name)
	}
	sort.Strings(names)
	return names
}

func defaultToggle() WorkloadToggle {
	return WorkloadToggle{EnableEnv: "WORKER_WORKLOADS", DisableEnv: "WORKER_DISABLED_WORKLOADS", DefaultOn: true}
}
