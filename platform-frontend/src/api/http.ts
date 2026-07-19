import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/router'
import { clearPlatformUser } from '@/stores/auth'

const http = axios.create({ baseURL: import.meta.env.VITE_API_BASE_URL || '', timeout: 30000, withCredentials: true })

http.interceptors.request.use((config) => {
  config.headers.set('X-Client-App', 'platform')
  return config
})

http.interceptors.response.use(
  (response) => {
    const { code, msg, data } = response.data || {}
    if (code !== 0) {
      ElMessage.error(msg || '操作失败')
      return Promise.reject(new Error(msg || 'operation failed'))
    }
    return data
  },
  async (error) => {
    const request = error.config as typeof error.config & { _retry?: boolean }
    if (error.response?.status === 401 && request && !request._retry) {
      request._retry = true
      const refresh = await fetch(`${import.meta.env.VITE_API_BASE_URL || ''}/api/v1/auth/refresh`, {
        method: 'POST', credentials: 'include', headers: { 'X-Client-App': 'platform' },
      })
      const body = await refresh.json().catch(() => ({}))
      if (body.code === 0) return http(request)
      clearPlatformUser()
      await router.push('/login')
    }
    const msg = error.response?.data?.msg || '请求失败，请稍后重试'
    ElMessage.error(msg)
    return Promise.reject(error)
  },
)

export default http
