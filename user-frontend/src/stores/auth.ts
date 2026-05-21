import { defineStore } from 'pinia'
import { login as loginApi } from '@/api/auth'
import { getToken, getUser, removeToken, removeUser, setToken, setUser } from '@/utils/token'
import type { User, LoginPayload, LoginResponse } from '@/types/domain'

interface AuthState {
  token: string | null
  user: User | null
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({ token: getToken(), user: getUser() }),
  getters: {
    isLoggedIn: (state: AuthState): boolean => Boolean(state.token),
    role: (state: AuthState): number | undefined => state.user?.role,
    username: (state: AuthState): string => state.user?.username || '',
  },
  actions: {
    async login(payload: LoginPayload): Promise<void> {
      const data: LoginResponse = await loginApi(payload)
      setToken(data.token)
      setUser({ user_id: data.user_id, role: data.role, username: data.username })
      this.token = data.token
      this.user = getUser()
    },
    logout(): void {
      removeToken()
      removeUser()
      this.token = ''
      this.user = null
    },
  },
})
