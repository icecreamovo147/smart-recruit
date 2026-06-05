package authz

// ── Permission keys ────────────────────────────────────────────────────
//
// Every protected business capability is represented by a stable string key.
// Route handlers and service methods reference these constants — never string
// literals scattered through the codebase.

const (
	// Auth / session
	PermAuthSessionRead = "auth.session.read" // read own session

	// Candidate self-service
	PermCandidateProfileManage     = "candidate.profile.manage"     // manage own candidate profile
	PermCandidateResumeManage      = "candidate.resume.manage"      // manage own resumes
	PermCandidateApplicationManage = "candidate.application.manage" // create and read own applications

	// Job management
	PermJobRead    = "job.read"    // read staff-visible job data
	PermJobCreate  = "job.create"  // create jobs
	PermJobUpdate  = "job.update"  // edit jobs
	PermJobPublish = "job.publish" // online/offline jobs

	// Application management
	PermApplicationRead        = "application.read"         // read applications in scope
	PermApplicationStatusUpdate = "application.status.update" // update application status in scope

	// Interview management
	PermInterviewRead     = "interview.read"     // read assigned or scoped interviews
	PermInterviewSchedule = "interview.schedule" // create/update/cancel interviews
	PermInterviewFeedback  = "interview.feedback.submit" // submit own feedback

	// Notifications
	PermNotificationRead = "notification.read" // read own notifications

	// AI assistants
	PermAIHRUse       = "ai.hr.use"       // use HR AI assistant
	PermAICandidateUse = "ai.candidate.use" // use candidate AI assistant

	// Admin — recruiting
	PermAdminInviteManage     = "admin.invite.manage"      // create/list/revoke/reactivate invite codes
	PermAdminDepartmentManage = "admin.department.manage"  // manage departments and department-location relations
	PermAdminLocationManage   = "admin.location.manage"    // manage job locations
	PermAdminUserManage       = "admin.user.manage"        // create/update/disable staff users and assign staff roles
	PermAdminRoleManage       = "admin.role.manage"        // manage role catalog and permission assignments

	// Audit
	PermAuditUsageRead    = "audit.usage.read"    // read third-party/AI usage logs
	PermAuditSecurityRead = "audit.security.read" // read authorization and security audit events

	// System
	PermSystemConfigManage = "system.config.manage" // manage platform/security configuration

	// Offer management
	PermOfferRead    = "offer.read"    // view offers in scope
	PermOfferManage  = "offer.manage"  // create/update/withdraw offers
	PermOfferSend    = "offer.send"    // send offers (snapshot terms)
	PermOfferDecisionManage = "offer.decision.manage" // candidate accept/reject own offer
)

// PermissionDisplayNames maps permission keys to human-readable descriptions.
var PermissionDisplayNames = map[string]string{
	PermAuthSessionRead:              "查看会话",
	PermCandidateProfileManage:       "管理求职者个人信息",
	PermCandidateResumeManage:        "管理简历",
	PermCandidateApplicationManage:   "管理投递",
	PermJobRead:                      "查看岗位",
	PermJobCreate:                    "创建岗位",
	PermJobUpdate:                    "编辑岗位",
	PermJobPublish:                   "发布/下线岗位",
	PermApplicationRead:              "查看候选人台账",
	PermApplicationStatusUpdate:      "变更候选人状态",
	PermInterviewRead:                "查看面试",
	PermInterviewSchedule:            "安排面试",
	PermInterviewFeedback:            "提交面试反馈",
	PermNotificationRead:             "查看通知",
	PermAIHRUse:                      "使用HR AI助手",
	PermAICandidateUse:               "使用求职者AI助手",
	PermAdminInviteManage:            "管理邀请码",
	PermAdminDepartmentManage:        "管理部门",
	PermAdminLocationManage:          "管理工作地点",
	PermAdminUserManage:              "管理人员账号",
	PermAdminRoleManage:              "管理角色与权限",
	PermAuditUsageRead:               "查看第三方服务审计",
	PermAuditSecurityRead:            "查看安全审计日志",
	PermSystemConfigManage:           "管理系统配置",
	PermOfferRead:                    "查看Offer",
	PermOfferManage:                  "管理Offer",
	PermOfferSend:                    "发送Offer",
	PermOfferDecisionManage:          "决定Offer（接受/拒绝）",
}
