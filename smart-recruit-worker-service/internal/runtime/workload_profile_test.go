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
	if !reflect.DeepEqual(cfg.Enabled, WorkloadProfileNames()) {
		t.Fatalf("default enabled workloads = %#v, profiles = %#v", cfg.Enabled, WorkloadProfileNames())
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
		if !profile.Toggle.DefaultOn || profile.Toggle.EnableEnv != "WORKER_WORKLOADS" || profile.Toggle.DisableEnv != "WORKER_DISABLED_WORKLOADS" {
			t.Fatalf("unexpected toggle for %s: %+v", name, profile.Toggle)
		}
		if profile.Contract.OwnerContext == "" || profile.Contract.DLQ == "" || profile.Contract.Writes == "" {
			t.Fatalf("incomplete owner contract for %s: %+v", name, profile.Contract)
		}
	}
}
