import { PERM } from '@/types/domain'

type PermissionChecker = {
  hasPermission: (perm: string) => boolean
}

const staffHomeCandidates = [
  { permission: PERM.JOB_READ, path: '/hr/workbench' },
  { permission: PERM.INTERVIEW_READ, path: '/hr/my-interviews' },
  { permission: PERM.ADMIN_DEPARTMENT_MANAGE, path: '/hr/admin/departments' },
  { permission: PERM.ADMIN_LOCATION_MANAGE, path: '/hr/admin/locations' },
  { permission: PERM.AI_HR_USE, path: '/hr/ai' },
  { permission: PERM.ADMIN_INVITE_MANAGE, path: '/hr/admin/invite-codes' },
  { permission: PERM.ADMIN_USER_MANAGE, path: '/hr/admin/staff-users' },
  { permission: PERM.SYSTEM_CONFIG_MANAGE, path: '/hr/admin/llm-config/providers' },
  { permission: PERM.AI_PROMPT_MANAGE, path: '/hr/admin/prompts' },
  { permission: PERM.AI_AGENT_MANAGE, path: '/hr/admin/agents' },
  { permission: PERM.AI_AGENT_SKILL_MANAGE, path: '/hr/admin/agent-skills' },
  { permission: PERM.AUDIT_USAGE_READ, path: '/hr/admin/usage-stats' },
  { permission: PERM.AUDIT_SECURITY_READ, path: '/hr/admin/security-audit' },
  { permission: PERM.AUTH_SESSION_READ, path: '/hr/profile' },
]

export const resolveStaffHomePath = (auth: PermissionChecker): string => {
  return staffHomeCandidates.find((item) => auth.hasPermission(item.permission))?.path || '/403'
}

export const resolveStaffHomePathFromPermissions = (permissions: string[] = []): string => {
  const permissionSet = new Set(permissions)
  return resolveStaffHomePath({ hasPermission: (permission) => permissionSet.has(permission) })
}
