package servicebinary

import (
	"fmt"
	"strings"
)

type Role string

const (
	RoleGateway Role = "gateway"
	RoleService Role = "service"
	RoleWorker  Role = "worker"
)

type Unit struct {
	Name         string
	Role         Role
	Command      string
	Image        string
	ConfigPrefix string
	Health       string
	Cutover      string
}

var units = []Unit{
	{
		Name:         "api-gateway",
		Role:         RoleGateway,
		Command:      "web-gin-service",
		Image:        "recruitment/web-gin-service",
		ConfigPrefix: "WEB_",
		Health:       "http readiness on gateway health endpoint",
		Cutover:      "existing public entrypoint; no TASK-BDME-026 routing change",
	},
	{
		Name:         "logic-grpc-service",
		Role:         RoleService,
		Command:      "logic-grpc-service",
		Image:        "recruitment/logic-grpc-service",
		ConfigPrefix: "LOGIC_",
		Health:       "gRPC health on port 50051",
		Cutover:      "current monolith backend; background workers disabled in API deployment",
	},
	{
		Name:         "logic-worker",
		Role:         RoleWorker,
		Command:      "logic-grpc-service --worker-only",
		Image:        "recruitment/logic-grpc-service",
		ConfigPrefix: "WORKER_",
		Health:       "process liveness until dedicated worker probes exist",
		Cutover:      "current worker deployment using the shared logic binary",
	},
	{
		Name:         "identity-service",
		Role:         RoleService,
		Command:      "logic-grpc-service/cmd/identity-service",
		Image:        "recruitment/identity-service",
		ConfigPrefix: "IDENTITY_",
		Health:       "gRPC health before traffic routing",
		Cutover:      "future extracted binary; no traffic until identity cutover TASK",
	},
	{
		Name:         "recruitment-service",
		Role:         RoleService,
		Command:      "logic-grpc-service/cmd/recruitment-service",
		Image:        "recruitment/recruitment-service",
		ConfigPrefix: "RECRUITMENT_",
		Health:       "gRPC health before traffic routing",
		Cutover:      "future extracted binary; no traffic until recruitment cutover TASK",
	},
	{
		Name:         "interview-service",
		Role:         RoleService,
		Command:      "logic-grpc-service/cmd/interview-service",
		Image:        "recruitment/interview-service",
		ConfigPrefix: "INTERVIEW_",
		Health:       "gRPC health before traffic routing",
		Cutover:      "future extracted binary; no traffic until interview cutover TASK",
	},
	{
		Name:         "offer-service",
		Role:         RoleService,
		Command:      "logic-grpc-service/cmd/offer-service",
		Image:        "recruitment/offer-service",
		ConfigPrefix: "OFFER_",
		Health:       "gRPC health before traffic routing",
		Cutover:      "future extracted binary; no traffic until offer cutover TASK",
	},
	{
		Name:         "notification-service",
		Role:         RoleService,
		Command:      "logic-grpc-service/cmd/notification-service",
		Image:        "recruitment/notification-service",
		ConfigPrefix: "NOTIFICATION_",
		Health:       "gRPC health before traffic routing",
		Cutover:      "future extracted binary; starts as shadow or dual-run only",
	},
	{
		Name:         "ai-agent-service",
		Role:         RoleService,
		Command:      "logic-grpc-service/cmd/ai-agent-service",
		Image:        "recruitment/ai-agent-service",
		ConfigPrefix: "AI_AGENT_",
		Health:       "gRPC health plus dependency degradation status before traffic routing",
		Cutover:      "future extracted binary; starts as shadow or dual-run only",
	},
	{
		Name:         "analytics-service",
		Role:         RoleService,
		Command:      "logic-grpc-service/cmd/analytics-service",
		Image:        "recruitment/analytics-service",
		ConfigPrefix: "ANALYTICS_",
		Health:       "gRPC health plus projection lag readiness before traffic routing",
		Cutover:      "future extracted binary; reads only Analytics-owned projections",
	},
	{
		Name:         "worker-services",
		Role:         RoleWorker,
		Command:      "logic-grpc-service/cmd/worker-services",
		Image:        "recruitment/worker-services",
		ConfigPrefix: "WORKER_",
		Health:       "process liveness plus queue backlog and dead-letter diagnostics",
		Cutover:      "future worker binary family; event consumers use shadow or controlled cutover",
	},
}

func Units() []Unit {
	copied := make([]Unit, len(units))
	copy(copied, units)
	return copied
}

func Find(name string) (Unit, bool) {
	for _, unit := range units {
		if unit.Name == name {
			return unit, true
		}
	}
	return Unit{}, false
}

func Validate(units []Unit) error {
	seen := map[string]struct{}{}
	for index, unit := range units {
		if strings.TrimSpace(unit.Name) == "" {
			return fmt.Errorf("unit %d has empty name", index)
		}
		if _, exists := seen[unit.Name]; exists {
			return fmt.Errorf("duplicate unit name %q", unit.Name)
		}
		seen[unit.Name] = struct{}{}
		if unit.Role != RoleGateway && unit.Role != RoleService && unit.Role != RoleWorker {
			return fmt.Errorf("unit %q has invalid role %q", unit.Name, unit.Role)
		}
		if strings.TrimSpace(unit.Command) == "" {
			return fmt.Errorf("unit %q has empty command", unit.Name)
		}
		if strings.TrimSpace(unit.Image) == "" {
			return fmt.Errorf("unit %q has empty image", unit.Name)
		}
		if strings.TrimSpace(unit.ConfigPrefix) == "" {
			return fmt.Errorf("unit %q has empty config prefix", unit.Name)
		}
		if strings.TrimSpace(unit.Health) == "" {
			return fmt.Errorf("unit %q has empty health convention", unit.Name)
		}
		if strings.TrimSpace(unit.Cutover) == "" {
			return fmt.Errorf("unit %q has empty cutover convention", unit.Name)
		}
	}
	return nil
}
