export const PLATFORM_PERMISSIONS = {
  DASHBOARD_READ: 'platform.dashboard.read',
  TENANT_READ: 'platform.tenant.read',
  TENANT_MANAGE: 'platform.tenant.manage',
  MEMBER_MANAGE: 'platform.member.manage',
  AUDIT_READ: 'platform.audit.read',
  USER_MANAGE: 'platform.user.manage',
  PLAN_READ: 'platform.plan.read',
  PLAN_MANAGE: 'platform.plan.manage',
  PLAN_PUBLISH: 'platform.plan.publish',
  BILLING_REFUND_REVIEW: 'platform.billing.refund.review',
  SUBSCRIPTION_MANAGE: 'platform.subscription.manage',
  USAGE_READ: 'platform.usage.read',
  ALERT_READ: 'platform.alert.read',
  ALERT_MANAGE: 'platform.alert.manage',
  AI_CONFIG_READ: 'platform.ai.config.read',
  AI_CONFIG_MANAGE: 'platform.ai.config.manage',
  AI_RELEASE_READ: 'platform.ai.release.read',
  AI_RELEASE_MANAGE: 'platform.ai.release.manage',
  AI_RELEASE_PUBLISH: 'platform.ai.release.publish',
  AI_DIAGNOSTICS_READ: 'platform.ai.diagnostics.read',
  AI_DIAGNOSTICS_EXECUTE: 'platform.ai.diagnostics.execute',
} as const

export type PlatformPermission = typeof PLATFORM_PERMISSIONS[keyof typeof PLATFORM_PERMISSIONS]

export const PLATFORM_ROLES = ['platform_admin', 'platform_operator', 'platform_auditor'] as const

export const roleLabel = (role: string) => ({
  platform_admin: '平台管理员',
  platform_operator: '平台运营管理员',
  platform_auditor: '平台审计员',
  recruiting_admin: '招聘管理员',
  recruiter: '招聘专员',
  interviewer: '面试官',
  system_admin: '系统管理员',
}[role] || role)
