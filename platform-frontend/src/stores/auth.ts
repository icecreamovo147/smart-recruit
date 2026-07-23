import { defineStore } from 'pinia'
import { login, logout, me } from '@/api/auth'
import type { PlatformUser } from '@/types'
import { PLATFORM_ROLES } from '@/permissions'

const KEY = 'recruitment_platform_user'

const read = (): PlatformUser | null => {
  try { return JSON.parse(localStorage.getItem(KEY) || 'null') as PlatformUser | null } catch { return null }
}
export const clearPlatformUser = () => localStorage.removeItem(KEY)

export const useAuthStore = defineStore('platform-auth', {
  state: () => ({
    user: read() as PlatformUser | null,
    suppressNextLoginRestore: false,
  }),
  getters: {
    isLoggedIn: (state) => state.user?.account_type === 'platform' && state.user.client_app === 'platform' && state.user.roles.some((role) => PLATFORM_ROLES.includes(role as typeof PLATFORM_ROLES[number])),
    username: (state) => state.user?.username || '',
    can: (state) => (permission: string) => Boolean(state.user?.permissions.includes(permission)),
  },
  actions: {
    persist(user: PlatformUser) { this.user = user; localStorage.setItem(KEY, JSON.stringify(user)) },
    clear() { this.user = null; clearPlatformUser() },
    async signIn(username: string, password: string) { this.persist(await login(username, password)) },
    async restore() {
      try { const user = await me(); this.persist(user); return this.isLoggedIn } catch { this.clear(); return false }
    },
    async signOut() {
      await logout()
      this.clear()
      this.suppressNextLoginRestore = true
    },
    consumeLoginRestoreSuppression() {
      const suppressed = this.suppressNextLoginRestore
      this.suppressNextLoginRestore = false
      return suppressed
    },
  },
})
