<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Monitor, Briefcase, Bell, UserFilled, Sunny, Moon, Menu, Close, Expand, Fold } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { useTheme } from '@/composables/useTheme'
import { updateEmail } from '@/api/auth'
import NotificationBell from '@/components/NotificationBell.vue'
import logoSmallLight from '@shared/assets/logo-small.webp'
import logoSmallDark from '@shared/assets/logo-small-dark.webp'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { isDark, toggleTheme } = useTheme()
const logoSrc = computed(() => isDark.value ? logoSmallDark : logoSmallLight)

const sidebarCollapsed = ref(false)
const mobileMenuOpen = ref(false)

const isAuthRoute = computed(() => route.path === '/login')

const navItems = [
  { path: '/dashboard', label: '工作台', icon: Monitor },
  { path: '/interviews', label: '我的面试', icon: Briefcase },
  { path: '/notifications', label: '通知', icon: Bell },
  { path: '/profile', label: '个人信息', icon: UserFilled },
]

const isActive = (path: string): boolean => {
  if (path === '/dashboard') return route.path === '/dashboard' || route.path === '/'
  return route.path.startsWith(path)
}

const pageTitle = computed(() => String(route.meta.title || '工作台'))

const userInitial = computed(() => {
  const name = auth.username
  return name ? name.charAt(0).toUpperCase() : '?'
})

const handleNavClick = (path: string) => {
  router.push(path)
  mobileMenuOpen.value = false
}

const handleLogout = async () => {
  await auth.logoutApi()
  auth.logout()
  router.push('/login')
}

const toggleSidebar = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
}

const toggleMobileMenu = () => {
  mobileMenuOpen.value = !mobileMenuOpen.value
}

// ── Email setup dialog ────────────────────────────────────────────
const showEmailSetup = ref(false)
const emailSetupInput = ref('')
const emailSetupSaving = ref(false)

watch(
  () => ({ path: route.path, loggedIn: auth.isLoggedIn, email: auth.email }),
  ({ path, loggedIn, email }) => {
    if (loggedIn && !email && path !== '/login') {
      showEmailSetup.value = true
    }
  },
  { immediate: true },
)

const saveEmailSetup = async () => {
  const email = emailSetupInput.value.trim()
  if (!email || !email.includes('@') || !email.includes('.')) {
    ElMessage.warning('请输入有效的邮箱地址')
    return
  }
  emailSetupSaving.value = true
  try {
    await updateEmail(email)
    ElMessage.success('邮箱设置成功')
    showEmailSetup.value = false
    await auth.restoreSession()
  } catch {
    // error handled by interceptor
  } finally {
    emailSetupSaving.value = false
  }
}
</script>

<template>
  <div v-if="isAuthRoute" class="auth-layout">
    <RouterView />
  </div>

  <div v-else class="app-layout">
    <!-- Mobile overlay -->
    <div
      class="sidebar-overlay"
      :class="{ 'sidebar-overlay--visible': mobileMenuOpen }"
      @click="mobileMenuOpen = false"
    />

    <!-- Sidebar -->
    <aside
      class="app-sidebar"
      :class="{
        'app-sidebar--collapsed': sidebarCollapsed,
        'app-sidebar--mobile-open': mobileMenuOpen,
      }"
    >
      <div class="sidebar-brand">
        <img class="sidebar-brand-logo" :src="logoSrc" alt="Smart Recruit" />
        <span v-if="!sidebarCollapsed">面试工作台</span>
      </div>

      <nav class="sidebar-nav">
        <a
          v-for="item in navItems"
          :key="item.path"
          class="sidebar-nav-item"
          :class="{ 'sidebar-nav-item--active': isActive(item.path) }"
          @click.prevent="handleNavClick(item.path)"
        >
          <el-icon><component :is="item.icon" /></el-icon>
          <span v-if="!sidebarCollapsed">{{ item.label }}</span>
        </a>
      </nav>
    </aside>

    <!-- Main area -->
    <div
      class="app-main"
      :class="{ 'app-main--collapsed': sidebarCollapsed }"
    >
      <header class="app-header">
        <div class="app-header-left">
          <button class="sidebar-toggle mobile-only" @click="toggleMobileMenu" aria-label="菜单">
            <el-icon :size="18">
              <Close v-if="mobileMenuOpen" />
              <Menu v-else />
            </el-icon>
          </button>
          <el-tooltip :content="sidebarCollapsed ? '展开菜单' : '折叠菜单'" placement="bottom">
            <button class="sidebar-toggle desktop-only" @click="toggleSidebar" aria-label="折叠侧边栏">
              <el-icon :size="18"><Expand v-if="sidebarCollapsed" /><Fold v-else /></el-icon>
            </button>
          </el-tooltip>
          <h1 class="app-header-title">{{ pageTitle }}</h1>
        </div>

        <div class="app-header-right">
          <button class="header-action-btn" @click="toggleTheme" :aria-label="isDark ? '切换到亮色模式' : '切换到暗色模式'">
            <el-icon :size="18">
              <Moon v-if="isDark" />
              <Sunny v-else />
            </el-icon>
          </button>

          <NotificationBell />

          <el-dropdown trigger="click" @command="(cmd: string) => { if (cmd === 'logout') handleLogout() }">
            <div class="user-menu-trigger">
              <div class="user-avatar">{{ userInitial }}</div>
              <span class="user-name">{{ auth.username }}</span>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">个人信息</el-dropdown-item>
                <el-dropdown-item divided command="logout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>

      <main class="app-content">
        <RouterView v-slot="{ Component }">
          <Transition name="page-fade" mode="out-in">
            <component :is="Component" />
          </Transition>
        </RouterView>
      </main>
    </div>

    <!-- 邮箱设置弹窗 -->
    <el-dialog
      v-model="showEmailSetup"
      title="设置通知邮箱"
      width="440px"
      :show-close="false"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
    >
      <p class="email-setup-hint">
        设置邮箱后，面试安排等重要通知将同时发送到您的邮箱，确保不会错过关键信息。
      </p>
      <el-input
        v-model="emailSetupInput"
        placeholder="请输入邮箱地址"
        clearable
        @keyup.enter="saveEmailSetup"
      />
      <template #footer>
        <el-button type="primary" :loading="emailSetupSaving" @click="saveEmailSetup">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.email-setup-hint {
  margin: 0 0 16px;
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.7;
}
.desktop-only {
  display: none;
}
@media (min-width: 769px) {
  .desktop-only {
    display: inline-flex;
  }
}
.mobile-only {
  display: inline-flex;
}
@media (min-width: 769px) {
  .mobile-only {
    display: none;
  }
}
</style>
