import { defineStore } from 'pinia'
import { login as loginApi } from '@/api/auth'
import { silentRefresh } from '@/api/authRefresh'
import { getUser, setUser, clearLocalAuthCache } from '@/utils/token'
import type { User, LoginPayload, LoginResponse } from '@/types/domain'

interface AuthState {
  user: User | null
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({ user: getUser() }),
  getters: {
    isLoggedIn: (state: AuthState): boolean => Boolean(state.user),
    role: (state: AuthState): number | undefined => state.user?.role,
    username: (state: AuthState): string => state.user?.username || '',
  },
  actions: {
    async login(payload: LoginPayload): Promise<void> {
      const data: LoginResponse = await loginApi(payload)
      setUser({ user_id: data.user_id, role: data.role, username: data.username })
      this.user = getUser()
    },
    logout(): void {
      clearLocalAuthCache()
      this.user = null
    },
    async restoreSession(): Promise<boolean> {
      try {
        let resp = await fetch(`${import.meta.env.VITE_API_BASE_URL || ''}/api/v1/auth/me`, {
          credentials: 'include',
          headers: { 'X-Client-App': 'hr' },
        })
        if (resp.status === 401) {
          await silentRefresh('hr')
          resp = await fetch(`${import.meta.env.VITE_API_BASE_URL || ''}/api/v1/auth/me`, {
            credentials: 'include',
            headers: { 'X-Client-App': 'hr' },
          })
        }
        if (!resp.ok) { this.logout(); return false }
        const json = await resp.json()
        if (json.code === 0 && json.data) {
          setUser({ user_id: Number(json.data.user_id), role: Number(json.data.role), username: String(json.data.username) })
          this.user = getUser()
          return true
        }
        if (json.code === 401) { this.logout(); return false }
        return false
      } catch {
        return false
      }
    },
  },
})
