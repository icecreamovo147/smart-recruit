<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowDown, Briefcase, ChatDotRound, Expand, Fold, Key, Menu, Monitor, Moon, Operation, Sunny, UserFilled } from '@element-plus/icons-vue'
import { ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { useTheme } from '@/composables/useTheme'
import NotificationBell from '@/components/NotificationBell.vue'
import logoSmall from '@/assets/logo-small.png'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { isDark, toggleTheme } = useTheme()
const sidebarCollapsed = ref(false)
const mobileSidebarOpen = ref(false)
const taxonomyOpen = ref(false)

const toggleTaxonomy = () => {
  taxonomyOpen.value = !taxonomyOpen.value
}

const openMobileSidebar = () => { mobileSidebarOpen.value = true }
const closeMobileSidebar = () => { mobileSidebarOpen.value = false }

watch(() => route.fullPath, () => {
  closeMobileSidebar()
  // Auto-expand taxonomy group when on a taxonomy sub-page
  if (route.path.startsWith('/hr/admin/departments') || route.path.startsWith('/hr/admin/locations')) {
    taxonomyOpen.value = true
  }
})

const logout = async () => {
  try {
    await ElMessageBox.confirm('确认退出当前 HR 账号？', '退出登录', {
      type: 'warning',
      confirmButtonText: '退出',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  auth.logout()
  router.push('/login')
}

const handleUserCommand = (command: string) => {
  if (command === 'profile') {
    router.push('/hr/profile')
    return
  }
  if (command === 'logout') logout()
}

const toggleSidebar = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
}
</script>

<template>
  <div class="app-shell">
    <div v-if="mobileSidebarOpen" class="mobile-sidebar-backdrop" @click="closeMobileSidebar"></div>
    <aside v-if="route.meta.requiresAuth" class="sidebar" :class="{ 'sidebar--collapsed': sidebarCollapsed, 'sidebar--mobile-open': mobileSidebarOpen }">
      <div class="sidebar-head">
        <RouterLink class="brand sidebar-brand" to="/hr/jobs" aria-label="智联招聘 HR">
          <img class="sidebar-brand__icon" :src="logoSmall" alt="智联招聘" />
          <span v-if="!sidebarCollapsed" class="sidebar-brand__text">智联招聘</span>
        </RouterLink>
      </div>
      <RouterLink class="sidebar-link" to="/hr/workbench" @click="closeMobileSidebar">
        <el-icon><Monitor /></el-icon>
        <span>工作台</span>
      </RouterLink>
      <!-- 基础数据 (admin only, first) -->
      <template v-if="auth.role === 3">
        <button class="sidebar-link sidebar-group-toggle" type="button" :aria-expanded="taxonomyOpen && !sidebarCollapsed" @click="toggleTaxonomy">
          <el-icon><Operation /></el-icon>
          <span>基础数据</span>
          <el-icon class="group-arrow" :class="{ 'group-arrow--open': taxonomyOpen }"><ArrowDown /></el-icon>
        </button>
        <el-collapse-transition>
          <div v-show="taxonomyOpen && !sidebarCollapsed" class="sidebar-sub-group">
            <RouterLink class="sidebar-link sidebar-sub-link" to="/hr/admin/departments" @click="closeMobileSidebar">
              <span>部门管理</span>
            </RouterLink>
            <RouterLink class="sidebar-link sidebar-sub-link" to="/hr/admin/locations" @click="closeMobileSidebar">
              <span>地点管理</span>
            </RouterLink>
          </div>
        </el-collapse-transition>
      </template>
      <RouterLink class="sidebar-link" to="/hr/jobs" @click="closeMobileSidebar">
        <el-icon><Briefcase /></el-icon>
        <span>岗位管理</span>
      </RouterLink>
      <RouterLink class="sidebar-link" to="/hr/ai" @click="closeMobileSidebar">
        <el-icon><ChatDotRound /></el-icon>
        <span>AI 数据助手</span>
      </RouterLink>
      <RouterLink v-if="auth.role === 3" class="sidebar-link" to="/hr/admin/invite-codes" @click="closeMobileSidebar">
        <el-icon><Key /></el-icon>
        <span>邀请码管理</span>
      </RouterLink>
    </aside>
    <div class="workspace">
      <header v-if="route.meta.requiresAuth" class="top-header">
        <div class="header-left">
          <el-tooltip :content="sidebarCollapsed ? '展开菜单' : '折叠菜单'" placement="bottom">
            <button class="sidebar-toggle desktop-only" @click="toggleSidebar">
              <el-icon :size="18"><Expand v-if="sidebarCollapsed" /><Fold v-else /></el-icon>
            </button>
          </el-tooltip>
          <button class="sidebar-toggle mobile-only" @click="openMobileSidebar">
            <el-icon :size="18"><Menu /></el-icon>
          </button>
          <div class="header-title">{{ route.meta.title || 'HR 工作台' }}</div>
        </div>
        <div style="display:flex;align-items:center;gap:4px;">
          <el-tooltip :content="isDark ? '切换日间模式' : '切换夜间模式'" placement="bottom">
            <button class="theme-toggle" @click="toggleTheme">
              <el-icon :size="18"><Moon v-if="!isDark" /><Sunny v-else /></el-icon>
            </button>
          </el-tooltip>
          <NotificationBell v-if="route.meta.requiresAuth" />
          <el-dropdown trigger="hover" @command="handleUserCommand">
          <button class="user-menu">
            <span class="user-avatar"><el-icon><UserFilled /></el-icon></span>
            <span class="user-name mobile-user-name">{{ auth.username || 'HR 用户' }}</span>
            <el-icon><ArrowDown /></el-icon>
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="profile">个人信息</el-dropdown-item>
              <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        </div>
      </header>
      <main class="main-panel">
        <RouterView v-slot="{ Component }">
          <Transition name="page-fade" mode="out-in">
            <component :is="Component" />
          </Transition>
        </RouterView>
      </main>
    </div>
  </div>
</template>
