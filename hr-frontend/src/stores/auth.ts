import { t } from '@shared/i18n'
import { defineStore } from 'pinia'
import { login as loginApi, switchTenant as switchTenantApi } from '@/api/auth'
import { silentRefresh } from '@/api/authRefresh'
import { getUser, setUser, clearLocalAuthCache } from '@/utils/token'
import { normalizeTenantMemberships, normalizeTenantUser } from '@/utils/tenantMembership'
import type { User, LoginPayload, LoginResponse } from '@/types/domain'
import { PERM, ROLE_KEY_RECRUITING_ADMIN, ROLE_KEY_RECRUITER, ROLE_KEY_SYSTEM_ADMIN } from '@/types/domain'

interface AuthState {
  user: User | null
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({ user: normalizeTenantUser(getUser()) }),

  getters: {
    isLoggedIn: (state: AuthState): boolean => Boolean(state.user),

    // Deprecated: use roles/permissions instead
    role: (state: AuthState): number | undefined => state.user?.role,

    username: (state: AuthState): string => state.user?.username || '',

    email: (state: AuthState): string => state.user?.email || '',

    accountType: (state: AuthState): string => state.user?.account_type || '',

    roles: (state: AuthState): string[] => state.user?.roles || [],

    permissions: (state: AuthState): string[] => state.user?.permissions || [],

    tenantId: (state: AuthState): number => state.user?.tenant_id || 0,

    membershipId: (state: AuthState): number => state.user?.membership_id || 0,

    memberships: (state: AuthState) => state.user?.memberships || [],

    activeTenant: (state: AuthState) => (state.user?.memberships || []).find((item) => item.tenant_id === state.user?.tenant_id),

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
      const user = normalizeTenantUser({
        user_id: data.user_id,
        role: data.role,
        username: data.username,
        account_type: data.account_type,
        roles: data.roles || [],
        permissions: data.permissions || [],
        email: data.email,
        tenant_id: data.tenant_id,
        membership_id: data.membership_id,
        client_app: data.client_app,
        available_apps: data.available_apps || [],
        memberships: normalizeTenantMemberships(data.memberships),
      })
      if (!user) throw new Error(t('common.unauthenticated'))
      setUser(user)
      this.user = user
    },

    async switchTenant(tenantId: number): Promise<void> {
      if (!tenantId || tenantId === this.tenantId) return
      await switchTenantApi(tenantId)
      if (!await this.restoreSession()) {
        throw new Error(t('common.refresh_failed'))
      }
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
            email: json.data.email ? String(json.data.email) : undefined,
            tenant_id: json.data.tenant_id ? Number(json.data.tenant_id) : undefined,
            membership_id: json.data.membership_id ? Number(json.data.membership_id) : undefined,
            client_app: json.data.client_app ? String(json.data.client_app) : undefined,
            available_apps: Array.isArray(json.data.available_apps) ? json.data.available_apps.map(String) : [],
            memberships: normalizeTenantMemberships(json.data.memberships),
          })
          this.user = normalizeTenantUser(getUser())
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
