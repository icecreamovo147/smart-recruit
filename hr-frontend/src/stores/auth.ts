import { defineStore } from 'pinia'
import { login as loginApi } from '@/api/auth'
import { silentRefresh } from '@/api/authRefresh'
import { getUser, setUser, clearLocalAuthCache } from '@/utils/token'
import type { User, LoginPayload, LoginResponse } from '@/types/domain'
import { PERM, ROLE_KEY_RECRUITING_ADMIN, ROLE_KEY_RECRUITER, ROLE_KEY_SYSTEM_ADMIN } from '@/types/domain'

interface AuthState {
  user: User | null
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({ user: getUser() }),

  getters: {
    isLoggedIn: (state: AuthState): boolean => Boolean(state.user),

    // Deprecated: use roles/permissions instead
    role: (state: AuthState): number | undefined => state.user?.role,

    username: (state: AuthState): string => state.user?.username || '',

    accountType: (state: AuthState): string => state.user?.account_type || '',

    roles: (state: AuthState): string[] => state.user?.roles || [],

    permissions: (state: AuthState): string[] => state.user?.permissions || [],

    isStaff: (state: AuthState): boolean => state.user?.account_type === 'staff',

    // Admin role getters
    isRecruitingAdmin(): boolean {
      return this.roles.includes(ROLE_KEY_RECRUITING_ADMIN)
    },
    isSystemAdmin(): boolean {
      return this.roles.includes(ROLE_KEY_SYSTEM_ADMIN)
    },
    isRecruiter(): boolean {
      return this.roles.includes(ROLE_KEY_RECRUITER)
    },
    // Deprecated: for backward compatibility during migration
    isLegacyAdmin(): boolean {
      return this.role === 3 || this.isRecruitingAdmin || this.isSystemAdmin
    },
  },

  actions: {
    hasPermission(perm: string): boolean {
      return this.permissions.includes(perm)
    },

    hasAnyPermission(...perms: string[]): boolean {
      return perms.some((p) => this.permissions.includes(p))
    },

    hasRole(roleKey: string): boolean {
      return this.roles.includes(roleKey)
    },

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
          setUser({
            user_id: Number(json.data.user_id),
            role: Number(json.data.role),
            username: String(json.data.username),
            account_type: json.data.account_type ? String(json.data.account_type) : undefined,
            roles: Array.isArray(json.data.roles) ? json.data.roles.map(String) : [],
            permissions: Array.isArray(json.data.permissions) ? json.data.permissions.map(String) : [],
          })
          this.user = getUser()
          return true
        }
        if (json.code === 401) { this.logout(); return false }
        return false
      } catch (err) {
        console.error('[auth] restoreSession failed:', err)
        return false
      }
    },
  },
})
