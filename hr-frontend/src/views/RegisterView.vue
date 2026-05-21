<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter, useRoute, RouterLink } from 'vue-router'
import { ElMessage } from 'element-plus'
import { register } from '@/api/auth'

const router = useRouter()
const route = useRoute()
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const formRef = ref<any>(null)
const loading = ref(false)
const form = reactive({ username: '', password: '', email: '', role: 2, invite_code: (route.query.invite_code as string) || '' })
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
      <h1 class="auth-title">注册 HR 账号</h1>
      <p class="auth-subtitle">创建管理账号，快速发布岗位。</p>
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
        <el-form-item label="邀请码" prop="invite_code">
          <el-input v-model="form.invite_code" placeholder="HR 注册需要管理员邀请码" />
        </el-form-item>
        <el-button type="primary" :loading="loading" style="width: 100%" @click="submit">注册</el-button>
      </el-form>
      <p class="auth-foot">已有账号？<RouterLink to="/login">返回登录</RouterLink></p>
    </section>
  </div>
</template>
