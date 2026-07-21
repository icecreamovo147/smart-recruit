<script setup lang="ts">
import { computed, ref } from 'vue'
import { ArrowDown, Bell, Connection, Cpu, DataAnalysis, DocumentChecked, Expand, Fold, Goods, MagicStick, Moon, OfficeBuilding, SetUp, Sunny, SwitchButton, Tools, UserFilled } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { useTheme } from '@/composables/useTheme'
import { PLATFORM_PERMISSIONS, roleLabel } from '@/permissions'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { isDark, isThemeTransitioning, toggleTheme } = useTheme()
const sidebarStorageKey = 'platform_console_sidebar_collapsed'
const readSidebarCollapsed = (): boolean => {
  try {
    return localStorage.getItem(sidebarStorageKey) === '1'
  } catch {
    return false
  }
}
const sidebarCollapsed = ref(readSidebarCollapsed())
const signingOut = ref(false)
const isLogin = computed(() => route.path === '/login')
const primaryRole = computed(() => roleLabel(auth.user?.roles[0] || ''))
const navigation = computed(() => [
  { path: '/dashboard', label: '运营总览', icon: DataAnalysis, permission: PLATFORM_PERMISSIONS.DASHBOARD_READ },
  { path: '/tenants', label: '租户管理', icon: OfficeBuilding, permission: PLATFORM_PERMISSIONS.TENANT_READ },
  { path: '/plans', label: '套餐与权益', icon: Goods, permission: PLATFORM_PERMISSIONS.PLAN_READ },
  { path: '/quota-alerts', label: '配额告警', icon: Bell, permission: PLATFORM_PERMISSIONS.ALERT_READ },
  { path: '/platform-users', label: '平台账号', icon: UserFilled, permission: PLATFORM_PERMISSIONS.USER_MANAGE },
  { path: '/audit-logs', label: '审计日志', icon: DocumentChecked, permission: PLATFORM_PERMISSIONS.AUDIT_READ },
].filter((item) => auth.can(item.permission)))

const aiSections = computed(() => [
  {
    label: '能力发布', icon: MagicStick,
    children: [{ path: '/ai/capabilities', label: '能力目录与版本', permission: PLATFORM_PERMISSIONS.AI_RELEASE_READ }],
  },
  {
    label: '模型资源', icon: Cpu,
    children: [
      { path: '/ai/llm/providers', label: 'LLM 供应商', permission: PLATFORM_PERMISSIONS.AI_CONFIG_READ },
      { path: '/ai/llm/models', label: 'LLM 模型', permission: PLATFORM_PERMISSIONS.AI_CONFIG_READ },
      { path: '/ai/embedding/providers', label: 'Embedding 供应商', permission: PLATFORM_PERMISSIONS.AI_CONFIG_READ },
      { path: '/ai/embedding/models', label: 'Embedding 模型', permission: PLATFORM_PERMISSIONS.AI_CONFIG_READ },
    ],
  },
  {
    label: 'Agent 资产', icon: SetUp,
    children: [
      { path: '/ai/agents', label: 'Agent', permission: PLATFORM_PERMISSIONS.AI_CONFIG_READ },
      { path: '/ai/prompts', label: 'Prompt', permission: PLATFORM_PERMISSIONS.AI_CONFIG_READ },
      { path: '/ai/skills', label: 'Skill', permission: PLATFORM_PERMISSIONS.AI_CONFIG_READ },
      { path: '/ai/agent-skills', label: 'Agent Skill', permission: PLATFORM_PERMISSIONS.AI_CONFIG_READ },
    ],
  },
  {
    label: '工具与诊断', icon: Tools,
    children: [
      { path: '/ai/mcp', label: 'MCP 工具治理', permission: PLATFORM_PERMISSIONS.AI_CONFIG_READ },
      { path: '/ai/semantic-retrieval', label: '语义召回诊断', permission: PLATFORM_PERMISSIONS.AI_DIAGNOSTICS_READ },
    ],
  },
].map((section) => ({ ...section, children: section.children.filter((item) => auth.can(item.permission)) }))
  .filter((section) => section.children.length))

const activeAiSection = computed(() => aiSections.value.find((section) => section.children.some((item) => route.path.startsWith(item.path))))

const signOut = async () => {
  if (signingOut.value) return
  try {
    await ElMessageBox.confirm('确认退出平台运营控制台？', '退出登录', {
      type: 'warning',
      confirmButtonText: '退出',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }

  signingOut.value = true
  try {
    await auth.signOut()
    ElMessage.success('已退出登录')
    await router.replace('/login')
  } catch {
    // The HTTP interceptor keeps the current session and reports the failure.
  } finally {
    signingOut.value = false
  }
}

const handleAccountCommand = (command: string) => {
  if (command === 'logout') void signOut()
}

const toggleSidebar = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
  try {
    localStorage.setItem(sidebarStorageKey, sidebarCollapsed.value ? '1' : '0')
  } catch {
    // Storage can be unavailable in restricted browser contexts.
  }
}
</script>

<template>
  <div class="platform-app">
  <RouterView v-if="isLogin" />
  <div v-else class="console-shell" :class="{ 'console-shell--sidebar-collapsed': sidebarCollapsed }">
    <aside class="sidebar" :class="{ 'sidebar--collapsed': sidebarCollapsed }">
      <div class="brand-mark"><span>SR</span></div>
      <div class="brand-copy"><strong>Smart Recruit</strong><small>PLATFORM CONSOLE</small></div>
      <el-scrollbar class="sidebar-nav-scroll" wrap-class="sidebar-nav-scroll__wrap">
        <nav class="sidebar-nav" aria-label="平台控制台导航">
          <RouterLink v-for="item in navigation" :key="item.path" :to="item.path" :aria-label="item.label" :title="sidebarCollapsed ? item.label : undefined">
            <el-icon><component :is="item.icon" /></el-icon><span>{{ item.label }}</span>
          </RouterLink>
          <section v-if="aiSections.length" class="sidebar-nav-group">
            <div class="sidebar-nav-group__title"><el-icon><Connection /></el-icon><span>AI 能力中心</span></div>
            <RouterLink
              v-for="section in aiSections"
              :key="section.label"
              :to="section.children[0].path"
              :aria-label="section.label"
              :title="sidebarCollapsed ? section.label : undefined"
              :class="{ 'router-link-active': section.children.some((item) => route.path.startsWith(item.path)) }"
            >
              <el-icon><component :is="section.icon" /></el-icon><span>{{ section.label }}</span>
            </RouterLink>
          </section>
        </nav>
      </el-scrollbar>
      <div class="sidebar-foot">
        <el-tooltip :content="sidebarCollapsed ? '展开菜单' : '收起菜单'" placement="right">
          <button class="sidebar-foot-action sidebar-foot-action--collapse" type="button" :aria-label="sidebarCollapsed ? '展开菜单' : '收起菜单'" :aria-expanded="!sidebarCollapsed" @click="toggleSidebar">
            <el-icon><Expand v-if="sidebarCollapsed" /><Fold v-else /></el-icon>
            <span>收起菜单</span>
          </button>
        </el-tooltip>
        <el-tooltip :content="isDark ? '切换亮色模式' : '切换暗色模式'" placement="right">
          <button class="sidebar-foot-action sidebar-foot-action--theme" type="button" :disabled="isThemeTransitioning" :aria-label="isDark ? '切换亮色模式' : '切换暗色模式'" @click="toggleTheme">
            <el-icon><Sunny v-if="isDark" /><Moon v-else /></el-icon>
            <span>{{ isDark ? '亮色模式' : '暗色模式' }}</span>
          </button>
        </el-tooltip>
      </div>
    </aside>
    <section class="workspace">
      <header class="topbar">
        <div><span class="topbar-title">平台运营控制台</span><span class="topbar-separator"></span><span class="topbar-page">{{ route.meta.title }}</span></div>
        <el-dropdown
          class="account-area"
          trigger="click"
          placement="bottom-end"
          popper-class="account-dropdown"
          :show-timeout="100"
          :hide-timeout="180"
          @command="handleAccountCommand"
        >
          <button class="account-trigger" type="button" :disabled="signingOut" aria-label="账号菜单">
            <span class="account-avatar" aria-hidden="true">{{ auth.username.slice(0, 1).toUpperCase() || 'U' }}</span>
            <span class="account-copy"><strong>{{ auth.username }}</strong><small>{{ primaryRole }}</small></span>
            <el-icon class="account-chevron"><ArrowDown /></el-icon>
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="logout" :icon="SwitchButton" :disabled="signingOut">
                {{ signingOut ? '正在退出…' : '退出登录' }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </header>
      <nav v-if="activeAiSection && activeAiSection.children.length > 1" class="ai-subnav" :aria-label="`${activeAiSection.label}导航`">
        <span>{{ activeAiSection.label }}</span>
        <RouterLink v-for="item in activeAiSection.children" :key="item.path" :to="item.path">{{ item.label }}</RouterLink>
      </nav>
      <main><RouterView /></main>
    </section>
  </div>
  </div>
</template>
