<script setup lang="ts">
import { reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { register } from '@/api/auth'
import type { RegisterPayload } from '@/types/domain'

const router = useRouter()
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const formRef = ref<any>(null)
const loading = ref(false)
const form = reactive<RegisterPayload>({ username: '', password: '', email: '', role: 1 })
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
    await register(form)
    ElMessage.success('注册成功，请登录')
    router.push('/login')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <section class="auth-box">
      <h1 class="auth-title">注册候选人账号</h1>
      <p class="auth-subtitle">创建账号，快速投递心仪岗位。</p>
      <el-form ref="formRef" label-position="top" :model="form" :rules="rules" @submit.prevent>
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="form.email" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="form.password" type="password" show-password />
        </el-form-item>
        <el-button type="primary" :loading="loading" style="width: 100%" @click="submit">注册</el-button>
      </el-form>
      <p class="auth-foot">已有账号？<RouterLink to="/login">返回登录</RouterLink></p>
    </section>
  </div>
</template>
