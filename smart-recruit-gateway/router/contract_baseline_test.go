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
		"GET /api/v1/hr/analytics/dashboard",
		"GET /api/v1/hr/admin/roles",
		"GET /api/v1/hr/admin/permissions",
		"GET /api/v1/hr/admin/llm-providers",
		"GET /api/v1/hr/admin/mcp-servers",
		"GET /api/v1/hr/admin/skills",
	}
	for _, route := range expected {
		if !registered[route] {
			t.Fatalf("expected core route to be registered: %s", route)
		}
	}
}
