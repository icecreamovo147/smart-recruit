<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const form = reactive({ username: '', password: '' })
const submit = async () => {
  if (!form.username.trim() || !form.password) { ElMessage.warning('请输入用户名和密码'); return }
  loading.value = true
  try {
    await auth.signIn(form.username.trim(), form.password)
    if (!auth.isLoggedIn || !auth.user?.roles.includes('platform_admin')) {
      await auth.signOut()
      ElMessage.error('当前账号没有平台控制台准入权限')
      return
    }
    await router.push('/tenants')
  } finally { loading.value = false }
}
</script>

<template>
  <div class="login-page">
    <section class="login-card">
      <p class="eyebrow">SMART RECRUIT</p>
      <h1>平台运营控制台</h1>
      <p>该入口仅面向平台管理员，不承载企业招聘业务。</p>
      <el-form label-position="top" @submit.prevent="submit">
        <el-form-item label="平台管理员账号"><el-input v-model="form.username" autocomplete="username" size="large" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="form.password" type="password" autocomplete="current-password" show-password size="large" /></el-form-item>
        <el-button type="primary" size="large" native-type="submit" :loading="loading">安全登录</el-button>
      </el-form>
    </section>
  </div>
</template>
