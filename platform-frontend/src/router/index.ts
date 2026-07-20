import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { PLATFORM_PERMISSIONS } from '@/permissions'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/dashboard' },
    { path: '/login', component: () => import('@/views/LoginView.vue'), meta: { title: '登录' } },
    {
      path: '/dashboard',
      component: () => import('@/views/DashboardView.vue'),
      meta: { requiresAuth: true, requiresPermission: PLATFORM_PERMISSIONS.DASHBOARD_READ, title: '运营总览' },
    },
    {
      path: '/tenants',
      component: () => import('@/views/TenantListView.vue'),
      meta: { requiresAuth: true, requiresPermission: PLATFORM_PERMISSIONS.TENANT_READ, title: '租户管理' },
    },
    {
      path: '/tenants/:tenantId',
      component: () => import('@/views/TenantDetailView.vue'),
      meta: { requiresAuth: true, requiresPermission: PLATFORM_PERMISSIONS.TENANT_READ, title: '租户详情' },
    },
    {
      path: '/audit-logs',
      component: () => import('@/views/AuditLogView.vue'),
      meta: { requiresAuth: true, requiresPermission: PLATFORM_PERMISSIONS.AUDIT_READ, title: '审计日志' },
    },
    {
      path: '/plans',
      component: () => import('@/views/PlanCatalogView.vue'),
      meta: { requiresAuth: true, requiresPermission: PLATFORM_PERMISSIONS.PLAN_READ, title: '套餐与权益' },
    },
    {
      path: '/quota-alerts',
      component: () => import('@/views/QuotaAlertView.vue'),
      meta: { requiresAuth: true, requiresPermission: PLATFORM_PERMISSIONS.ALERT_READ, title: '配额告警' },
    },
    {
      path: '/platform-users',
      component: () => import('@/views/PlatformUserView.vue'),
      meta: { requiresAuth: true, requiresPermission: PLATFORM_PERMISSIONS.USER_MANAGE, title: '平台账号' },
    },
    { path: '/ai', redirect: '/ai/capabilities' },
    { path: '/ai/capabilities', component: () => import('@/views/ai/AICapabilityReleaseView.vue'), meta: { requiresAuth: true, requiresPermission: PLATFORM_PERMISSIONS.AI_RELEASE_READ, title: 'AI 能力发布' } },
    { path: '/ai/llm', redirect: '/ai/llm/providers' },
    { path: '/ai/llm/providers', component: () => import('@/views/ai/LlmProviderConfigView.vue'), meta: { requiresAuth: true, requiresPermission: PLATFORM_PERMISSIONS.AI_CONFIG_READ, title: 'LLM Provider' } },
    { path: '/ai/llm/models', component: () => import('@/views/ai/LlmModelConfigView.vue'), meta: { requiresAuth: true, requiresPermission: PLATFORM_PERMISSIONS.AI_CONFIG_READ, title: 'LLM Model' } },
    { path: '/ai/embedding', redirect: '/ai/embedding/providers' },
    { path: '/ai/embedding/providers', component: () => import('@/views/ai/EmbeddingProviderConfigView.vue'), meta: { requiresAuth: true, requiresPermission: PLATFORM_PERMISSIONS.AI_CONFIG_READ, title: 'Embedding Provider' } },
    { path: '/ai/embedding/models', component: () => import('@/views/ai/EmbeddingModelConfigView.vue'), meta: { requiresAuth: true, requiresPermission: PLATFORM_PERMISSIONS.AI_CONFIG_READ, title: 'Embedding Model' } },
    { path: '/ai/prompts', component: () => import('@/views/ai/PromptManageView.vue'), meta: { requiresAuth: true, requiresPermission: PLATFORM_PERMISSIONS.AI_CONFIG_READ, title: 'Prompt 管理' } },
    { path: '/ai/agents', component: () => import('@/views/ai/AgentManageView.vue'), meta: { requiresAuth: true, requiresPermission: PLATFORM_PERMISSIONS.AI_CONFIG_READ, title: 'Agent 管理' } },
    { path: '/ai/skills', component: () => import('@/views/ai/SkillManageView.vue'), meta: { requiresAuth: true, requiresPermission: PLATFORM_PERMISSIONS.AI_CONFIG_READ, title: '高级 Skill' } },
    { path: '/ai/agent-skills', component: () => import('@/views/ai/AgentSkillManageView.vue'), meta: { requiresAuth: true, requiresPermission: PLATFORM_PERMISSIONS.AI_CONFIG_READ, title: 'Agent Skill' } },
    { path: '/ai/mcp', component: () => import('@/views/ai/McpManageView.vue'), meta: { requiresAuth: true, requiresPermission: PLATFORM_PERMISSIONS.AI_CONFIG_READ, title: 'MCP 工具治理' } },
    { path: '/ai/semantic-retrieval', component: () => import('@/views/ai/SemanticRetrievalDebugView.vue'), meta: { requiresAuth: true, requiresPermission: PLATFORM_PERMISSIONS.AI_DIAGNOSTICS_READ, title: '语义召回诊断' } },
    { path: '/forbidden', component: () => import('@/views/ForbiddenView.vue'), meta: { requiresAuth: true, title: '无访问权限' } },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

const firstAllowedPath = (auth: ReturnType<typeof useAuthStore>) => {
  if (auth.can(PLATFORM_PERMISSIONS.DASHBOARD_READ)) return '/dashboard'
  if (auth.can(PLATFORM_PERMISSIONS.TENANT_READ)) return '/tenants'
  if (auth.can(PLATFORM_PERMISSIONS.PLAN_READ)) return '/plans'
  if (auth.can(PLATFORM_PERMISSIONS.ALERT_READ)) return '/quota-alerts'
  if (auth.can(PLATFORM_PERMISSIONS.USER_MANAGE)) return '/platform-users'
  if (auth.can(PLATFORM_PERMISSIONS.AI_RELEASE_READ)) return '/ai/capabilities'
  if (auth.can(PLATFORM_PERMISSIONS.AI_CONFIG_READ)) return '/ai/llm/providers'
  if (auth.can(PLATFORM_PERMISSIONS.AUDIT_READ)) return '/audit-logs'
  return '/forbidden'
}

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (to.path === '/login') {
    if (auth.consumeLoginRestoreSuppression()) return true
    // A public login page should not wait for a session probe when there is no
    // cached platform identity. Cached identities are still verified before we
    // redirect so an expired session cannot flash the authenticated shell.
    if (!auth.isLoggedIn) return true
    if (await auth.restore()) return firstAllowedPath(auth)
    return true
  }
  if (!to.meta.requiresAuth) return true
  if (!auth.isLoggedIn && !(await auth.restore())) return { path: '/login', query: { redirect: to.fullPath } }
  const permission = String(to.meta.requiresPermission || '')
  if (permission && !auth.can(permission)) return to.path === '/forbidden' ? true : '/forbidden'
  if (to.path === '/dashboard' && !permission) return firstAllowedPath(auth)
  return true
})

router.afterEach((to) => {
  document.title = `${String(to.meta.title || '平台控制台')} · Smart Recruit`
})

export default router
