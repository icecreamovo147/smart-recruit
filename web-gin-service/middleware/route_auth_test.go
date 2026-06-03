package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"web-gin-service/pkg/authz"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// syntheticPrincipalMiddleware injects a synthetic principal into the Gin context,
// simulating what JWTAuthByClient would do after validating a JWT.
func syntheticPrincipalMiddleware(roles, permissions []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", int64(1))
		c.Set("username", "test_user")
		c.Set("account_type", "staff")
		c.Set("roles", roles)
		c.Set("permissions", permissions)
		c.Set("token_version", int32(1))
		c.Next()
	}
}

// syntheticCandidatePrincipal injects a candidate principal.
func syntheticCandidatePrincipal() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", int64(2))
		c.Set("username", "candidate_user")
		c.Set("account_type", "candidate")
		c.Set("roles", []string{authz.RoleCandidate})
		c.Set("permissions", []string{
			authz.PermCandidateProfileManage,
			authz.PermCandidateResumeManage,
			authz.PermCandidateApplicationManage,
			authz.PermNotificationRead,
			authz.PermAICandidateUse,
		})
		c.Set("token_version", int32(1))
		c.Next()
	}
}

type routeAuthCase struct {
	name        string
	method      string
	path        string
	roles       []string
	permissions []string
	middleware  []gin.HandlerFunc // the middleware chain to test
	wantStatus  int
}

func runRouteAuthTest(t *testing.T, tc routeAuthCase) {
	t.Helper()
	router := gin.New()

	// Inject synthetic principal
	principal := syntheticPrincipalMiddleware(tc.roles, tc.permissions)
	router.Use(principal)

	// Add the middleware under test
	for _, m := range tc.middleware {
		router.Use(m)
	}

	// Stub handler — if middleware passes, return 200
	router.Handle(tc.method, tc.path, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(tc.method, tc.path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != tc.wantStatus {
		t.Errorf("%s: %s %s with roles=%v perms=%v → got %d, want %d",
			tc.name, tc.method, tc.path, tc.roles, tc.permissions, w.Code, tc.wantStatus)
	}
}

// ── Candidate route tests ──────────────────────────────────────────────

func TestCandidateCannotAccessHRRoutes(t *testing.T) {
	cases := []routeAuthCase{
		{
			name:   "candidate blocked from HR job list",
			method: "GET", path: "/api/v1/hr/jobs",
			roles: []string{authz.RoleCandidate}, permissions: []string{},
			middleware: []gin.HandlerFunc{RequireAnyRole(authz.StaffRoles()...)},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "candidate blocked from HR job create",
			method: "POST", path: "/api/v1/hr/jobs",
			roles: []string{authz.RoleCandidate}, permissions: []string{},
			middleware: []gin.HandlerFunc{RequireAnyRole(authz.StaffRoles()...)},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "candidate blocked from admin invite codes",
			method: "GET", path: "/api/v1/hr/admin/invite-codes",
			roles: []string{authz.RoleCandidate}, permissions: []string{},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermAdminInviteManage),
			},
			wantStatus: http.StatusForbidden,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runRouteAuthTest(t, tc) })
	}
}

func TestCandidateCanAccessOwnRoutes(t *testing.T) {
	cases := []routeAuthCase{
		{
			name:   "candidate can access profile",
			method: "GET", path: "/api/v1/candidate/profile",
			roles: []string{authz.RoleCandidate},
			permissions: []string{authz.PermCandidateProfileManage},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.RoleCandidate),
				RequirePermission(authz.PermCandidateProfileManage),
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "candidate can access own applications",
			method: "GET", path: "/api/v1/candidate/applications",
			roles: []string{authz.RoleCandidate},
			permissions: []string{authz.PermCandidateApplicationManage},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.RoleCandidate),
				RequirePermission(authz.PermCandidateApplicationManage),
			},
			wantStatus: http.StatusOK,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runRouteAuthTest(t, tc) })
	}
}

// ── Recruiter route tests ───────────────────────────────────────────────

func TestRecruiterCanAccessJobRoutes(t *testing.T) {
	cases := []routeAuthCase{
		{
			name:   "recruiter can read jobs",
			method: "GET", path: "/api/v1/hr/jobs",
			roles: []string{authz.RoleRecruiter},
			permissions: []string{authz.PermJobRead},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermJobRead),
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "recruiter can create jobs",
			method: "POST", path: "/api/v1/hr/jobs",
			roles: []string{authz.RoleRecruiter},
			permissions: []string{authz.PermJobCreate},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermJobCreate),
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "recruiter can update application status",
			method: "PATCH", path: "/api/v1/hr/applications/1/status",
			roles: []string{authz.RoleRecruiter},
			permissions: []string{authz.PermApplicationStatusUpdate},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermApplicationStatusUpdate),
			},
			wantStatus: http.StatusOK,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runRouteAuthTest(t, tc) })
	}
}

func TestRecruiterCannotAccessAdminRoutes(t *testing.T) {
	cases := []routeAuthCase{
		{
			name:   "recruiter blocked from invite codes",
			method: "GET", path: "/api/v1/hr/admin/invite-codes",
			roles: []string{authz.RoleRecruiter},
			permissions: []string{authz.PermJobRead}, // has job perms but not admin
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermAdminInviteManage),
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "recruiter blocked from role management",
			method: "GET", path: "/api/v1/hr/admin/roles",
			roles: []string{authz.RoleRecruiter},
			permissions: []string{authz.PermJobRead},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermAdminRoleManage),
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "recruiter blocked from staff user management",
			method: "GET", path: "/api/v1/hr/admin/staff-users",
			roles: []string{authz.RoleRecruiter},
			permissions: []string{authz.PermJobRead},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermAdminUserManage),
			},
			wantStatus: http.StatusForbidden,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runRouteAuthTest(t, tc) })
	}
}

// ── Recruiting Admin route tests ────────────────────────────────────────

func TestRecruitingAdminCanAccessAdminRoutes(t *testing.T) {
	cases := []routeAuthCase{
		{
			name:   "recruiting_admin can manage invite codes",
			method: "GET", path: "/api/v1/hr/admin/invite-codes",
			roles: []string{authz.RoleRecruitingAdmin},
			permissions: []string{authz.PermAdminInviteManage},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermAdminInviteManage),
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "recruiting_admin can manage roles",
			method: "GET", path: "/api/v1/hr/admin/roles",
			roles: []string{authz.RoleRecruitingAdmin},
			permissions: []string{authz.PermAdminRoleManage},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermAdminRoleManage),
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "recruiting_admin without recruiter perm cannot update application status",
			method: "PATCH", path: "/api/v1/hr/applications/1/status",
			roles: []string{authz.RoleRecruitingAdmin},
			permissions: []string{authz.PermAdminInviteManage, authz.PermAdminRoleManage}, // has admin perms but NOT app.status.update
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermApplicationStatusUpdate),
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "recruiting_admin with recruiter perm can update application status",
			method: "PATCH", path: "/api/v1/hr/applications/1/status",
			roles: []string{authz.RoleRecruitingAdmin, authz.RoleRecruiter},
			permissions: []string{authz.PermAdminInviteManage, authz.PermApplicationStatusUpdate},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermApplicationStatusUpdate),
			},
			wantStatus: http.StatusOK,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runRouteAuthTest(t, tc) })
	}
}

// ── System Admin route tests ────────────────────────────────────────────

func TestSystemAdminCanAccessAdminRoutes(t *testing.T) {
	cases := []routeAuthCase{
		{
			name:   "system_admin can manage roles",
			method: "GET", path: "/api/v1/hr/admin/roles",
			roles: []string{authz.RoleSystemAdmin},
			permissions: []string{authz.PermAdminRoleManage},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermAdminRoleManage),
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "system_admin can manage users",
			method: "GET", path: "/api/v1/hr/admin/staff-users",
			roles: []string{authz.RoleSystemAdmin},
			permissions: []string{authz.PermAdminUserManage},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermAdminUserManage),
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "system_admin can read audit logs",
			method: "GET", path: "/api/v1/hr/admin/third-party-usage-logs",
			roles: []string{authz.RoleSystemAdmin},
			permissions: []string{authz.PermAuditUsageRead},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermAuditUsageRead),
			},
			wantStatus: http.StatusOK,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runRouteAuthTest(t, tc) })
	}
}

func TestSystemAdminWithoutRecruiterPermCannotMutateRecruitingState(t *testing.T) {
	cases := []routeAuthCase{
		{
			name:   "system_admin without job.create cannot create jobs",
			method: "POST", path: "/api/v1/hr/jobs",
			roles: []string{authz.RoleSystemAdmin},
			permissions: []string{authz.PermAdminRoleManage, authz.PermAdminUserManage}, // admin only, no job.create
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermJobCreate),
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "system_admin without app.status.update cannot update status",
			method: "PATCH", path: "/api/v1/hr/applications/1/status",
			roles: []string{authz.RoleSystemAdmin},
			permissions: []string{authz.PermAdminRoleManage},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermApplicationStatusUpdate),
			},
			wantStatus: http.StatusForbidden,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runRouteAuthTest(t, tc) })
	}
}

// ── Interviewer route tests ─────────────────────────────────────────────

func TestInterviewerCannotMutateApplicationStatus(t *testing.T) {
	tc := routeAuthCase{
		name:   "interviewer blocked from updating application status",
		method: "PATCH", path: "/api/v1/hr/applications/1/status",
		roles: []string{authz.RoleInterviewer},
		permissions: []string{authz.PermInterviewRead, authz.PermInterviewFeedback},
		middleware: []gin.HandlerFunc{
			RequireAnyRole(authz.StaffRoles()...),
			RequirePermission(authz.PermApplicationStatusUpdate),
		},
		wantStatus: http.StatusForbidden,
	}
	runRouteAuthTest(t, tc)
}

func TestInterviewerCanAccessNotificationRoutes(t *testing.T) {
	tc := routeAuthCase{
		name:   "interviewer can read notifications",
		method: "GET", path: "/api/v1/hr/notifications",
		roles: []string{authz.RoleInterviewer},
		permissions: []string{authz.PermNotificationRead, authz.PermInterviewRead, authz.PermInterviewFeedback},
		middleware: []gin.HandlerFunc{
			RequireAnyRole(authz.StaffRoles()...),
			RequirePermission(authz.PermNotificationRead),
		},
		wantStatus: http.StatusOK,
	}
	runRouteAuthTest(t, tc)
}

func TestInterviewerCannotAccessRecruiterJobRoutes(t *testing.T) {
	cases := []routeAuthCase{
		{
			name:   "interviewer blocked from creating jobs",
			method: "POST", path: "/api/v1/hr/jobs",
			roles: []string{authz.RoleInterviewer},
			permissions: []string{authz.PermInterviewRead},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermJobCreate),
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "interviewer blocked from admin routes",
			method: "GET", path: "/api/v1/hr/admin/invite-codes",
			roles: []string{authz.RoleInterviewer},
			permissions: []string{authz.PermInterviewRead},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermAdminInviteManage),
			},
			wantStatus: http.StatusForbidden,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runRouteAuthTest(t, tc) })
	}
}

// ── Permission-only access tests ────────────────────────────────────────

func TestNoPermissionDeniesAccess(t *testing.T) {
	cases := []routeAuthCase{
		{
			name:   "staff role without job.read cannot access jobs",
			method: "GET", path: "/api/v1/hr/jobs",
			roles: []string{authz.RoleRecruiter},
			permissions: []string{}, // no permissions at all
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermJobRead),
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "staff with wrong perm cannot access",
			method: "POST", path: "/api/v1/hr/jobs",
			roles: []string{authz.RoleRecruiter},
			permissions: []string{authz.PermJobRead}, // has job.read but NOT job.create
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequirePermission(authz.PermJobCreate),
			},
			wantStatus: http.StatusForbidden,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runRouteAuthTest(t, tc) })
	}
}

// ── RequireAnyPermission tests ──────────────────────────────────────────

func TestRequireAnyPermission_AllowsEither(t *testing.T) {
	// Dashboard endpoint: requires job.read OR application.read
	cases := []routeAuthCase{
		{
			name:   "dashboard accessible with job.read",
			method: "GET", path: "/api/v1/hr/dashboard/summary",
			roles: []string{authz.RoleRecruiter},
			permissions: []string{authz.PermJobRead},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequireAnyPermission(authz.PermJobRead, authz.PermApplicationRead),
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "dashboard accessible with application.read",
			method: "GET", path: "/api/v1/hr/dashboard/summary",
			roles: []string{authz.RoleRecruiter},
			permissions: []string{authz.PermApplicationRead},
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequireAnyPermission(authz.PermJobRead, authz.PermApplicationRead),
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "dashboard denied without either perm",
			method: "GET", path: "/api/v1/hr/dashboard/summary",
			roles: []string{authz.RoleRecruiter},
			permissions: []string{authz.PermNotificationRead}, // unrelated
			middleware: []gin.HandlerFunc{
				RequireAnyRole(authz.StaffRoles()...),
				RequireAnyPermission(authz.PermJobRead, authz.PermApplicationRead),
			},
			wantStatus: http.StatusForbidden,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runRouteAuthTest(t, tc) })
	}
}

// ── RequireRoleByKey tests ──────────────────────────────────────────────

func TestRequireRoleByKey_ExactMatch(t *testing.T) {
	cases := []routeAuthCase{
		{
			name:   "exact recruiter role match passes",
			method: "GET", path: "/test",
			roles: []string{authz.RoleRecruiter},
			permissions: []string{},
			middleware: []gin.HandlerFunc{RequireRoleByKey(authz.RoleRecruiter)},
			wantStatus: http.StatusOK,
		},
		{
			name:   "wrong role fails",
			method: "GET", path: "/test",
			roles: []string{authz.RoleCandidate},
			permissions: []string{},
			middleware: []gin.HandlerFunc{RequireRoleByKey(authz.RoleRecruiter)},
			wantStatus: http.StatusForbidden,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runRouteAuthTest(t, tc) })
	}
}

func TestRequireAnyRole_MultipleRoles(t *testing.T) {
	cases := []routeAuthCase{
		{
			name:   "recruiter passes staff gate",
			method: "GET", path: "/api/v1/hr/test",
			roles: []string{authz.RoleRecruiter},
			permissions: []string{},
			middleware: []gin.HandlerFunc{RequireAnyRole(authz.StaffRoles()...)},
			wantStatus: http.StatusOK,
		},
		{
			name:   "interviewer passes staff gate",
			method: "GET", path: "/api/v1/hr/test",
			roles: []string{authz.RoleInterviewer},
			permissions: []string{},
			middleware: []gin.HandlerFunc{RequireAnyRole(authz.StaffRoles()...)},
			wantStatus: http.StatusOK,
		},
		{
			name:   "candidate fails staff gate",
			method: "GET", path: "/api/v1/hr/test",
			roles: []string{authz.RoleCandidate},
			permissions: []string{},
			middleware: []gin.HandlerFunc{RequireAnyRole(authz.StaffRoles()...)},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "no roles fails staff gate",
			method: "GET", path: "/api/v1/hr/test",
			roles: []string{},
			permissions: []string{},
			middleware: []gin.HandlerFunc{RequireAnyRole(authz.StaffRoles()...)},
			wantStatus: http.StatusForbidden,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runRouteAuthTest(t, tc) })
	}
}

// ── HasPermission / HasAnyPermission context helpers ────────────────────

func TestHasPermission_ContextHelper(t *testing.T) {
	router := gin.New()
	router.Use(syntheticPrincipalMiddleware(
		[]string{authz.RoleRecruiter},
		[]string{authz.PermJobRead, authz.PermApplicationRead},
	))
	router.GET("/test", func(c *gin.Context) {
		if !HasPermission(c, authz.PermJobRead) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HasPermission failed, got %d", w.Code)
	}
}

func TestHasAnyPermission_ContextHelper(t *testing.T) {
	router := gin.New()
	router.Use(syntheticPrincipalMiddleware(
		[]string{authz.RoleRecruiter},
		[]string{authz.PermJobRead},
	))
	router.GET("/test", func(c *gin.Context) {
		if !HasAnyPermission(c, authz.PermJobRead, authz.PermApplicationRead) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HasAnyPermission failed, got %d", w.Code)
	}
}

func TestHasRole_ContextHelper(t *testing.T) {
	router := gin.New()
	router.Use(syntheticPrincipalMiddleware(
		[]string{authz.RoleRecruiter, authz.RoleRecruitingAdmin},
		[]string{},
	))
	router.GET("/test", func(c *gin.Context) {
		if !HasRole(c, authz.RoleRecruitingAdmin) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HasRole failed, got %d", w.Code)
	}
}
