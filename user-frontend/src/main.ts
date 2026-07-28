import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import en from 'element-plus/es/locale/lang/en'
import 'element-plus/dist/index.css'
import './styles/fonts.css'
import './styles.css'
import App from './App.vue'
import router from './router'
import { useTheme } from '@/composables/useTheme'
import { initializeLocale } from '@shared/i18n'

const bootstrap = async () => {
  useTheme().initTheme()
  const locale = await initializeLocale(import.meta.env.VITE_API_BASE_URL || '')

  createApp(App)
    .use(createPinia())
    .use(router)
    .use(ElementPlus, { locale: locale === 'en-US' ? en : zhCn })
    .mount('#app')
}

void bootstrap()
