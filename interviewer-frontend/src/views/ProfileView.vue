<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Edit } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { updateEmail } from '@/api/auth'
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

const editingEmail = ref(false)
const emailInput = ref(auth.email || '')
const savingEmail = ref(false)

const startEditEmail = () => {
  emailInput.value = auth.email || ''
  editingEmail.value = true
}

const saveEmail = async () => {
  savingEmail.value = true
  try {
    await updateEmail(emailInput.value.trim())
    ElMessage.success('邮箱更新成功')
    editingEmail.value = false
    await auth.restoreSession()
  } catch {
    // error handled by interceptor
  } finally {
    savingEmail.value = false
  }
}

const cancelEditEmail = () => {
  editingEmail.value = false
}

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
        <div class="profile-field">
          <span class="field-label">邮箱</span>
          <span class="field-value">
            <template v-if="editingEmail">
              <div class="email-edit-row">
                <el-input v-model="emailInput" placeholder="请输入邮箱" clearable size="small" style="max-width: 240px" />
                <el-button type="primary" :loading="savingEmail" size="small" @click="saveEmail">保存</el-button>
                <el-button size="small" @click="cancelEditEmail">取消</el-button>
              </div>
            </template>
            <template v-else>
              <span>{{ auth.email || '未设置' }}</span>
              <el-button :icon="Edit" link type="primary" size="small" style="margin-left: 8px" @click="startEditEmail">编辑</el-button>
            </template>
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

.email-edit-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.actions {
  display: flex;
  gap: 12px;
}
</style>
