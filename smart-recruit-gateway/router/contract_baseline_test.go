package router

import (
	"testing"

	"github.com/gin-gonic/gin"

	"smart-recruit-gateway/config"
	"smart-recruit-gateway/rpc"
)

func TestCoreHTTPRouteGroupsRemainRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine, limiters := Setup(config.Config{}, &rpc.Clients{}, nil)
	if limiters != nil {
		defer limiters.Close()
	}

	registered := make(map[string]bool)
	for _, route := range engine.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	expected := []string{
		"GET /health",
		"GET /livez",
		"GET /readyz",
		"POST /api/v1/auth/login",
		"POST /api/v1/auth/refresh",
		"GET /api/v1/auth/me",
		"GET /api/v1/platform/dashboard",
		"GET /api/v1/platform/tenants",
		"GET /api/v1/platform/tenants/:tenant_id",
		"PATCH /api/v1/platform/tenants/:tenant_id/memberships/:membership_id/status",
		"GET /api/v1/platform/audit-logs",
		"GET /api/v1/platform/plans",
		"POST /api/v1/platform/plans/:plan_id/versions",
		"POST /api/v1/platform/plans/:plan_id/versions/:version_id/publish",
		"GET /api/v1/platform/tenants/:tenant_id/subscription",
		"PUT /api/v1/platform/tenants/:tenant_id/subscription",
		"PUT /api/v1/platform/tenants/:tenant_id/entitlement-override",
		"GET /api/v1/platform/tenants/:tenant_id/usage",
		"GET /api/v1/platform/quota-alerts",
		"PATCH /api/v1/platform/quota-alerts/:alert_id",
		"GET /api/v1/platform/users",
		"POST /api/v1/platform/users",
		"PATCH /api/v1/platform/users/:user_id",
		"GET /api/v1/platform/ai/capabilities",
		"POST /api/v1/platform/ai/capabilities/:capability_id/versions",
		"DELETE /api/v1/platform/ai/capability-versions/:version_id",
		"POST /api/v1/platform/ai/capability-versions/:version_id/publish",
		"GET /api/v1/platform/ai/llm-providers",
		"GET /api/v1/platform/ai/mcp-servers",
		"GET /api/v1/jobs",
		"GET /api/v1/jobs/:job_id",
		"GET /api/v1/candidate/profile",
		"POST /api/v1/candidate/resume/presign",
		"POST /api/v1/candidate/applications",
		"GET /api/v1/candidate/notifications",
		"POST /api/v1/candidate/ai/chat",
		"POST /api/v1/candidate/ai/chat/stream",
		"GET /api/v1/candidate/ai/models",
		"GET /api/v1/hr/jobs",
		"POST /api/v1/hr/jobs",
		"PATCH /api/v1/hr/applications/:application_id/status",
		"POST /api/v1/hr/interviews",
		"POST /api/v1/hr/offers",
		"GET /api/v1/hr/notifications",
		"GET /api/v1/hr/dashboard/summary",
		"POST /api/v1/hr/ai/chat",
		"POST /api/v1/hr/ai/runs",
		"GET /api/v1/hr/ai/capabilities",
		"GET /api/v1/hr/analytics/dashboard",
		"GET /api/v1/hr/admin/roles",
		"GET /api/v1/hr/admin/permissions",
	}
	for _, route := range expected {
		if !registered[route] {
			t.Fatalf("expected core route to be registered: %s", route)
		}
	}

	removedTenantTechnicalRoutes := []string{
		"GET /api/v1/hr/admin/llm-providers",
		"POST /api/v1/hr/admin/llm-models",
		"GET /api/v1/hr/admin/embedding-providers",
		"POST /api/v1/hr/admin/embedding-models",
		"GET /api/v1/hr/admin/prompt-templates",
		"POST /api/v1/hr/admin/agent-configs",
		"GET /api/v1/hr/admin/mcp-servers",
		"POST /api/v1/hr/admin/skills",
		"GET /api/v1/hr/admin/agent-skills",
	}
	for _, route := range removedTenantTechnicalRoutes {
		if registered[route] {
			t.Fatalf("tenant technical AI route must be removed after one-time cutover: %s", route)
		}
	}

	removedLegacySkillRoutes := []string{
		"GET /api/v1/platform/ai/skills",
		"POST /api/v1/platform/ai/skills",
		"PUT /api/v1/platform/ai/skills/:id",
		"GET /api/v1/platform/ai/skills/:id/versions",
		"POST /api/v1/platform/ai/skills/:id/versions",
		"POST /api/v1/platform/ai/skills/:id/versions/:version_id/activate",
		"GET /api/v1/platform/ai/skills/:id/tools",
		"PUT /api/v1/platform/ai/skills/:id/tools/:tool_id",
		"GET /api/v1/hr/capabilities",
		"GET /api/v1/hr/capabilities/:id",
		"POST /api/v1/hr/capabilities/from-template",
		"GET /api/v1/hr/ai/skill-capabilities",
	}
	for _, route := range removedLegacySkillRoutes {
		if registered[route] {
			t.Fatalf("legacy Skill route must be absent after the registry retirement: %s", route)
		}
	}
}
