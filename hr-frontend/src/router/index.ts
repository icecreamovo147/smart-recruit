import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { getUser } from '@/utils/token'
import { useAuthStore } from '@/stores/auth'
import { ROLE_HR, ROLE_HR_ADMIN } from '@/types/domain'
import LoginView from '@/views/LoginView.vue'
import RegisterView from '@/views/RegisterView.vue'
import WorkbenchView from '@/views/hr/WorkbenchView.vue'
import JobManageView from '@/views/hr/JobManageView.vue'
import ApplicationListView from '@/views/hr/ApplicationListView.vue'
import AIChatView from '@/views/hr/AIChatView.vue'
import ProfileView from '@/views/hr/ProfileView.vue'
import InviteCodeManageView from '@/views/hr/InviteCodeManageView.vue'
import DepartmentManageView from '@/views/hr/DepartmentManageView.vue'
import LocationManageView from '@/views/hr/LocationManageView.vue'
import UsageAuditView from '@/views/hr/UsageAuditView.vue'
import ForbiddenView from '@/views/ForbiddenView.vue'

const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/hr/workbench' },
  { path: '/login', component: LoginView },
  { path: '/register', component: RegisterView },
  { path: '/403', component: ForbiddenView },
  { path: '/hr/workbench', component: WorkbenchView, meta: { requiresAuth: true, requiresHR: true, title: '工作台' } },
  { path: '/hr/jobs', component: JobManageView, meta: { requiresAuth: true, requiresHR: true, title: '岗位管理' } },
  { path: '/hr/jobs/:jobId/applications', component: ApplicationListView, meta: { requiresAuth: true, requiresHR: true, title: '候选人台账' } },
  { path: '/hr/ai', component: AIChatView, meta: { requiresAuth: true, requiresHR: true, title: 'AI 数据助手' } },
  { path: '/hr/profile', component: ProfileView, meta: { requiresAuth: true, requiresHR: true, title: '个人信息' } },
  { path: '/hr/admin/invite-codes', component: InviteCodeManageView, meta: { requiresAuth: true, requiresHR: true, requiresAdmin: true, title: '邀请码管理' } },
  { path: '/hr/admin/job-taxonomy', redirect: '/hr/admin/departments' },
  { path: '/hr/admin/departments', component: DepartmentManageView, meta: { requiresAuth: true, requiresHR: true, requiresAdmin: true, title: '部门管理' } },
  { path: '/hr/admin/locations', component: LocationManageView, meta: { requiresAuth: true, requiresHR: true, requiresAdmin: true, title: '地点管理' } },
  { path: '/hr/admin/usage-audit', component: UsageAuditView, meta: { requiresAuth: true, requiresHR: true, requiresAdmin: true, title: '第三方服务审计' } },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to, _from, next) => {
  let user = getUser()
  if (to.meta.requiresAuth && !user) {
    // Try restoring session from httpOnly cookie before rejecting.
    const auth = useAuthStore()
    await auth.restoreSession()
    user = getUser()
  }
  if (to.meta.requiresAuth && !user) {
    next('/login')
    return
  }
  if (to.meta.requiresHR && (!user?.role || (user.role !== ROLE_HR && user.role !== ROLE_HR_ADMIN))) {
    next('/403')
    return
  }
  if (to.meta.requiresAdmin && user?.role !== ROLE_HR_ADMIN) {
    next('/403')
    return
  }
  next()
})

export default router
