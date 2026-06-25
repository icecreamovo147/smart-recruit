<script setup lang="ts">
import { computed, ref } from 'vue'
import { Edit } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { updateEmail } from '@/api/auth'
import type { User } from '@/types/domain'

const auth = useAuthStore()
const user = computed((): Partial<User> => auth.user || {})

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
</script>

<template>
  <section>
    <div class="page-header">
      <h1 class="page-title">个人信息</h1>
    </div>
    <div class="content-surface profile-panel">
      <div class="profile-avatar">{{ (auth.username || 'H').slice(0, 1).toUpperCase() }}</div>
      <div class="profile-info">
        <h2>{{ auth.username || 'HR 用户' }}</h2>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="账号角色">{{ auth.isRecruitingAdmin ? '招聘管理员' : auth.isSystemAdmin ? '系统管理员' : '招聘专员' }}</el-descriptions-item>
          <el-descriptions-item label="邮箱">
            <template v-if="editingEmail">
              <div class="email-edit-row">
                <el-input v-model="emailInput" placeholder="请输入邮箱" clearable style="max-width: 280px" />
                <el-button type="primary" :loading="savingEmail" size="small" @click="saveEmail">保存</el-button>
                <el-button size="small" @click="cancelEditEmail">取消</el-button>
              </div>
            </template>
            <template v-else>
              <span>{{ auth.email || '未设置' }}</span>
              <el-button :icon="Edit" link type="primary" size="small" style="margin-left: 8px" @click="startEditEmail">编辑</el-button>
            </template>
          </el-descriptions-item>
          <el-descriptions-item label="登录状态">已登录</el-descriptions-item>
        </el-descriptions>
      </div>
    </div>
  </section>
</template>

<style scoped>
.email-edit-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
