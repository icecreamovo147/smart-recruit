<script setup lang="ts">
import { useRouter } from 'vue-router'
import { ArrowDown, Moon, Sunny, UserFilled } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { useTheme } from '@/composables/useTheme'
import CandidateAIAssistant from '@/components/CandidateAIAssistant.vue'
import NotificationBell from '@/components/NotificationBell.vue'
import logoFull from '@/assets/logo-full.png'

const router = useRouter()
const auth = useAuthStore()
const { isDark, toggleTheme } = useTheme()

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
</script>

<template>
  <div>
    <header class="topbar">
      <RouterLink class="brand" to="/jobs" aria-label="智联招聘">
        <img class="brand-logo" :src="logoFull" alt="智联招聘" />
      </RouterLink>
      <nav>
        <RouterLink to="/jobs">岗位</RouterLink>
        <RouterLink v-if="auth.isLoggedIn" to="/profile">资料</RouterLink>
        <RouterLink v-if="auth.isLoggedIn" to="/resume">简历</RouterLink>
        <RouterLink v-if="auth.isLoggedIn" to="/applications">投递</RouterLink>
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
    <main class="container">
      <RouterView v-slot="{ Component }">
        <Transition name="page-fade" mode="out-in">
          <component :is="Component" />
        </Transition>
      </RouterView>
    </main>
    <CandidateAIAssistant v-if="auth.isLoggedIn" />
  </div>
</template>
