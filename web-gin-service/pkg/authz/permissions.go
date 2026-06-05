package authz

// ── Permission keys ────────────────────────────────────────────────────

const (
	// Auth / session
	PermAuthSessionRead = "auth.session.read"

	// Candidate self-service
	PermCandidateProfileManage     = "candidate.profile.manage"
	PermCandidateResumeManage      = "candidate.resume.manage"
	PermCandidateApplicationManage = "candidate.application.manage"

	// Job management
	PermJobRead    = "job.read"
	PermJobCreate  = "job.create"
	PermJobUpdate  = "job.update"
	PermJobPublish = "job.publish"

	// Application management
	PermApplicationRead        = "application.read"
	PermApplicationStatusUpdate = "application.status.update"

	// Interview management
	PermInterviewRead     = "interview.read"
	PermInterviewSchedule = "interview.schedule"
	PermInterviewFeedback  = "interview.feedback.submit"

	// Notifications
	PermNotificationRead = "notification.read"

	// AI assistants
	PermAIHRUse       = "ai.hr.use"
	PermAICandidateUse = "ai.candidate.use"

	// Admin — recruiting
	PermAdminInviteManage     = "admin.invite.manage"
	PermAdminDepartmentManage = "admin.department.manage"
	PermAdminLocationManage   = "admin.location.manage"
	PermAdminUserManage       = "admin.user.manage"
	PermAdminRoleManage       = "admin.role.manage"

	// Audit
	PermAuditUsageRead    = "audit.usage.read"
	PermAuditSecurityRead = "audit.security.read"

	// System
	PermSystemConfigManage = "system.config.manage"

	// Offer management
	PermOfferRead    = "offer.read"
	PermOfferManage  = "offer.manage"
	PermOfferSend    = "offer.send"
	PermOfferDecisionManage = "offer.decision.manage"

	// Phase 4: Candidate Collaboration
	PermCollaborationNoteRead   = "collaboration.note.read"
	PermCollaborationNoteCreate = "collaboration.note.create"
	PermCollaborationTagManage  = "collaboration.tag.manage"
	PermCollaborationTaskManage = "collaboration.task.manage"
)
