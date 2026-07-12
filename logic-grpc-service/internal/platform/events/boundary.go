package events

import (
	"errors"
	"fmt"
	"strings"
)

type ConsumerBoundary struct {
	Name                 string
	Owner                string
	Queue                string
	SideEffect           string
	IdempotencyKeySource string
	InboxRequired        bool
	RetryRequired        bool
	DLQRequired          bool
	ReplayRequired       bool
}

func DefaultConsumerBoundaries() []ConsumerBoundary {
	return []ConsumerBoundary{
		{Name: "notification-consumer", Owner: "notification", Queue: "RABBITMQ_NOTIFICATION_QUEUE", SideEffect: "create notification records", IdempotencyKeySource: "envelope.idempotency_key + notification business key", InboxRequired: true, RetryRequired: true, DLQRequired: true, ReplayRequired: true},
		{Name: "email-consumer", Owner: "notification", Queue: "RABBITMQ_EMAIL_QUEUE", SideEffect: "send email and persist email log", IdempotencyKeySource: "envelope.idempotency_key + email log unique key", InboxRequired: true, RetryRequired: true, DLQRequired: true, ReplayRequired: true},
		{Name: "resume-parse-consumer", Owner: "recruitment", Queue: "RABBITMQ_RESUME_PARSE_QUEUE", SideEffect: "parse uploaded resume and update derived resume text/profile state", IdempotencyKeySource: "envelope.idempotency_key + resume parse run", InboxRequired: true, RetryRequired: true, DLQRequired: true, ReplayRequired: true},
		{Name: "embedding-consumer", Owner: "ai-agent", Queue: "RABBITMQ_EMBEDDING_QUEUE", SideEffect: "upsert embeddings", IdempotencyKeySource: "envelope.idempotency_key + content hash + model key", InboxRequired: true, RetryRequired: true, DLQRequired: true, ReplayRequired: true},
		{Name: "agent-run-consumer", Owner: "ai-agent", Queue: "RABBITMQ_AGENT_RUN_QUEUE", SideEffect: "execute durable agent run", IdempotencyKeySource: "envelope.idempotency_key + agent run status transition", InboxRequired: true, RetryRequired: true, DLQRequired: true, ReplayRequired: true},
		{Name: "analytics-projection-consumer", Owner: "analytics", Queue: "domain-event projections", SideEffect: "project domain event into analytics read models", IdempotencyKeySource: "envelope.event_id + analytics projection ledger", InboxRequired: true, RetryRequired: true, DLQRequired: true, ReplayRequired: true},
	}
}

func ValidateConsumerBoundaries(boundaries []ConsumerBoundary) error {
	if len(boundaries) == 0 {
		return errors.New("consumer boundaries are required")
	}
	seen := map[string]struct{}{}
	var problems []error
	for _, boundary := range boundaries {
		name := strings.TrimSpace(boundary.Name)
		if name == "" {
			problems = append(problems, errors.New("consumer boundary name is required"))
			continue
		}
		if _, exists := seen[name]; exists {
			problems = append(problems, fmt.Errorf("duplicate consumer boundary %q", name))
		}
		seen[name] = struct{}{}
		if strings.TrimSpace(boundary.Owner) == "" {
			problems = append(problems, fmt.Errorf("%s owner is required", name))
		}
		if strings.TrimSpace(boundary.Queue) == "" {
			problems = append(problems, fmt.Errorf("%s queue is required", name))
		}
		if strings.TrimSpace(boundary.SideEffect) == "" {
			problems = append(problems, fmt.Errorf("%s side effect is required", name))
		}
		if strings.TrimSpace(boundary.IdempotencyKeySource) == "" {
			problems = append(problems, fmt.Errorf("%s idempotency key source is required", name))
		}
		if !boundary.InboxRequired {
			problems = append(problems, fmt.Errorf("%s must require inbox idempotency", name))
		}
		if !boundary.RetryRequired {
			problems = append(problems, fmt.Errorf("%s must require retry", name))
		}
		if !boundary.DLQRequired {
			problems = append(problems, fmt.Errorf("%s must require dlq", name))
		}
		if !boundary.ReplayRequired {
			problems = append(problems, fmt.Errorf("%s must require replay", name))
		}
	}
	if len(problems) > 0 {
		return errors.Join(problems...)
	}
	return nil
}

type ReplayRule struct {
	Source           string
	RequiresEnvelope bool
	RequiresInbox    bool
	RequiresDLQ      bool
	RequiresOperator bool
}

func DefaultReplayRule() ReplayRule {
	return ReplayRule{
		Source:           "event_outbox or queue.dlq",
		RequiresEnvelope: true,
		RequiresInbox:    true,
		RequiresDLQ:      true,
		RequiresOperator: true,
	}
}

func (rule ReplayRule) Validate() error {
	var problems []error
	if strings.TrimSpace(rule.Source) == "" {
		problems = append(problems, errors.New("replay source is required"))
	}
	if !rule.RequiresEnvelope {
		problems = append(problems, errors.New("replay must require event envelope validation"))
	}
	if !rule.RequiresInbox {
		problems = append(problems, errors.New("replay must require inbox idempotency"))
	}
	if !rule.RequiresDLQ {
		problems = append(problems, errors.New("replay must require dlq source support"))
	}
	if !rule.RequiresOperator {
		problems = append(problems, errors.New("replay must require explicit operator action"))
	}
	if len(problems) > 0 {
		return errors.Join(problems...)
	}
	return nil
}
