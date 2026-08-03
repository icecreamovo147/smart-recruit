<script setup lang="ts">
import { t } from '@shared/i18n'
import { computed, reactive, ref } from 'vue'
import { Lock, Moon, Sunny, User } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useTheme } from '@/composables/useTheme'
import heroImageLight from '@/assets/platform-login-hero.png'
import heroImageDark from '@/assets/platform-login-hero-dark.png'
import brandLogoLight from '@shared/assets/logo-small.webp'
import brandLogoDark from '@shared/assets/logo-small-dark.webp'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { isDark, isThemeTransitioning, toggleTheme } = useTheme()

const formRef = ref<FormInstance>()
const loading = ref(false)
const form = reactive({ username: '', password: '' })
const brandLogo = computed(() => isDark.value ? brandLogoDark : brandLogoLight)
const heroImage = computed(() => isDark.value ? heroImageDark : heroImageLight)
const rules: FormRules<typeof form> = {
  username: [{ required: true, message: '请输入平台管理员账号', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
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
    await auth.signIn(form.username.trim(), form.password)
    if (!auth.isLoggedIn) {
      await auth.signOut()
      ElMessage.error(t('frontend.operation_failed'))
      return
    }
    const requestedPath = typeof route.query.redirect === 'string' && route.query.redirect.startsWith('/')
      ? route.query.redirect
      : '/'
    await router.replace(requestedPath)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="platform-login">
    <section class="platform-login__story" aria-label="平台运营控制台介绍">
      <img class="platform-login__hero" :src="heroImage" alt="" aria-hidden="true" />
      <div class="platform-login__story-content">
        <div class="platform-login__brand">
          <img :src="brandLogo" alt="Smart Recruit" />
          <div>
            <strong>Smart Recruit</strong>
            <span>PLATFORM CONSOLE</span>
          </div>
        </div>

        <div class="platform-login__message">
          <p class="platform-login__kicker">PLATFORM OPERATIONS</p>
          <h1>平台运营控制台</h1>
          <span class="platform-login__accent" aria-hidden="true"></span>
          <p>统一治理租户、平台身份与关键运营风险</p>
        </div>
      </div>
    </section>

    <section class="platform-login__access">
      <header class="platform-login__access-top">
        <span>安全访问</span>
        <el-tooltip :content="isDark ? '切换到亮色模式' : '切换到暗色模式'" placement="bottom">
          <button
            class="platform-login__theme"
            type="button"
            :disabled="isThemeTransitioning"
            :aria-label="isDark ? '切换到亮色模式' : '切换到暗色模式'"
            @click="toggleTheme"
          >
            <el-icon><Sunny v-if="isDark" /><Moon v-else /></el-icon>
            <span>{{ isDark ? '亮色模式' : '暗色模式' }}</span>
          </button>
        </el-tooltip>
      </header>

      <main class="platform-login__form-wrap">
        <div class="platform-login__form-heading">
          <p class="platform-login__kicker">PLATFORM ACCESS</p>
          <h2>平台管理员账号</h2>
          <p>使用授权的平台账号进入统一运营与治理工作区。</p>
        </div>

        <el-form
          ref="formRef"
          class="platform-login__form"
          :model="form"
          :rules="rules"
          label-position="top"
          @submit.prevent="submit"
        >
          <el-form-item label="平台管理员账号" prop="username">
            <el-input
              v-model="form.username"
              autocomplete="username"
              placeholder="请输入平台管理员账号"
              size="large"
            >
              <template #prefix><el-icon><User /></el-icon></template>
            </el-input>
          </el-form-item>
          <el-form-item label="密码" prop="password">
            <el-input
              v-model="form.password"
              type="password"
              autocomplete="current-password"
              placeholder="请输入密码"
              show-password
              size="large"
            >
              <template #prefix><el-icon><Lock /></el-icon></template>
            </el-input>
          </el-form-item>
          <el-button
            class="platform-login__submit"
            type="primary"
            size="large"
            native-type="submit"
            :loading="loading"
          >
            安全登录
          </el-button>
        </el-form>

        <p class="platform-login__notice">
          <el-icon><Lock /></el-icon>
          <span>仅限授权的平台管理员、操作员与审计人员使用</span>
        </p>
      </main>

      <footer class="platform-login__access-foot">Smart Recruit Platform · Secure Access</footer>
    </section>
  </div>
</template>
