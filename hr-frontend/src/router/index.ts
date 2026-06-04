import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { getUser } from '@/utils/token'
import { useAuthStore } from '@/stores/auth'
import { PERM } from '@/types/domain'
import LoginView from '@/views/LoginView.vue'
import RegisterView from '@/views/RegisterView.vue'
import WorkbenchView from '@/views/hr/WorkbenchView.vue'
import JobManageView from '@/views/hr/JobManageView.vue'
import ApplicationListView from '@/views/hr/ApplicationListView.vue'
import InterviewTaskView from '@/views/hr/InterviewTaskView.vue'
import InterviewScheduleView from '@/views/hr/InterviewScheduleView.vue'
import AIChatView from '@/views/hr/AIChatView.vue'
import ProfileView from '@/views/hr/ProfileView.vue'
import InviteCodeManageView from '@/views/hr/InviteCodeManageView.vue'
import DepartmentManageView from '@/views/hr/DepartmentManageView.vue'
import LocationManageView from '@/views/hr/LocationManageView.vue'
import UsageAuditView from '@/views/hr/UsageAuditView.vue'
import StaffUserManageView from '@/views/hr/StaffUserManageView.vue'
import ForbiddenView from '@/views/ForbiddenView.vue'

const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/hr/workbench' },
  { path: '/login', component: LoginView },
  { path: '/register', component: RegisterView },
  { path: '/403', component: ForbiddenView },

  // Workbench — accessible to any staff user with basic permissions
  {
    path: '/hr/workbench',
    component: WorkbenchView,
    meta: { requiresAuth: true, requiresPermission: PERM.JOB_READ, title: '工作台' },
  },
  // Job management — requires job.create permission
  {
    path: '/hr/jobs',
    component: JobManageView,
    meta: { requiresAuth: true, requiresPermission: PERM.JOB_READ, title: '岗位管理' },
  },
  // Application list — requires application.read permission
  {
    path: '/hr/jobs/:jobId/applications',
    component: ApplicationListView,
    meta: { requiresAuth: true, requiresPermission: PERM.APPLICATION_READ, title: '候选人台账' },
  },
  // Interview tasks — for interviewers (requires interview.read)
  {
    path: '/hr/interviews',
    component: InterviewTaskView,
    meta: { requiresAuth: true, requiresPermission: PERM.INTERVIEW_READ, title: '面试管理' },
  },
  // Schedule interview — requires interview.schedule
  {
    path: '/hr/interviews/schedule',
    component: InterviewScheduleView,
    meta: { requiresAuth: true, requiresPermission: PERM.INTERVIEW_SCHEDULE, title: '安排面试' },
  },
  // AI assistant — requires ai.hr.use permission
  {
    path: '/hr/ai',
    component: AIChatView,
    meta: { requiresAuth: true, requiresPermission: PERM.AI_HR_USE, title: 'AI 数据助手' },
  },
  // Profile — any authenticated staff user
  {
    path: '/hr/profile',
    component: ProfileView,
    meta: { requiresAuth: true, requiresPermission: PERM.AUTH_SESSION_READ, title: '个人信息' },
  },

  // ── Admin routes ────────────────────────────────────────────────────
  // Each admin route requires a specific admin permission, not role-based inheritance.
  {
    path: '/hr/admin/invite-codes',
    component: InviteCodeManageView,
    meta: { requiresAuth: true, requiresPermission: PERM.ADMIN_INVITE_MANAGE, title: '邀请码管理' },
  },
  { path: '/hr/admin/job-taxonomy', redirect: '/hr/admin/departments' },
  {
    path: '/hr/admin/departments',
    component: DepartmentManageView,
    meta: { requiresAuth: true, requiresPermission: PERM.ADMIN_DEPARTMENT_MANAGE, title: '部门管理' },
  },
  {
    path: '/hr/admin/locations',
    component: LocationManageView,
    meta: { requiresAuth: true, requiresPermission: PERM.ADMIN_LOCATION_MANAGE, title: '地点管理' },
  },
  {
    path: '/hr/admin/usage-audit',
    component: UsageAuditView,
    meta: { requiresAuth: true, requiresPermission: PERM.AUDIT_USAGE_READ, title: '第三方服务审计' },
  },
  {
    path: '/hr/admin/staff-users',
    component: StaffUserManageView,
    meta: { requiresAuth: true, requiresPermission: PERM.ADMIN_USER_MANAGE, title: '员工账号管理' },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to, _from, next) => {
  const auth = useAuthStore()
  let user = getUser()

  if (to.meta.requiresAuth && !user) {
    // Try restoring session from httpOnly cookie before rejecting.
    await auth.restoreSession()
    user = getUser()
  }
  if (to.meta.requiresAuth && !user) {
    next('/login')
    return
  }

  // Permission-based guard
  const requiredPerm = to.meta.requiresPermission as string | undefined
  if (requiredPerm && user) {
    if (!auth.hasPermission(requiredPerm)) {
      next('/403')
      return
    }
  }

  next()
})

export default router
