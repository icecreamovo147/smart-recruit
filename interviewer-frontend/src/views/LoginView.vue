<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Lock, Moon, Sunny } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import { useTheme } from '@/composables/useTheme'
import logoSmallLight from '@/assets/logo-small.png'
import logoSmallDark from '@/assets/logo-small-dark.png'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { isDark, toggleTheme } = useTheme()
const brandLogo = computed(() => isDark.value ? logoSmallDark : logoSmallLight)

const loading = ref(false)
const form = reactive({
  username: '',
  password: '',
})

const showError = ref(false)
const errorMessage = ref('')

// Show error from query params
if (route.query.error === 'not_interviewer') {
  showError.value = true
  errorMessage.value = '请使用面试官账号登录'
}

const handleLogin = async () => {
  if (!form.username.trim()) {
    ElMessage.warning('请输入用户名')
    return
  }
  if (!form.password || form.password.length < 6) {
    ElMessage.warning('密码长度至少 6 位')
    return
  }

  loading.value = true
  showError.value = false
  try {
    await auth.login({ username: form.username, password: form.password })

    // Verify interviewer role after login
    if (!auth.isInterviewer) {
      await auth.logoutApi()
      auth.logout()
      showError.value = true
      errorMessage.value = '当前账号不是面试官账号，无法登录面试官工作台'
      return
    }

    const redirect = route.query.redirect as string
    router.push(redirect || '/dashboard')
  } catch (error: unknown) {
    showError.value = true
    errorMessage.value = error instanceof Error ? error.message : '登录失败，请检查用户名和密码'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page" :class="{ 'auth-page--dark': isDark }">
    <section class="auth-shell">
      <div class="auth-brand-panel">
        <div class="auth-logo-frame">
          <img class="auth-logo-mark" :src="brandLogo" alt="智联招聘" />
          <div class="auth-logo-copy">
            <strong>智联招聘</strong>
            <span>Interviewer Workspace</span>
          </div>
        </div>

        <div class="auth-panel-copy">
          <p class="auth-kicker">Enterprise Interview Suite</p>
          <h1>让每一次面试，<br />都有清晰的准备与结论</h1>
          <p>集中查看面试安排、候选人材料，并在统一流程中提交专业反馈。</p>
        </div>

        <div class="auth-metrics" aria-label="面试官工作台能力">
          <div><strong>日程</strong><span>统一面试安排</span></div>
          <div><strong>资料</strong><span>候选人上下文</span></div>
          <div><strong>反馈</strong><span>结构化结论</span></div>
        </div>
      </div>

      <div class="auth-form-panel">
        <div class="auth-panel-top">
          <span>面试官端</span>
          <el-tooltip :content="isDark ? '切换日间模式' : '切换夜间模式'" placement="bottom">
            <button
              class="theme-button"
              type="button"
              @click="toggleTheme"
              :aria-label="isDark ? '切换到亮色模式' : '切换到暗色模式'"
            >
              <el-icon :size="18"><Sunny v-if="isDark" /><Moon v-else /></el-icon>
            </button>
          </el-tooltip>
        </div>

        <section class="auth-box">
          <p class="auth-kicker">Interviewer Workspace</p>
          <h2 class="auth-title">登录面试官工作台</h2>
          <p class="auth-subtitle">使用企业分配的面试官账号继续。</p>

          <el-alert
            v-if="showError"
            :title="errorMessage"
            type="error"
            show-icon
            :closable="true"
            class="login-alert"
            @close="showError = false"
          />

          <el-form :model="form" label-position="top" class="login-form" @submit.prevent="handleLogin">
            <el-form-item label="用户名">
              <el-input
                v-model="form.username"
                placeholder="请输入用户名"
                size="large"
                autocomplete="username"
                @keyup.enter="handleLogin"
              />
            </el-form-item>
            <el-form-item label="密码">
              <el-input
                v-model="form.password"
                type="password"
                placeholder="请输入密码"
                size="large"
                show-password
                autocomplete="current-password"
                @keyup.enter="handleLogin"
              />
            </el-form-item>
            <el-button
              type="primary"
              size="large"
              :loading="loading"
              class="login-submit"
              @click="handleLogin"
            >
              登录
            </el-button>
          </el-form>

          <p class="auth-foot">
            <el-icon><Lock /></el-icon>
            <span>账号由企业招聘管理员统一开通</span>
          </p>
        </section>
      </div>
    </section>
  </div>
</template>

<style scoped>
.auth-page {
  height: 100vh;
  height: 100dvh;
  padding: clamp(20px, 4vw, 40px);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  background: var(--app-bg);
}

.auth-shell {
  width: min(1080px, 100%);
  height: min(640px, calc(100dvh - 80px));
  min-height: min(640px, calc(100dvh - 80px));
  display: grid;
  grid-template-columns: minmax(0, 1.08fr) minmax(380px, 0.92fr);
  overflow: hidden;
  border: 1px solid var(--border-primary);
  border-radius: 8px;
  background: var(--surface-primary);
  box-shadow: 0 18px 48px rgba(23, 32, 51, 0.12);
}

.auth-brand-panel {
  min-width: 0;
  min-height: 100%;
  padding: 44px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  color: #17345f;
  background: #eaf2ff;
}

.auth-logo-frame {
  display: flex;
  align-items: center;
  gap: 12px;
}

.auth-logo-mark {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  object-fit: cover;
}

.auth-logo-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.auth-logo-copy strong {
  font-size: 22px;
  line-height: 1.3;
  letter-spacing: 0.12em;
}

.auth-logo-copy span {
  color: #607797;
  font-size: 11px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

.auth-panel-copy {
  max-width: 500px;
}

.auth-kicker {
  margin: 0 0 16px;
  color: var(--brand-primary);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.auth-panel-copy h1 {
  margin: 0;
  font-size: 38px;
  font-weight: 600;
  line-height: 1.2;
  letter-spacing: -0.02em;
}

.auth-panel-copy p:last-child {
  max-width: 460px;
  margin: 18px 0 0;
  color: #526b8d;
  font-size: 16px;
  line-height: 1.8;
}

.auth-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.auth-metrics div {
  min-height: 84px;
  padding: 14px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  border: 1px solid rgba(37, 99, 235, 0.16);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.5);
}

.auth-metrics strong {
  font-size: 20px;
  line-height: 1.2;
}

.auth-metrics span {
  margin-top: 6px;
  color: #607797;
  font-size: 13px;
}

.auth-form-panel {
  min-width: 0;
  padding: 34px 42px;
  display: flex;
  flex-direction: column;
  background: var(--surface-primary);
}

.auth-panel-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--text-muted);
  font-size: 14px;
  font-weight: 600;
}

.theme-button {
  width: 36px;
  height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  background: var(--surface-primary);
  cursor: pointer;
  transition: color 120ms ease, border-color 120ms ease, background-color 120ms ease;
}

.theme-button:hover {
  color: var(--brand-primary);
  border-color: var(--brand-primary);
  background: var(--brand-soft);
}

.auth-box {
  width: 100%;
  max-width: 390px;
  margin: auto;
}

.auth-title {
  margin: 0;
  color: var(--text-primary);
  font-size: 28px;
  font-weight: 600;
  line-height: 1.25;
}

.auth-subtitle {
  margin: 8px 0 28px;
  color: var(--text-muted);
  font-size: 14px;
  line-height: 1.7;
}

.login-alert {
  margin-bottom: 22px;
}

.login-form {
  display: flex;
  flex-direction: column;
}

.login-form :deep(.el-form-item) {
  margin-bottom: 22px;
}

.login-form :deep(.el-form-item__label) {
  padding-bottom: 8px;
  color: var(--text-secondary);
  font-size: 14px;
  font-weight: 600;
}

.login-form :deep(.el-input__wrapper) {
  min-height: 46px;
  border-radius: var(--radius-sm);
  background: var(--surface-primary);
  box-shadow: 0 0 0 1px var(--border-primary) inset;
}

.login-form :deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px var(--text-muted) inset;
}

.login-form :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px var(--brand-primary) inset, 0 0 0 3px var(--brand-soft);
}

.login-form :deep(.el-button) {
  min-height: 46px;
  border-radius: var(--radius-sm);
  font-weight: 600;
}

.login-submit {
  width: 100%;
  margin-top: 2px;
}

.auth-foot {
  margin: 20px 0 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.6;
}

.auth-page--dark .auth-brand-panel {
  background: #0a1c3a;
  color: #f4f7fb;
}

.auth-page--dark .auth-logo-copy span,
.auth-page--dark .auth-panel-copy p:last-child,
.auth-page--dark .auth-metrics span {
  color: rgba(226, 232, 240, 0.72);
}

.auth-page--dark .auth-brand-panel .auth-kicker {
  color: #93c5fd;
}

.auth-page--dark .auth-metrics div {
  border-color: rgba(148, 163, 184, 0.25);
  background: rgba(15, 23, 42, 0.34);
}

.auth-page--dark .login-form :deep(.el-input__wrapper) {
  background: var(--surface-secondary);
}

@media (max-height: 760px) and (min-width: 769px) {
  .auth-shell {
    height: calc(100dvh - 40px);
  }

  .auth-brand-panel {
    padding: 30px 36px;
  }

  .auth-panel-copy h1 {
    font-size: 33px;
  }
}

@media (max-width: 768px) {
  .auth-page {
    padding: 16px;
  }

  .auth-shell {
    height: calc(100dvh - 32px);
    min-height: 0;
    grid-template-columns: 1fr;
  }

  .auth-brand-panel {
    display: none;
  }

  .auth-form-panel {
    min-height: 0;
    padding: 22px 18px 28px;
  }

  .auth-box {
    max-width: none;
    margin: auto;
  }
}

@media (max-width: 480px) {
  .auth-title {
    font-size: 25px;
  }
}
</style>
