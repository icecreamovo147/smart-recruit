package domain

import "testing"

func TestCurrentJobStatusValuesRemainCompatible(t *testing.T) {
	t.Parallel()

	if JobStatusOffline != 0 {
		t.Fatalf("offline status drifted: %d", JobStatusOffline)
	}
	if JobStatusOnline != 1 {
		t.Fatalf("online status drifted: %d", JobStatusOnline)
	}
}

func TestJobScopeOrderingMatchesCurrentServiceSemantics(t *testing.T) {
	t.Parallel()

	if !(JobScopeDenied < JobScopeOwned &&
		JobScopeOwned < JobScopeDepartmentOrLocation &&
		JobScopeDepartmentOrLocation < JobScopeFull) {
		t.Fatal("job scope ordering drifted")
	}
}
