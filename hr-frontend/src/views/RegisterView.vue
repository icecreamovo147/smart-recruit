<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { Moon, Sunny } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { register } from '@/api/auth'
import { useTheme } from '@/composables/useTheme'

const router = useRouter()
const { isDark, toggleTheme } = useTheme()
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const formRef = ref<any>(null)
const loading = ref(false)
const form = reactive({ username: '', password: '', email: '', role: 2 })
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 8, message: '密码至少 8 位', trigger: 'blur' },
    {
      pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d|.*[!@#$%^&*])/,
      message: '密码需包含大小写字母、数字或特殊字符中的至少三类',
      trigger: 'blur',
    },
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
          <h1>从账号创建开始，搭建高效招聘工作台</h1>
          <p>面向 HR 团队的岗位发布、候选人管理和数据洞察能力，帮助企业沉淀标准化招聘流程。</p>
        </div>
        <div class="auth-metrics">
          <div><strong>协同</strong><span>团队统一操作</span></div>
          <div><strong>安全</strong><span>权限角色隔离</span></div>
          <div><strong>数据</strong><span>全链路可追踪</span></div>
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
          <p class="auth-kicker">Create workspace</p>
          <h1 class="auth-title">注册 HR 账号</h1>
          <p class="auth-subtitle">创建管理账号，快速发布岗位。</p>
          <el-form ref="formRef" label-position="top" :model="form" :rules="rules" @submit.prevent>
            <el-form-item label="用户名" prop="username">
              <el-input v-model="form.username" size="large" />
            </el-form-item>
            <el-form-item label="邮箱">
              <el-input v-model="form.email" size="large" />
            </el-form-item>
            <el-form-item label="密码" prop="password">
              <el-input v-model="form.password" type="password" show-password size="large" />
            </el-form-item>
            <el-button type="primary" :loading="loading" size="large" style="width: 100%" @click="submit">注册</el-button>
          </el-form>
          <p class="auth-foot">已有账号？<RouterLink to="/login">返回登录</RouterLink></p>
        </section>
      </div>
    </section>
  </div>
</template>
