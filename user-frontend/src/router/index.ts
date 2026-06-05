import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { getUser } from '@/utils/token'
import { useAuthStore } from '@/stores/auth'
const ROLE_CANDIDATE = 1
import LoginView from '@/views/LoginView.vue'
import RegisterView from '@/views/RegisterView.vue'
import JobListView from '@/views/candidate/JobListView.vue'
import JobDetailView from '@/views/candidate/JobDetailView.vue'
import ProfileView from '@/views/candidate/ProfileView.vue'
import ResumeUploadView from '@/views/candidate/ResumeUploadView.vue'
import MyApplicationsView from '@/views/candidate/MyApplicationsView.vue'
import MyInterviewsView from '@/views/candidate/MyInterviewsView.vue'
import MyOffersView from '@/views/candidate/MyOffersView.vue'
import ForbiddenView from '@/views/ForbiddenView.vue'

const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/jobs' },
  { path: '/login', component: LoginView },
  { path: '/register', component: RegisterView },
  { path: '/403', component: ForbiddenView },
  { path: '/jobs', component: JobListView },
  { path: '/jobs/:jobId', component: JobDetailView },
  { path: '/profile', component: ProfileView, meta: { requiresAuth: true, requiresCandidate: true } },
  { path: '/resume', component: ResumeUploadView, meta: { requiresAuth: true, requiresCandidate: true } },
  { path: '/applications', component: MyApplicationsView, meta: { requiresAuth: true, requiresCandidate: true } },
  { path: '/interviews', component: MyInterviewsView, meta: { requiresAuth: true, requiresCandidate: true } },
  { path: '/offers', component: MyOffersView, meta: { requiresAuth: true, requiresCandidate: true } },
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
