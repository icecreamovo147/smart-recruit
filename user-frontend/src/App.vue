<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowDown, Moon, Sunny, UserFilled } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { useTheme } from '@/composables/useTheme'
import { useOfferBadge } from '@/composables/useOfferBadge'
import request from '@/api/request'
import { updateEmail } from '@/api/auth'
import CandidateAIAssistant from '@/components/CandidateAIAssistant.vue'
import NotificationBell from '@/components/NotificationBell.vue'
import EmailSetupDialog from '@shared/components/EmailSetupDialog.vue'
import logoFullLight from '@shared/assets/logo-full.webp'
import logoFullDark from '@shared/assets/logo-full-dark.webp'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const { isDark, toggleTheme } = useTheme()
const { pendingOfferCount, refreshPendingOfferCount } = useOfferBadge()
const logoSrc = computed(() => (isDark.value ? logoFullDark : logoFullLight))

// Fetch pending-offer badge count on mount when logged in
onMounted(() => {
  if (auth.isLoggedIn) {
    refreshPendingOfferCount()
  }
})

const logout = async () => {
  try {
    await ElMessageBox.confirm('确认退出当前账号？', '退出登录', {
      type: 'warning',
      confirmButtonText: '退出',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  try {
    // Clear httpOnly cookie server-side first; only clean local state on success.
    await request.post('/api/v1/auth/logout')
  } catch {
    ElMessage.error('退出登录失败，请稍后重试')
    return
  }
  auth.logout()
  ElMessage.success('已退出登录')
  router.push('/jobs')
}

const handleUserCommand = (command: string) => {
  if (command === 'profile') {
    router.push('/profile')
    return
  }
  if (command === 'logout') logout()
}

// ── Email setup dialog ────────────────────────────────────────────
const showEmailSetup = ref(false)

watch(
  () => ({ path: route.fullPath, loggedIn: auth.isLoggedIn, email: auth.email }),
  ({ path, loggedIn, email }) => {
    if (loggedIn && !email && path !== '/login' && path !== '/register') {
      showEmailSetup.value = true
    }
  },
  { immediate: true },
)

const handleEmailSaved = async (email: string) => {
  try {
    await updateEmail(email)
    ElMessage.success('邮箱设置成功')
    showEmailSetup.value = false
    await auth.restoreSession()
  } catch {
    // error handled by interceptor
  }
}


</script>

<template>
  <div :class="{ 'candidate-auth-shell': route.path === '/login' || route.path === '/register' }">
    <header class="topbar">
      <RouterLink class="brand" to="/jobs" aria-label="智联招聘">
        <img class="brand-logo" :src="logoSrc" alt="智联招聘" />
      </RouterLink>
      <nav class="topbar-nav" aria-label="主导航">
        <el-scrollbar class="topbar-nav__scroll">
          <div class="topbar-nav__inner">
            <RouterLink to="/jobs">岗位列表</RouterLink>
            <RouterLink v-if="auth.isLoggedIn" to="/progress">
              求职进展
              <el-badge v-if="pendingOfferCount > 0" :value="pendingOfferCount" class="nav-badge" />
            </RouterLink>
            <RouterLink v-if="auth.isLoggedIn" to="/profile">个人资料</RouterLink>
            <RouterLink v-if="auth.isLoggedIn" to="/resume">个人简历</RouterLink>
          </div>
        </el-scrollbar>
      </nav>
      <div class="account" style="display:flex;align-items:center;gap:4px;">
        <el-tooltip :content="isDark ? '切换日间模式' : '切换夜间模式'" placement="bottom">
          <button class="theme-toggle" @click="toggleTheme">
            <el-icon :size="18"><Moon v-if="!isDark" /><Sunny v-else /></el-icon>
          </button>
        </el-tooltip>
        <NotificationBell v-if="auth.isLoggedIn" />
        <el-dropdown v-if="auth.isLoggedIn" trigger="click" @command="handleUserCommand">
          <button class="user-menu">
            <span class="user-avatar"><el-icon><UserFilled /></el-icon></span>
            <span class="user-name mobile-user-name">{{ auth.username || '候选人' }}</span>
            <el-icon><ArrowDown /></el-icon>
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="profile">个人资料</el-dropdown-item>
              <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-button v-else type="primary" @click="router.push('/login')">登录</el-button>
      </div>
    </header>
    <main
      class="container"
      :class="{
        'container--auth': route.path === '/login' || route.path === '/register',
        'container--content':
          route.path === '/jobs' ||
          route.path === '/progress' ||
          route.path === '/profile' ||
          route.path === '/resume',
        'container--jobs': route.path === '/jobs',
      }"
    >
      <RouterView v-slot="{ Component }">
        <Transition name="page-fade" mode="out-in">
          <component :is="Component" />
        </Transition>
      </RouterView>
    </main>
    <CandidateAIAssistant v-if="auth.isLoggedIn" />

    <!-- 邮箱设置弹窗 -->
    <EmailSetupDialog v-model="showEmailSetup" @saved="handleEmailSaved" @error="(msg) => ElMessage.warning(msg)" />
  </div>
</template>

<style scoped>
.topbar-nav {
  flex: 1;
  min-width: 0;
}

.topbar-nav__scroll {
  width: 100%;
}

.topbar-nav__scroll :deep(.el-scrollbar__wrap) {
  overflow-y: hidden;
}

.topbar-nav__scroll :deep(.el-scrollbar__bar.is-vertical) {
  display: none;
}

.topbar-nav__inner {
  display: flex;
  align-items: center;
  gap: 18px;
  white-space: nowrap;
  padding-bottom: 2px;
}

.nav-badge {
  margin-left: 2px;
}
.nav-badge :deep(.el-badge__content) {
  font-size: 10px;
  height: 16px;
  line-height: 16px;
  padding: 0 4px;
  border: none;
}
</style>
