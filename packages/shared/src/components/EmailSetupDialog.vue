<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'saved', email: string): void
  (e: 'error', message: string): void
}>()

const emailInput = ref('')
const saving = ref(false)

const save = async () => {
  const email = emailInput.value.trim()
  if (!email || !email.includes('@') || !email.includes('.')) {
    emit('error', '请输入有效的邮箱地址')
    return
  }
  saving.value = true
  try {
    emit('saved', email)
    // Parent handles the API call; on success parent will set modelValue to false
  } catch {
    // error handled by parent
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    title="设置通知邮箱"
    width="440px"
    :show-close="false"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
  >
    <p class="email-setup-hint">
      设置邮箱后，面试安排、Offer 等重要通知将同时发送到您的邮箱，确保不会错过关键信息。
    </p>
    <el-input
      v-model="emailInput"
      placeholder="请输入邮箱地址"
      clearable
      @keyup.enter="save"
    />
    <template #footer>
      <el-button type="primary" :loading="saving" @click="save">保存</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.email-setup-hint {
  margin: 0 0 16px;
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.7;
}
</style>
