import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { getToken, getUser } from '@/utils/token'
import { ROLE_CANDIDATE } from '@/types/domain'
import LoginView from '@/views/LoginView.vue'
import RegisterView from '@/views/RegisterView.vue'
import JobListView from '@/views/candidate/JobListView.vue'
import JobDetailView from '@/views/candidate/JobDetailView.vue'
import ProfileView from '@/views/candidate/ProfileView.vue'
import ResumeUploadView from '@/views/candidate/ResumeUploadView.vue'
import MyApplicationsView from '@/views/candidate/MyApplicationsView.vue'
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
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to, _from, next) => {
  const token = getToken()
  const user = getUser()
  if (to.meta.requiresAuth && !token) {
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
