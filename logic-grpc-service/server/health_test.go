package server

import (
	"context"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
)

type fakeSQLPinger struct {
	err error
}

func (f fakeSQLPinger) PingContext(context.Context) error {
	return f.err
}

type fakeRedisPinger struct {
	err error
}

func (f fakeRedisPinger) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	cmd.SetErr(f.err)
	return cmd
}

type fakeMQState struct {
	closed bool
	ready  bool
}

func (f fakeMQState) IsClosed() bool { return f.closed }
func (f fakeMQState) IsReady() bool  { return f.ready }

func TestEvaluateReadinessServiceDegradesOnRabbitMQ(t *testing.T) {
	report := EvaluateReadiness(context.Background(), ReadinessOptions{
		Mode:  ReadinessModeService,
		DB:    fakeSQLPinger{},
		Redis: fakeRedisPinger{},
		MQ:    fakeMQState{ready: false},
	})
	if report.Status != "degraded" {
		t.Fatalf("Status = %q, want degraded; report=%+v", report.Status, report)
	}
	if got := dependency(report, "rabbitmq").Severity; got != "soft" {
		t.Fatalf("rabbitmq severity = %q, want soft", got)
	}
}

func TestEvaluateReadinessWorkerFailsOnRabbitMQ(t *testing.T) {
	report := EvaluateReadiness(context.Background(), ReadinessOptions{
		Mode:  ReadinessModeWorker,
		DB:    fakeSQLPinger{},
		Redis: fakeRedisPinger{},
		MQ:    fakeMQState{ready: false},
	})
	if report.Status != "not_ready" {
		t.Fatalf("Status = %q, want not_ready; report=%+v", report.Status, report)
	}
	if got := dependency(report, "rabbitmq").Severity; got != "hard" {
		t.Fatalf("rabbitmq severity = %q, want hard", got)
	}
}

func TestEvaluateReadinessFailsOnHardDependency(t *testing.T) {
	report := EvaluateReadiness(context.Background(), ReadinessOptions{
		Mode:  ReadinessModeService,
		DB:    fakeSQLPinger{err: errors.New("db down")},
		Redis: fakeRedisPinger{},
		MQ:    fakeMQState{ready: true},
	})
	if report.Status != "not_ready" {
		t.Fatalf("Status = %q, want not_ready; report=%+v", report.Status, report)
	}
	if got := dependency(report, "mysql").Status; got != DependencyStatusFailed {
		t.Fatalf("mysql status = %q, want failed", got)
	}
}

func dependency(report ReadinessReport, name string) DependencyReport {
	for _, dep := range report.Dependencies {
		if dep.Name == name {
			return dep
		}
	}
	return DependencyReport{}
}
