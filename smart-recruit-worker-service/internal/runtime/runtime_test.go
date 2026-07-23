package runtime

import (
	"context"
	"reflect"
	"testing"
)

func TestParseWorkloadConfigDefaultsAndDisables(t *testing.T) {
	cfg, err := ParseWorkloadConfig("", "email-consumer,analytics-projection-consumer")
	if err != nil {
		t.Fatalf("ParseWorkloadConfig returned error: %v", err)
	}
	enabled := cfg.EnabledSet()
	if _, ok := enabled["outbox-dispatcher"]; !ok {
		t.Fatal("default config should enable outbox-dispatcher")
	}
	if _, ok := enabled["resume-parse-consumer"]; !ok {
		t.Fatal("default config should enable resume-parse-consumer")
	}
	if len(enabled) != 2 {
		t.Fatalf("default config enabled = %#v, want outbox-dispatcher and resume-parse-consumer", cfg.Enabled)
	}
	if _, ok := enabled["email-consumer"]; ok {
		t.Fatal("disabled email-consumer should not be enabled")
	}
	if _, ok := enabled["analytics-projection-consumer"]; ok {
		t.Fatal("disabled analytics-projection-consumer should not be enabled")
	}
}

func TestRuntimeStartsEnabledWorkloadsInDescriptorOrder(t *testing.T) {
	cfg, err := ParseWorkloadConfig("agent-run-consumer,outbox-dispatcher,embedding-consumer", "")
	if err != nil {
		t.Fatalf("ParseWorkloadConfig returned error: %v", err)
	}
	var started []string
	runtime, err := New(Deps{
		Config:   cfg,
		Starters: recordingStarters(cfg.Enabled, &started),
		Status:   func(context.Context) DependencyStatus { return DependencyStatus{RabbitMQ: true, MySQL: true} },
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	want := []string{"outbox-dispatcher", "embedding-consumer", "agent-run-consumer"}
	if !reflect.DeepEqual(started, want) {
		t.Fatalf("started workloads = %#v, want %#v", started, want)
	}
	profiles := runtime.EnabledProfiles()
	if len(profiles) != len(want) || profiles[0].Name != "outbox-dispatcher" || profiles[0].Contract.OwnerContext == "" {
		t.Fatalf("enabled profiles = %#v", profiles)
	}
}

func TestRuntimeReadinessRequiresRabbitMQAndMySQL(t *testing.T) {
	cfg, err := ParseWorkloadConfig("notification-consumer", "")
	if err != nil {
		t.Fatalf("ParseWorkloadConfig returned error: %v", err)
	}
	runtime, err := New(Deps{
		Config:   cfg,
		Starters: recordingStarters(cfg.Enabled, nil),
		Status:   func(context.Context) DependencyStatus { return DependencyStatus{RabbitMQ: false, MySQL: true} },
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := runtime.Ready(context.Background()); err == nil {
		t.Fatal("Ready accepted missing RabbitMQ")
	}
	runtime, err = New(Deps{
		Config:   cfg,
		Starters: recordingStarters(cfg.Enabled, nil),
		Status:   func(context.Context) DependencyStatus { return DependencyStatus{RabbitMQ: true, MySQL: false} },
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := runtime.Ready(context.Background()); err == nil {
		t.Fatal("Ready accepted missing MySQL")
	}
}

func TestRuntimeRejectsUnknownMissingStarterAndDuplicateUnsafeWork(t *testing.T) {
	if _, err := ParseWorkloadConfig("unknown-worker", ""); err == nil {
		t.Fatal("ParseWorkloadConfig accepted unknown worker")
	}
	cfg, err := ParseWorkloadConfig("resume-parse-consumer", "")
	if err != nil {
		t.Fatalf("ParseWorkloadConfig returned error: %v", err)
	}
	if _, err := New(Deps{
		Config:   cfg,
		Starters: map[string]Starter{},
		Status:   func(context.Context) DependencyStatus { return DependencyStatus{RabbitMQ: true, MySQL: true} },
	}); err == nil {
		t.Fatal("New accepted missing starter")
	}
}

func TestRuntimeGracefulStopCancelsWorkloadContextAndStopsInReverseOrder(t *testing.T) {
	cfg, err := ParseWorkloadConfig("outbox-dispatcher,notification-consumer", "")
	if err != nil {
		t.Fatalf("ParseWorkloadConfig returned error: %v", err)
	}
	var stopped []string
	starters := map[string]Starter{
		"outbox-dispatcher":     &blockingStarter{name: "outbox-dispatcher", stopped: &stopped},
		"notification-consumer": &blockingStarter{name: "notification-consumer", stopped: &stopped},
	}
	runtime, err := New(Deps{
		Config:   cfg,
		Starters: starters,
		Status:   func(context.Context) DependencyStatus { return DependencyStatus{RabbitMQ: true, MySQL: true} },
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if err := runtime.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	want := []string{"notification-consumer", "outbox-dispatcher"}
	if !reflect.DeepEqual(stopped, want) {
		t.Fatalf("stopped workloads = %#v, want %#v", stopped, want)
	}
	for name, starter := range starters {
		if !starter.(*blockingStarter).canceled {
			t.Fatalf("%s did not observe canceled context", name)
		}
	}
}

func recordingStarters(names []string, started *[]string) map[string]Starter {
	starters := make(map[string]Starter, len(names))
	for _, name := range names {
		workloadName := name
		starters[workloadName] = StarterFunc(func(context.Context) error {
			if started != nil {
				*started = append(*started, workloadName)
			}
			return nil
		})
	}
	return starters
}

type blockingStarter struct {
	name     string
	stopped  *[]string
	cancelCh chan struct{}
	canceled bool
}

func (s *blockingStarter) Start(ctx context.Context) error {
	s.cancelCh = make(chan struct{})
	go func() {
		<-ctx.Done()
		s.canceled = true
		close(s.cancelCh)
	}()
	return nil
}

func (s *blockingStarter) Stop(context.Context) error {
	<-s.cancelCh
	*s.stopped = append(*s.stopped, s.name)
	return nil
}
