package mq

import (
	"context"
	"fmt"

	"logic-grpc-service/internal/platform/events"
)

type EnvelopeHandler func(ctx context.Context, envelope *events.Envelope) error

func (c *Conn) PublishEnvelope(ctx context.Context, routingKey string, envelope events.Envelope) error {
	body, err := events.MarshalJSONEnvelope(envelope)
	if err != nil {
		return fmt.Errorf("marshal event envelope: %w", err)
	}
	return c.Publish(ctx, routingKey, body)
}

func (c *Conn) ConsumeEnvelope(ctx context.Context, queue string, handler EnvelopeHandler) error {
	if handler == nil {
		return fmt.Errorf("event envelope handler is required")
	}
	return c.Consume(ctx, queue, func(ctx context.Context, body []byte) error {
		envelope, err := events.UnmarshalJSONEnvelope(body)
		if err != nil {
			return fmt.Errorf("decode event envelope: %w", err)
		}
		return handler(ctx, envelope)
	})
}

type QueuePlan struct {
	Queue      string
	Retry      string
	DLQ        string
	RoutingKey string
}

func (c *Conn) QueuePlans() []QueuePlan {
	c.mu.Lock()
	defer c.mu.Unlock()
	plans := make([]QueuePlan, 0, len(c.bindings()))
	for _, binding := range c.bindings() {
		plans = append(plans, QueuePlan{
			Queue:      binding.name,
			Retry:      binding.name + ".retry",
			DLQ:        binding.name + ".dlq",
			RoutingKey: binding.routingKey,
		})
	}
	return plans
}

type ReplayPlan struct {
	Queue       string
	DLQ         string
	Retry       string
	Envelope    bool
	Idempotency bool
}

func (plan ReplayPlan) Validate() error {
	if plan.Queue == "" {
		return fmt.Errorf("replay queue is required")
	}
	if plan.DLQ == "" {
		return fmt.Errorf("replay dlq is required")
	}
	if plan.Retry == "" {
		return fmt.Errorf("replay retry queue is required")
	}
	if !plan.Envelope {
		return fmt.Errorf("replay requires event envelope validation")
	}
	if !plan.Idempotency {
		return fmt.Errorf("replay requires inbox/idempotency check")
	}
	return nil
}

func ReplayPlans(queuePlans []QueuePlan) []ReplayPlan {
	plans := make([]ReplayPlan, 0, len(queuePlans))
	for _, queue := range queuePlans {
		plans = append(plans, ReplayPlan{
			Queue:       queue.Queue,
			DLQ:         queue.DLQ,
			Retry:       queue.Retry,
			Envelope:    true,
			Idempotency: true,
		})
	}
	return plans
}
