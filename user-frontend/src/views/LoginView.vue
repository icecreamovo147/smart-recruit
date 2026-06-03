<script setup lang="ts">
import { reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import request from '@/api/request'

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
    if (auth.accountType !== 'candidate') {
      await request.post('/api/v1/auth/logout').catch(() => {})
      auth.logout()
      ElMessage.error('请使用候选人账号登录')
      return
    }
    ElMessage.success('登录成功')
    router.push('/jobs')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <section class="auth-shell">
      <div class="auth-brand-panel">
        <div class="auth-logo-frame">
          <span class="auth-logo-text">智联招聘</span>
        </div>
        <div class="auth-panel-copy">
          <p class="auth-kicker">Candidate Career Portal</p>
          <h1>把职位发现、简历管理和投递进度放在同一个入口</h1>
          <p>登录后继续浏览匹配岗位，维护个人资料，并实时查看每一次投递的处理状态。</p>
        </div>
        <div class="auth-metrics">
          <div><strong>精选</strong><span>企业岗位池</span></div>
          <div><strong>简历</strong><span>在线资料管理</span></div>
          <div><strong>进度</strong><span>投递状态追踪</span></div>
        </div>
      </div>

      <div class="auth-form-panel">
        <div class="auth-panel-top">
          <span>候选人端</span>
          <RouterLink to="/jobs">浏览岗位</RouterLink>
        </div>
        <section class="auth-box">
          <p class="auth-kicker">Welcome back</p>
          <h1 class="auth-title">候选人登录</h1>
          <p class="auth-subtitle">欢迎回来，继续寻找你的下一份工作。</p>
          <el-form ref="formRef" label-position="top" :model="form" :rules="rules" @submit.prevent>
            <el-form-item label="用户名" prop="username">
              <el-input v-model="form.username" autocomplete="username" size="large" />
            </el-form-item>
            <el-form-item label="密码" prop="password">
              <el-input v-model="form.password" type="password" show-password autocomplete="current-password" size="large" />
            </el-form-item>
            <el-button type="primary" :loading="loading" size="large" style="width: 100%" @click="submit">登录</el-button>
          </el-form>
          <p class="auth-foot">没有账号？<RouterLink to="/register">注册候选人账号</RouterLink></p>
        </section>
      </div>
    </section>
  </div>
</template>
