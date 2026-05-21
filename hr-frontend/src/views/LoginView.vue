<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const formRef = ref<any>(null)
const loading = ref(false)
const form = reactive({ username: '', password: '' })
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码至少 6 位', trigger: 'blur' },
  ],
}

const submit = async () => {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch {
    return
  }
  loading.value = true
  try {
    await auth.login(form)
    if (auth.role !== 2 && auth.role !== 3) {
      auth.logout()
      ElMessage.error('请使用 HR 账号登录')
      return
    }
    router.push('/hr/jobs')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <section class="auth-box">
      <h1 class="auth-title">HR 管理端登录</h1>
      <p class="auth-subtitle">掌控招聘流程，从这里开始。</p>
      <el-form ref="formRef" label-position="top" :model="form" :rules="rules" @submit.prevent>
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" autocomplete="username" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="form.password" type="password" autocomplete="current-password" show-password />
        </el-form-item>
        <el-button type="primary" :loading="loading" style="width: 100%" @click="submit">登录</el-button>
      </el-form>
      <p class="auth-foot">没有账号？<RouterLink to="/register">注册 HR 账号</RouterLink></p>
    </section>
  </div>
</template>
