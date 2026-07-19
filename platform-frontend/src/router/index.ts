import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({ history: createWebHistory(), routes: [
  { path: '/', redirect: '/tenants' },
  { path: '/login', component: () => import('@/views/LoginView.vue') },
  { path: '/tenants', component: () => import('@/views/TenantListView.vue'), meta: { requiresAuth: true } },
  { path: '/:pathMatch(.*)*', redirect: '/tenants' },
] })

router.beforeEach(async (to) => {
  if (!to.meta.requiresAuth) return true
  const auth = useAuthStore()
  if (!auth.isLoggedIn && !(await auth.restore())) return { path: '/login', query: { redirect: to.fullPath } }
  if (!auth.user?.roles.includes('platform_admin')) return '/login'
  return true
})

export default router
