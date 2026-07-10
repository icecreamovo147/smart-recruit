package service

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"logic-grpc-service/mq"
	"logic-grpc-service/pkg/logger"
)

type agentRunExecuteEnvelope struct {
	EventID string `json:"event_id"`
	RunID   uint64 `json:"run_id"`
}

// AgentRunConsumer executes durable HR Agent Runs from the shared
// Outbox/RabbitMQ worker mechanism.
type AgentRunConsumer struct {
	ai *AIService
}

func NewAgentRunConsumer(ai *AIService) *AgentRunConsumer {
	return &AgentRunConsumer{ai: ai}
}

func (c *AgentRunConsumer) Start(ctx context.Context, mqConn *mq.Conn) error {
	return mqConn.Consume(ctx, mqConn.AgentRunQueue(), func(ctx context.Context, body []byte) error {
		return c.handle(ctx, body)
	})
}

func (c *AgentRunConsumer) handle(ctx context.Context, body []byte) error {
	if c == nil || c.ai == nil {
		return fmt.Errorf("agent run consumer: ai service not configured")
	}
	var p agentRunExecuteEnvelope
	if err := json.Unmarshal(body, &p); err != nil {
		logger.L().Error("[agent-run-consumer] invalid payload", zap.Error(err))
		return fmt.Errorf("invalid payload: %w", err)
	}
	if p.RunID == 0 {
		return fmt.Errorf("invalid payload: missing run_id")
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	logger.L().Info("[agent-run-consumer] executing durable agent run",
		zap.String("event_id", p.EventID),
		zap.Uint64("run_id", p.RunID))
	if err := c.ai.executeDurableAgentRun(p.RunID); err != nil {
		logger.L().Warn("[agent-run-consumer] durable agent run failed",
			zap.String("event_id", p.EventID),
			zap.Uint64("run_id", p.RunID),
			zap.Error(err))
		return err
	}
	return nil
}
