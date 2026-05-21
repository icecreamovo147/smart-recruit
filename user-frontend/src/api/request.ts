import axios from 'axios'
import type { AxiosError, AxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/router'
import { useAuthStore } from '@/stores/auth'
import { getToken, removeToken, removeUser } from '@/utils/token'
import { BusinessError } from '@/types/api'

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  timeout: 60000,
})

http.interceptors.request.use((config) => {
  const token = getToken()
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

http.interceptors.response.use(
  (response) => {
    const { code, msg, data } = response.data
    if (code !== 0) {
      if (code === 401) {
        removeToken()
        removeUser()
        useAuthStore().$reset()
        router.push('/login')
      }
      const error = new BusinessError(code, friendlyBusinessMessage(code, msg), response.data.request_id)
      ElMessage.error(error.message)
      return Promise.reject(error)
    }
    return data
  },
  (error: AxiosError) => {
    const friendlyMessage = friendlyNetworkMessage(error)
    ElMessage.error(friendlyMessage)
    return Promise.reject(new Error(friendlyMessage))
  },
)

const friendlyBusinessMessage = (code: number, msg: string): string => {
  if (code === 401) return '登录状态已失效，请重新登录'
  if (code === 403 || code === 4030) return '当前账号没有权限执行这个操作'
  if (code === 4001) return '请先完善个人资料后再投递'
  if (code === 4002) return '请先上传简历后再投递'
  if (code === 4003) return '你已经投递过这个岗位'
  if (code === 4004) return '该岗位已下架，无法投递'
  if (code === 404) return '请求的资源不存在或已失效'
  if (code === 499) return '请求已取消，请重新操作'
  if (code === 502) return '第三方服务暂时不可用，请稍后重试'
  if (code === 503) return '后端服务暂不可用，请稍后重试'
  if (code === 504) return '请求处理超时，请稍后重试'
  if (code === 500) return '服务暂时开小差了，请稍后再试'
  return msg || '操作没有成功，请稍后再试'
}

const friendlyNetworkMessage = (error: AxiosError): string => {
  if (error.code === 'ECONNABORTED') return '请求处理超时，请稍后重试'
  if (error.code === 'ERR_NETWORK') return '网络连接失败，请确认后端服务已启动'
  if (error.response && error.response.status >= 500) return '服务器暂时不可用，请稍后再试'
  return '请求失败，请稍后再试'
}

// Typed request interface — the interceptor unwraps response.data,
// so callers get T directly instead of AxiosResponse<T>.
interface RequestInstance {
  get<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T>
  post<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T>
  put<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T>
  patch<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T>
  delete<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T>
}

export default http as unknown as RequestInstance
