<script setup lang="ts">
import { computed, ref } from 'vue'
import { ArrowDown, Bell, DataAnalysis, DocumentChecked, Expand, Fold, Goods, Moon, OfficeBuilding, Sunny, SwitchButton, UserFilled } from '@element-plus/icons-vue'
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
  <RouterView v-if="isLogin" />
  <div v-else class="console-shell" :class="{ 'console-shell--sidebar-collapsed': sidebarCollapsed }">
    <aside class="sidebar" :class="{ 'sidebar--collapsed': sidebarCollapsed }">
      <div class="brand-mark"><span>SR</span></div>
      <div class="brand-copy"><strong>Smart Recruit</strong><small>PLATFORM CONSOLE</small></div>
      <nav class="sidebar-nav" aria-label="平台控制台导航">
        <RouterLink v-for="item in navigation" :key="item.path" :to="item.path" :aria-label="item.label" :title="sidebarCollapsed ? item.label : undefined">
          <el-icon><component :is="item.icon" /></el-icon><span>{{ item.label }}</span>
        </RouterLink>
      </nav>
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
      <main><RouterView /></main>
    </section>
  </div>
</template>
