import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import 'element-plus/dist/index.css'
import './styles.css'
import App from './App.vue'
import router from './router'
import { useTheme } from './composables/useTheme'

const bootstrap = async () => {
  useTheme().initTheme()

  const app = createApp(App)
  app.use(createPinia())
  app.use(router)
  app.use(ElementPlus, { locale: zhCn })

  await router.isReady()
  app.mount('#app')
}

void bootstrap()
