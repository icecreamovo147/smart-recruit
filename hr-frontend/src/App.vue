<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowDown, Briefcase, Calendar, ChatDotRound, Collection, DataAnalysis, Expand, Fold, Key, Menu, Monitor, Moon, OfficeBuilding, Operation, Sunny, UserFilled } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { useTheme } from '@/composables/useTheme'
import request from '@/api/request'
import { updateEmail } from '@/api/auth'
import NotificationBell from '@/components/NotificationBell.vue'
import SidebarNavGroup from '@/components/SidebarNavGroup.vue'
import EmailSetupDialog from '@shared/components/EmailSetupDialog.vue'
import { PERM } from '@/types/domain'
import { resolveStaffHomePath } from '@/utils/navigation'
import logoSmallLight from '@shared/assets/logo-small.webp'
import logoSmallDark from '@shared/assets/logo-small-dark.webp'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { isDark, toggleTheme } = useTheme()
const logoSrc = computed(() => isDark.value ? logoSmallDark : logoSmallLight)
const sidebarCollapsed = ref(false)
const mobileSidebarOpen = ref(false)
const switchingTenant = ref(false)
const taxonomyOpen = ref(false)
const usageAuditOpen = ref(false)
const isAuthRoute = computed(() => route.path === '/login' || route.path === '/register')
const homePath = computed(() => resolveStaffHomePath(auth))
const activeMemberships = computed(() => auth.memberships.filter(
  (membership) => membership.tenant_status === 'active' && membership.membership_status === 'active',
))
const canSwitchTenant = computed(() => activeMemberships.value.length > 1)
const activeTenantName = computed(() => auth.activeTenant?.name?.trim() || '当前企业')

const toggleTaxonomy = () => {
  taxonomyOpen.value = !taxonomyOpen.value
}

const toggleUsageAudit = () => {
  usageAuditOpen.value = !usageAuditOpen.value
}

const openMobileSidebar = () => { mobileSidebarOpen.value = true }
const closeMobileSidebar = () => { mobileSidebarOpen.value = false }

watch(() => route.fullPath, () => {
  closeMobileSidebar()
  // Auto-expand taxonomy group when on a taxonomy sub-page
  if (route.path.startsWith('/hr/admin/departments') || route.path.startsWith('/hr/admin/locations')) {
    taxonomyOpen.value = true
  }
  if (route.path.startsWith('/hr/admin/usage')) {
    usageAuditOpen.value = true
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
  try {
    // Clear httpOnly cookie server-side first; only clean local state on success.
    await request.post('/api/v1/auth/logout')
  } catch {
    ElMessage.error('退出登录失败，请稍后重试')
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

const handleTenantChange = async (tenantId: number) => {
  if (!tenantId || tenantId === auth.tenantId || switchingTenant.value) return
  switchingTenant.value = true
  try {
    await auth.switchTenant(tenantId)
    ElMessage.success(`已切换至${auth.activeTenant?.name || '目标企业'}`)
    window.location.assign(homePath.value)
  } catch {
    ElMessage.error('企业切换失败，请重新登录后重试')
  } finally {
    switchingTenant.value = false
  }
}

const toggleSidebar = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
}

// ── Email setup dialog ────────────────────────────────────────────
const showEmailSetup = ref(false)
const shouldShowEmailSetup = computed(() => Boolean(route.meta.requiresAuth && auth.isLoggedIn && !auth.email))

watch(
  shouldShowEmailSetup,
  (shouldShow) => {
    showEmailSetup.value = shouldShow
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

const routeViewKey = (viewRoute: { fullPath: string; path: string; params: Record<string, unknown> }): string => {
  const candidateUserId = viewRoute.params.candidateUserId
  if (viewRoute.path.startsWith('/hr/candidates/') && candidateUserId) {
    return `/hr/candidates/${String(candidateUserId)}`
  }
  // For AI chat, use path-only key so query changes (application_id → session_id)
  // don't destroy and recreate the component mid-stream.
  if (viewRoute.path === '/hr/ai') {
    return '/hr/ai'
  }
  return viewRoute.fullPath
}

</script>

<template>
  <div class="app-shell" :class="{ 'app-shell--auth': isAuthRoute }">
    <div v-if="mobileSidebarOpen" class="mobile-sidebar-backdrop" @click="closeMobileSidebar"></div>
    <aside v-if="route.meta.requiresAuth" class="sidebar" :class="{ 'sidebar--collapsed': sidebarCollapsed, 'sidebar--mobile-open': mobileSidebarOpen }">
      <div class="sidebar-head">
        <RouterLink class="brand sidebar-brand" :to="homePath" aria-label="智联招聘 HR">
          <img class="sidebar-brand__icon" :src="logoSrc" alt="智联招聘" />
          <span v-if="!sidebarCollapsed" class="sidebar-brand__text">智联招聘</span>
        </RouterLink>
      </div>
      <div class="sidebar-nav">
        <el-scrollbar>
          <RouterLink v-if="auth.hasPermission(PERM.JOB_READ)" class="sidebar-link" to="/hr/workbench" @click="closeMobileSidebar">
            <el-icon><Monitor /></el-icon>
            <span>工作台</span>
          </RouterLink>
          <SidebarNavGroup
            v-if="auth.hasAnyPermission(PERM.ADMIN_DEPARTMENT_MANAGE, PERM.ADMIN_LOCATION_MANAGE)"
            :icon="Operation"
            label="基础数据"
            :open="taxonomyOpen"
            :collapsed="sidebarCollapsed"
            :mobile-open="mobileSidebarOpen"
            :items="[
              { to: '/hr/admin/departments', label: '部门管理', visible: auth.hasPermission(PERM.ADMIN_DEPARTMENT_MANAGE) },
              { to: '/hr/admin/locations', label: '地点管理', visible: auth.hasPermission(PERM.ADMIN_LOCATION_MANAGE) },
            ]"
            @toggle="toggleTaxonomy"
            @close-mobile="closeMobileSidebar"
          />
          <RouterLink v-if="auth.hasPermission(PERM.JOB_READ)" class="sidebar-link" to="/hr/jobs" @click="closeMobileSidebar">
            <el-icon><Briefcase /></el-icon>
            <span>岗位管理</span>
          </RouterLink>
          <RouterLink v-if="auth.hasPermission(PERM.INTERVIEW_READ)" class="sidebar-link" to="/hr/my-interviews" @click="closeMobileSidebar">
            <el-icon><Calendar /></el-icon>
            <span>我的面试</span>
          </RouterLink>
          <RouterLink v-if="auth.hasPermission(PERM.AI_HR_USE)" class="sidebar-link" to="/hr/ai" @click="closeMobileSidebar">
            <el-icon><ChatDotRound /></el-icon>
            <span>AI 数据助手</span>
          </RouterLink>
          <RouterLink v-if="auth.hasPermission(PERM.BILLING_MANAGE)" class="sidebar-link" to="/hr/billing" @click="closeMobileSidebar">
            <el-icon><DataAnalysis /></el-icon>
            <span>AI 套餐与额度</span>
          </RouterLink>
          <RouterLink v-if="auth.hasPermission(PERM.ADMIN_INVITE_MANAGE)" class="sidebar-link" to="/hr/admin/invite-codes" @click="closeMobileSidebar">
            <el-icon><Key /></el-icon>
            <span>邀请码管理</span>
          </RouterLink>
          <RouterLink v-if="auth.hasPermission(PERM.ADMIN_USER_MANAGE)" class="sidebar-link" to="/hr/admin/staff-users" @click="closeMobileSidebar">
            <el-icon><UserFilled /></el-icon>
            <span>员工账号</span>
          </RouterLink>
          <SidebarNavGroup
            v-if="auth.hasPermission(PERM.AUDIT_USAGE_READ)"
            :icon="DataAnalysis"
            label="第三方服务审计"
            :open="usageAuditOpen"
            :collapsed="sidebarCollapsed"
            :mobile-open="mobileSidebarOpen"
            :items="[
              { to: '/hr/admin/usage-stats', label: '使用统计' },
              { to: '/hr/admin/usage-audit', label: '审计日志' },
            ]"
            @toggle="toggleUsageAudit"
            @close-mobile="closeMobileSidebar"
          />
        </el-scrollbar>
      </div>
      <div class="sidebar-footer">
        <el-tooltip :content="sidebarCollapsed ? '展开菜单' : '折叠菜单'" placement="right">
          <button class="sidebar-footer-action sidebar-footer-action--collapse sidebar-toggle" :aria-label="sidebarCollapsed ? '展开菜单' : '折叠菜单'" @click="toggleSidebar">
            <el-icon :size="18"><Expand v-if="sidebarCollapsed" /><Fold v-else /></el-icon>
          </button>
        </el-tooltip>
        <el-tooltip :content="isDark ? '切换日间模式' : '切换夜间模式'" placement="right">
          <button class="sidebar-footer-action sidebar-footer-action--theme theme-toggle" :aria-label="isDark ? '切换日间模式' : '切换夜间模式'" @click="toggleTheme">
            <el-icon :size="18"><Moon v-if="!isDark" /><Sunny v-else /></el-icon>
            <span class="sidebar-footer-action__label">{{ isDark ? '日间模式' : '夜间模式' }}</span>
          </button>
        </el-tooltip>
      </div>
    </aside>
    <div class="workspace" :class="{ 'workspace--collapsed': sidebarCollapsed }">
      <header v-if="route.meta.requiresAuth" class="top-header">
        <div class="header-left">
          <button class="sidebar-toggle mobile-only" @click="openMobileSidebar">
            <el-icon :size="18"><Menu /></el-icon>
          </button>
          <NotificationBell v-if="route.meta.requiresAuth" />
        </div>
        <div class="header-actions">
          <el-select
            v-if="canSwitchTenant"
            class="tenant-switcher"
            :model-value="auth.tenantId"
            :loading="switchingTenant"
            aria-label="切换企业"
            @change="handleTenantChange"
          >
            <template #prefix>
              <el-icon><OfficeBuilding /></el-icon>
            </template>
            <el-option
              v-for="membership in activeMemberships"
              :key="membership.membership_id"
              :label="membership.name"
              :value="membership.tenant_id"
            />
          </el-select>
          <div
            v-else-if="auth.activeTenant"
            class="tenant-context"
            :title="activeTenantName"
            aria-label="当前企业"
          >
            <el-icon><OfficeBuilding /></el-icon>
            <span>{{ activeTenantName }}</span>
          </div>
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
      <main class="main-panel" :class="{ 'main-panel--auth': isAuthRoute }">
        <RouterView v-slot="{ Component, route: viewRoute }">
          <Transition name="page-fade" mode="out-in">
            <component :is="Component" :key="routeViewKey(viewRoute)" />
          </Transition>
        </RouterView>
      </main>
    </div>

    <!-- 邮箱设置弹窗 -->
    <EmailSetupDialog
      v-if="route.meta.requiresAuth"
      v-model="showEmailSetup"
      @saved="handleEmailSaved"
      @error="(msg) => ElMessage.warning(msg)"
    />
  </div>
</template>
