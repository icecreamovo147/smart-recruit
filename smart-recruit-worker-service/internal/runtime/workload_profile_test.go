package runtime

import (
	"reflect"
	"testing"
)

func TestWorkloadProfilesMatchDefaultRuntimeDescriptors(t *testing.T) {
	if err := ValidateWorkloadProfiles(); err != nil {
		t.Fatalf("ValidateWorkloadProfiles returned %v", err)
	}
	cfg, err := ParseWorkloadConfig("", "")
	if err != nil {
		t.Fatalf("ParseWorkloadConfig returned %v", err)
	}
	if !reflect.DeepEqual(cfg.Enabled, []string{"outbox-dispatcher", "resume-parse-consumer"}) {
		t.Fatalf("default enabled workloads = %#v, want outbox-dispatcher and resume-parse-consumer", cfg.Enabled)
	}
}

func TestWorkloadProfilesRecordOwnerContracts(t *testing.T) {
	for _, name := range []string{
		"outbox-dispatcher",
		"notification-consumer",
		"email-consumer",
		"resume-parse-consumer",
		"embedding-consumer",
		"agent-run-consumer",
		"analytics-projection-consumer",
	} {
		profile, ok := ProfileByName(name)
		if !ok {
			t.Fatalf("missing profile %s", name)
		}
		if profile.Toggle.EnableEnv != "WORKER_WORKLOADS" || profile.Toggle.DisableEnv != "WORKER_DISABLED_WORKLOADS" {
			t.Fatalf("unexpected toggle for %s: %+v", name, profile.Toggle)
		}
		switch name {
		case "outbox-dispatcher", "resume-parse-consumer":
			if !profile.Toggle.DefaultOn {
				t.Fatalf("%s should be default-on: %+v", name, profile.Toggle)
			}
		default:
			if profile.Toggle.DefaultOn {
				t.Fatalf("%s should stay default-off until its real starter is wired", name)
			}
		}
		if profile.Contract.OwnerContext == "" || profile.Contract.DLQ == "" || profile.Contract.Retry == "" || profile.Contract.Writes == "" {
			t.Fatalf("incomplete owner contract for %s: %+v", name, profile.Contract)
		}
	}
}

func TestWorkloadProfilesCoverReadinessIdempotencyRetryAndDLQ(t *testing.T) {
	for _, profile := range DefaultWorkloadProfiles {
		if profile.Toggle.EnableEnv != "WORKER_WORKLOADS" || profile.Toggle.DisableEnv != "WORKER_DISABLED_WORKLOADS" {
			t.Fatalf("%s has unexpected toggle envs: %+v", profile.Name, profile.Toggle)
		}
		if profile.Contract.Idempotency == "" {
			t.Fatalf("%s missing idempotency contract", profile.Name)
		}
		if profile.Contract.Retry == "" {
			t.Fatalf("%s missing retry contract", profile.Name)
		}
		if profile.Contract.DLQ == "" {
			t.Fatalf("%s missing DLQ contract", profile.Name)
		}
	}
}
