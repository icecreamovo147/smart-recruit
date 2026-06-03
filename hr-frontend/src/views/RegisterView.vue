<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { useRouter, useRoute, RouterLink } from 'vue-router'
import { Moon, Sunny } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { register, validateInviteCode } from '@/api/auth'
import { useTheme } from '@/composables/useTheme'

const router = useRouter()
const route = useRoute()
const { isDark, toggleTheme } = useTheme()
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const formRef = ref<any>(null)
const loading = ref(false)

// ── Invite code state ──
const inviteCode = ref('')
const inviteValid = ref<boolean | null>(null) // null=unchecked, true=valid, false=invalid
const inviteValidating = ref(false)
const inviteMsg = ref('')

// Read invite_code from URL query param
const urlInviteCode = (route.query.invite_code as string) || ''

const form = reactive({ username: '', password: '', email: '', role: 2 })

const validatePasswordComplexity = (_rule: any, value: string, callback: any) => {
  if (!value) { callback(); return }
  let categories = 0
  if (/[a-z]/.test(value)) categories++
  if (/[A-Z]/.test(value)) categories++
  if (/\d/.test(value)) categories++
  if (/[!@#$%^&*]/.test(value)) categories++
  if (categories >= 3) { callback(); return }
  callback(new Error('密码需包含大小写字母、数字或特殊字符中的至少三类'))
}

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 8, message: '密码至少 8 位', trigger: 'blur' },
    { validator: validatePasswordComplexity, trigger: 'blur' },
  ],
}

// ── Validate invite code on mount if provided in URL ──
onMounted(async () => {
  if (urlInviteCode) {
    inviteCode.value = urlInviteCode
    await checkInviteCode()
  }
})

const checkInviteCode = async () => {
  const code = inviteCode.value.trim()
  if (!code) {
    inviteValid.value = null
    inviteMsg.value = ''
    return
  }
  inviteValidating.value = true
  inviteValid.value = null
  try {
    const res = await validateInviteCode(code)
    inviteValid.value = res.valid
    inviteMsg.value = res.valid ? '邀请码有效，可以注册 HR 账号' : (res.msg || '邀请码无效或已过期')
  } catch {
    inviteValid.value = false
    inviteMsg.value = '邀请码验证失败，请稍后重试'
  } finally {
    inviteValidating.value = false
  }
}

const submit = async () => {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch {
    return
  }
  if (inviteValid.value !== true) {
    ElMessage.warning('请先验证邀请码')
    return
  }
  loading.value = true
  try {
    await register({ ...form, invite_code: inviteCode.value.trim() })
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
        <section class="auth-box register-box">
          <p class="auth-kicker">Create workspace</p>
          <h1 class="auth-title">注册 HR 账号</h1>
          <p class="auth-subtitle">HR 账号需通过邀请码注册，请联系管理员获取。</p>

          <el-form ref="formRef" label-position="top" :model="form" :rules="rules" @submit.prevent class="register-form">
            <!-- ── Invite code section ── -->
            <el-form-item label="邀请码" class="invite-item">
              <div class="invite-code-row">
                <el-input
                  v-model="inviteCode"
                  placeholder="请输入管理员提供的邀请码"
                  :disabled="inviteValid === true"
                  @keyup.enter="checkInviteCode"
                >
                  <template v-if="inviteValid === true" #suffix>
                    <el-tag type="success" size="small">有效</el-tag>
                  </template>
                  <template v-if="inviteValid === false" #suffix>
                    <el-tag type="danger" size="small">无效</el-tag>
                  </template>
                </el-input>
                <el-button
                  v-if="inviteValid !== true"
                  type="primary"
                  :loading="inviteValidating"
                  :disabled="!inviteCode.trim()"
                  @click="checkInviteCode"
                >
                  验证
                </el-button>
                <el-button
                  v-else
                  @click="inviteValid = null; inviteMsg = ''; inviteCode = ''"
                >
                  重新输入
                </el-button>
              </div>
              <p v-if="inviteMsg" class="invite-msg" :class="{ 'invite-msg--ok': inviteValid, 'invite-msg--err': !inviteValid }">
                {{ inviteMsg }}
              </p>
            </el-form-item>

            <el-form-item label="用户名" prop="username">
              <el-input v-model="form.username" :disabled="inviteValid !== true" />
            </el-form-item>
            <el-form-item label="邮箱">
              <el-input v-model="form.email" :disabled="inviteValid !== true" />
            </el-form-item>
            <el-form-item label="密码" prop="password">
              <el-input v-model="form.password" type="password" show-password :disabled="inviteValid !== true" />
            </el-form-item>
            <el-button
              type="primary"
              :loading="loading"
              :disabled="inviteValid !== true"
              style="width: 100%"
              @click="submit"
            >
              注册
            </el-button>
          </el-form>
          <p class="auth-foot">已有账号？<RouterLink to="/login">返回登录</RouterLink></p>
        </section>
      </div>
    </section>
  </div>
</template>

<style scoped>
/* ── Compact the register form to fit the fixed auth-shell height ── */

.register-box .auth-kicker {
  margin-bottom: 2px;
}

.register-box .auth-title {
  font-size: 24px;
  margin-bottom: 2px;
}

.register-box .auth-subtitle {
  margin-bottom: 14px;
  font-size: 13px;
}

.register-form :deep(.el-form-item) {
  margin-bottom: 12px;
}

.register-form :deep(.el-form-item__label) {
  margin-bottom: 2px;
  font-size: 13px;
}

/* Match login button spacing — register button has no form-item wrapper */
.register-form > .el-button {
  margin-top: 4px;
}

.register-box .auth-foot {
  margin-top: 10px;
}

/* ── Invite code row ── */
.invite-item {
  margin-bottom: 10px !important;
}

.invite-code-row {
  display: flex;
  gap: 8px;
  width: 100%;
}

.invite-code-row .el-input {
  flex: 1;
}

.invite-msg {
  margin: 2px 0 0;
  font-size: 12px;
  line-height: 1.4;
}

.invite-msg--ok {
  color: var(--el-color-success);
}

.invite-msg--err {
  color: var(--el-color-danger);
}
</style>
