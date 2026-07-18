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
		Command:      "smart-recruit-gateway",
		Image:        "recruitment/smart-recruit-gateway",
		ConfigPrefix: "WEB_",
		Health:       "http readiness on gateway health endpoint",
		Cutover:      "active public entrypoint",
	},
	{
		Name:         "smart-recruit-commons",
		Role:         RoleService,
		Command:      "smart-recruit-commons",
		Image:        "recruitment/smart-recruit-commons",
		ConfigPrefix: "LOGIC_",
		Health:       "gRPC health on port 50051",
		Cutover:      "shared domain library; not deployed as a network service",
	},
	{
		Name:         "worker-service",
		Role:         RoleWorker,
		Command:      "smart-recruit-worker-service",
		Image:        "recruitment/smart-recruit-commons",
		ConfigPrefix: "WORKER_",
		Health:       "process liveness until dedicated worker probes exist",
		Cutover:      "dedicated worker service runtime",
	},
	{
		Name:         "identity-service",
		Role:         RoleService,
		Command:      "smart-recruit-identity-service/cmd/identity-service",
		Image:        "recruitment/identity-service",
		ConfigPrefix: "IDENTITY_",
		Health:       "gRPC health before traffic routing",
		Cutover:      "routed microservice runtime",
	},
	{
		Name:         "recruitment-service",
		Role:         RoleService,
		Command:      "smart-recruit-recruitment-service/cmd/recruitment-service",
		Image:        "recruitment/recruitment-service",
		ConfigPrefix: "RECRUITMENT_",
		Health:       "gRPC health before traffic routing",
		Cutover:      "routed microservice runtime",
	},
	{
		Name:         "interview-service",
		Role:         RoleService,
		Command:      "smart-recruit-interview-service/cmd/interview-service",
		Image:        "recruitment/interview-service",
		ConfigPrefix: "INTERVIEW_",
		Health:       "gRPC health before traffic routing",
		Cutover:      "routed microservice runtime",
	},
	{
		Name:         "offer-service",
		Role:         RoleService,
		Command:      "smart-recruit-offer-service/cmd/offer-service",
		Image:        "recruitment/offer-service",
		ConfigPrefix: "OFFER_",
		Health:       "gRPC health before traffic routing",
		Cutover:      "routed microservice runtime",
	},
	{
		Name:         "notification-service",
		Role:         RoleService,
		Command:      "smart-recruit-notification-service/cmd/notification-service",
		Image:        "recruitment/notification-service",
		ConfigPrefix: "NOTIFICATION_",
		Health:       "gRPC health before traffic routing",
		Cutover:      "routed microservice runtime",
	},
	{
		Name:         "ai-agent-service",
		Role:         RoleService,
		Command:      "smart-recruit-ai-agent-service/cmd/ai-agent-service",
		Image:        "recruitment/ai-agent-service",
		ConfigPrefix: "AI_AGENT_",
		Health:       "gRPC health plus dependency degradation status before traffic routing",
		Cutover:      "routed microservice runtime",
	},
	{
		Name:         "analytics-service",
		Role:         RoleService,
		Command:      "smart-recruit-analytics-service/cmd/analytics-service",
		Image:        "recruitment/analytics-service",
		ConfigPrefix: "ANALYTICS_",
		Health:       "gRPC health plus projection lag readiness before traffic routing",
		Cutover:      "routed microservice runtime; reads Analytics-owned projections",
	},
	{
		Name:         "worker-services",
		Role:         RoleWorker,
		Command:      "smart-recruit-worker-service/cmd/worker-service",
		Image:        "recruitment/worker-services",
		ConfigPrefix: "WORKER_",
		Health:       "process liveness plus queue backlog and dead-letter diagnostics",
		Cutover:      "dedicated worker microservice runtime",
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
