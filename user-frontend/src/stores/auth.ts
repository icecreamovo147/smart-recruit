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
    accountType: (state: AuthState): string => state.user?.account_type || 'candidate',
  },
  actions: {
    async login(payload: LoginPayload): Promise<void> {
      const data: LoginResponse = await loginApi(payload)
      setUser({
        user_id: data.user_id,
        role: data.role,
        username: data.username,
        account_type: data.account_type,
        roles: data.roles || [],
        permissions: data.permissions || [],
      })
      this.user = getUser()
    },
    logout(): void {
      clearLocalAuthCache()
      this.user = null
    },
    // restoreSession validates the httpOnly cookie against /auth/me and
    // syncs localStorage user state. Returns true if the session is valid.
    // Uses raw fetch to bypass the axios 401→redirect interceptor.
    async restoreSession(): Promise<boolean> {
      try {
        let resp = await fetch(`${import.meta.env.VITE_API_BASE_URL || ''}/api/v1/auth/me`, {
          credentials: 'include',
          headers: { 'X-Client-App': 'candidate' },
        })
        if (resp.status === 401) {
          await silentRefresh('candidate')
          resp = await fetch(`${import.meta.env.VITE_API_BASE_URL || ''}/api/v1/auth/me`, {
            credentials: 'include',
            headers: { 'X-Client-App': 'candidate' },
          })
        }
        if (!resp.ok) { this.logout(); return false }
        const json = await resp.json()
        if (json.code === 0 && json.data) {
          const d = json.data
          setUser({
            user_id: Number(d.user_id),
            role: Number(d.role),
            username: String(d.username),
            account_type: d.account_type ? String(d.account_type) : undefined,
            roles: d.roles ?? undefined,
            permissions: d.permissions ?? undefined,
          })
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
