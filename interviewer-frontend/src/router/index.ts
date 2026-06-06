import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const LoginView = () => import('@/views/LoginView.vue')
const DashboardView = () => import('@/views/DashboardView.vue')
const InterviewListView = () => import('@/views/InterviewListView.vue')
const InterviewDetailView = () => import('@/views/InterviewDetailView.vue')
const NotificationView = () => import('@/views/NotificationView.vue')
const ProfileView = () => import('@/views/ProfileView.vue')
const ForbiddenView = () => import('@/views/ForbiddenView.vue')
const NotFoundView = () => import('@/views/NotFoundView.vue')

const routes: RouteRecordRaw[] = [
  { path: '/login', component: LoginView, meta: { title: '登录' } },
  { path: '/', redirect: '/dashboard' },
  { path: '/dashboard', component: DashboardView, meta: { requiresAuth: true, title: '工作台' } },
  { path: '/interviews', component: InterviewListView, meta: { requiresAuth: true, title: '我的面试' } },
  { path: '/interviews/:interviewId', component: InterviewDetailView, meta: { requiresAuth: true, title: '面试详情' } },
  { path: '/notifications', component: NotificationView, meta: { requiresAuth: true, title: '通知' } },
  { path: '/profile', component: ProfileView, meta: { requiresAuth: true, title: '个人信息' } },
  { path: '/403', component: ForbiddenView, meta: { title: '无权限' } },
  { path: '/:pathMatch(.*)*', component: NotFoundView, meta: { title: '页面不存在' } },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

let sessionRestoring: Promise<boolean> | null = null

router.beforeEach(async (to) => {
  document.title = `${to.meta.title || '面试官工作台'} - 面试官工作台`

  if (!to.meta.requiresAuth) return true

  const auth = useAuthStore()

  if (!auth.isLoggedIn) {
    if (!sessionRestoring) {
      sessionRestoring = auth.restoreSession()
    }
    const restored = await sessionRestoring
    sessionRestoring = null
    if (!restored || !auth.isLoggedIn) {
      return { path: '/login', query: { redirect: to.fullPath } }
    }
  }

  // Verify interviewer role
  if (!auth.isInterviewer) {
    await auth.logoutApi()
    auth.logout()
    return { path: '/login', query: { error: 'not_interviewer' } }
  }

  return true
})

export default router
