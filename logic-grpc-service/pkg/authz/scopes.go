package authz

// ── Data scope keys ────────────────────────────────────────────────────
//
// Scopes constrain permission usage by limiting which resources a user can
// access. A permission + scope combination determines effective access.
// Scope evaluation combines permission plus resource relationship.

const (
	ScopeSelf               = "self"                // own user data only
	ScopeAssignedInterviews = "assigned_interviews" // interviews explicitly assigned to the user
	ScopeOwnJobs            = "own_jobs"            // jobs created by or assigned to the recruiter
	ScopeDepartment         = "department"          // jobs/applications in assigned departments
	ScopeLocation           = "location"            // jobs/applications in assigned locations
	ScopeRecruitingAll      = "recruiting_all"      // all recruiting data
	ScopeSystemAll          = "system_all"           // platform-wide administrative scope
)

// ScopeDisplayNames maps scope keys to human-readable names.
var ScopeDisplayNames = map[string]string{
	ScopeSelf:               "仅自己",
	ScopeAssignedInterviews: "被分配的面试",
	ScopeOwnJobs:            "自己负责的岗位",
	ScopeDepartment:         "指定部门",
	ScopeLocation:           "指定地点",
	ScopeRecruitingAll:      "全部招聘数据",
	ScopeSystemAll:          "全部系统数据",
}
