<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { Moon, Sunny } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { useTheme } from '@/composables/useTheme'
import request from '@/api/request'

const router = useRouter()
const auth = useAuthStore()
const { isDark, toggleTheme } = useTheme()
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
    if (auth.accountType !== 'staff') {
      await request.post('/api/v1/auth/logout').catch(() => {})
      auth.logout()
      ElMessage.error('请使用 HR 账号登录')
      return
    }
    router.push('/hr/workbench')
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
          <div class="auth-logo-accent"></div>
          <div class="auth-logo-sub">
            <span class="auth-logo-sub-name">Enterprise</span>
            <span class="auth-logo-sub-desc">Hiring Suite</span>
          </div>
        </div>
        <div class="auth-panel-copy">
          <p class="auth-kicker">Enterprise Hiring Suite</p>
          <h1>让招聘管理进入清晰、可控、可追踪的节奏</h1>
          <p>统一管理岗位、候选人和 AI 数据助手，让团队在同一个工作台里完成筛选、沟通与决策。</p>
        </div>
        <div class="auth-metrics">
          <div><strong>360°</strong><span>候选人视图</span></div>
          <div><strong>AI</strong><span>智能辅助筛选</span></div>
          <div><strong>实时</strong><span>招聘流程协同</span></div>
        </div>
      </div>

      <div class="auth-form-panel">
        <div class="auth-panel-top">
          <span>HR 管理端</span>
          <el-tooltip :content="isDark ? '切换日间模式' : '切换夜间模式'" placement="bottom">
            <button class="theme-toggle" type="button" @click="toggleTheme">
              <el-icon :size="18"><Moon v-if="!isDark" /><Sunny v-else /></el-icon>
            </button>
          </el-tooltip>
        </div>
        <section class="auth-box">
          <p class="auth-kicker">Welcome back</p>
          <h1 class="auth-title">HR 管理端登录</h1>
          <p class="auth-subtitle">掌控招聘流程，从这里开始。</p>
          <el-form ref="formRef" label-position="top" :model="form" :rules="rules" @submit.prevent>
            <el-form-item label="用户名" prop="username">
              <el-input v-model="form.username" autocomplete="username" size="large" />
            </el-form-item>
            <el-form-item label="密码" prop="password">
              <el-input v-model="form.password" type="password" autocomplete="current-password" show-password size="large" />
            </el-form-item>
            <el-button type="primary" :loading="loading" size="large" style="width: 100%" @click="submit">登录</el-button>
          </el-form>
          <p class="auth-foot">没有账号？<RouterLink to="/register">注册 HR 账号</RouterLink></p>
        </section>
      </div>
    </section>
  </div>
</template>
