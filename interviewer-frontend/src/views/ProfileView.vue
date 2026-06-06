<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ROLE_KEY_INTERVIEWER } from '@/types/domain'

const ROLE_LABELS: Record<string, string> = {
  interviewer: '面试官',
  hr: 'HR',
  admin: '管理员',
}

const router = useRouter()
const auth = useAuthStore()

const accountTypeLabel = computed(() => {
  const t = auth.accountType
  if (t === 'staff') return '员工账号'
  if (t === 'candidate') return '候选人账号'
  return t || '-'
})

const handleLogout = async () => {
  await auth.logoutApi()
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="profile-view">
    <section class="profile-card">
      <h3 class="card-title">个人信息</h3>

      <div class="profile-fields">
        <div class="profile-field">
          <span class="field-label">用户名</span>
          <span class="field-value">{{ auth.username || '-' }}</span>
        </div>
        <div class="profile-field">
          <span class="field-label">账号类型</span>
          <span class="field-value">{{ accountTypeLabel }}</span>
        </div>
        <div class="profile-field">
          <span class="field-label">角色</span>
          <span class="field-value">
            <template v-if="auth.roles.length > 0">
              <el-tag
                v-for="role in auth.roles"
                :key="role"
                :type="role === ROLE_KEY_INTERVIEWER ? 'primary' : 'default'"
                size="small"
                effect="light"
                style="margin-right: 4px"
              >
                {{ ROLE_LABELS[role] || role }}
              </el-tag>
            </template>
            <span v-else>-</span>
          </span>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.profile-view {
  max-width: 600px;
}

.profile-card {
  background: var(--surface-primary);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 20px 24px;
  margin-bottom: 20px;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 16px;
}

.profile-fields {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.profile-field {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.field-label {
  font-size: 13px;
  color: var(--text-muted);
  width: 72px;
  flex-shrink: 0;
  padding-top: 2px;
}

.field-value {
  font-size: 14px;
  color: var(--text-primary);
  font-weight: 500;
}

.actions {
  display: flex;
  gap: 12px;
}
</style>
