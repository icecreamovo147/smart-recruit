import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { getUser } from '@/utils/token'
import { useAuthStore } from '@/stores/auth'
const ROLE_CANDIDATE = 1

const LoginView = () => import('@/views/LoginView.vue')
const RegisterView = () => import('@/views/RegisterView.vue')
const JobListView = () => import('@/views/candidate/JobListView.vue')
const JobDetailView = () => import('@/views/candidate/JobDetailView.vue')
const ProfileView = () => import('@/views/candidate/ProfileView.vue')
const ResumeUploadView = () => import('@/views/candidate/ResumeUploadView.vue')
const JobProgressView = () => import('@/views/candidate/JobProgressView.vue')
const ForbiddenView = () => import('@/views/ForbiddenView.vue')

const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/jobs' },
  { path: '/login', component: LoginView },
  { path: '/register', component: RegisterView },
  { path: '/403', component: ForbiddenView },
  { path: '/jobs', component: JobListView },
  { path: '/jobs/:jobId', component: JobDetailView },
  { path: '/profile', component: ProfileView, meta: { requiresAuth: true, requiresCandidate: true } },
  { path: '/resume', component: ResumeUploadView, meta: { requiresAuth: true, requiresCandidate: true } },
  { path: '/progress', component: JobProgressView, meta: { requiresAuth: true, requiresCandidate: true } },
  // Legacy routes kept for notification deep-links – redirect to /progress with query preserved
  { path: '/applications', redirect: (to) => ({ path: '/progress', query: { ...to.query, tab: 'applications' } }) },
  { path: '/interviews', redirect: (to) => ({ path: '/progress', query: { ...to.query, tab: 'interviews' } }) },
  { path: '/offers', redirect: (to) => ({ path: '/progress', query: { ...to.query, tab: 'offers' } }) },
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
  if (to.meta.requiresCandidate && user?.role !== ROLE_CANDIDATE) {
    next('/403')
    return
  }
  next()
})

export default router
