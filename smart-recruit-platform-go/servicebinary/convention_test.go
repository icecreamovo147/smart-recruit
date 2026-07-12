package servicebinary

import "testing"

func TestUnitsAreValid(t *testing.T) {
	if err := Validate(Units()); err != nil {
		t.Fatalf("Validate(Units()) error = %v", err)
	}
}

func TestRequiredUnitsExist(t *testing.T) {
	required := []string{
		"api-gateway",
		"smart-recruit-domain-go",
		"worker-service",
		"identity-service",
		"recruitment-service",
		"interview-service",
		"offer-service",
		"notification-service",
		"ai-agent-service",
		"analytics-service",
		"worker-services",
	}
	for _, name := range required {
		if _, ok := Find(name); !ok {
			t.Fatalf("Find(%q) returned false", name)
		}
	}
}

func TestUnitsReturnsCopy(t *testing.T) {
	copied := Units()
	copied[0].Name = "mutated"
	if unit, ok := Find("api-gateway"); !ok || unit.Name != "api-gateway" {
		t.Fatalf("Units returned mutable backing store; unit=%+v ok=%v", unit, ok)
	}
}

func TestValidateRejectsDuplicateNames(t *testing.T) {
	invalid := []Unit{
		{
			Name:         "notification-service",
			Role:         RoleService,
			Command:      "cmd/a",
			Image:        "image/a",
			ConfigPrefix: "A_",
			Health:       "health",
			Cutover:      "cutover",
		},
		{
			Name:         "notification-service",
			Role:         RoleWorker,
			Command:      "cmd/b",
			Image:        "image/b",
			ConfigPrefix: "B_",
			Health:       "health",
			Cutover:      "cutover",
		},
	}
	if err := Validate(invalid); err == nil {
		t.Fatal("Validate duplicate names returned nil error")
	}
}
