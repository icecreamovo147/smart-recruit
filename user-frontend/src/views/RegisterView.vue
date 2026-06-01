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
        </div>
        <div class="auth-panel-copy">
          <p class="auth-kicker">Candidate Career Portal</p>
          <h1>创建职业档案，让合适的岗位更快找到你</h1>
          <p>完善个人资料、上传简历并保存投递记录，用更清晰的方式推进下一次职业机会。</p>
        </div>
        <div class="auth-metrics">
          <div><strong>机会</strong><span>岗位智能匹配</span></div>
          <div><strong>资料</strong><span>简历集中维护</span></div>
          <div><strong>反馈</strong><span>申请进度同步</span></div>
        </div>
      </div>

      <div class="auth-form-panel">
        <div class="auth-panel-top">
          <span>候选人端</span>
          <RouterLink to="/jobs">浏览岗位</RouterLink>
        </div>
        <section class="auth-box">
          <p class="auth-kicker">Create profile</p>
          <h1 class="auth-title">注册候选人账号</h1>
          <p class="auth-subtitle">创建账号，快速投递心仪岗位。</p>
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
