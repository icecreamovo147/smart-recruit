import { defineStore } from 'pinia'
import { login, logout, me } from '@/api/auth'
import type { PlatformUser } from '@/types'

const KEY = 'recruitment_platform_user'

const read = (): PlatformUser | null => {
  try { return JSON.parse(localStorage.getItem(KEY) || 'null') as PlatformUser | null } catch { return null }
}
export const clearPlatformUser = () => localStorage.removeItem(KEY)

export const useAuthStore = defineStore('platform-auth', {
  state: () => ({ user: read() as PlatformUser | null }),
  getters: {
    isLoggedIn: (state) => state.user?.account_type === 'platform' && state.user.client_app === 'platform',
    username: (state) => state.user?.username || '',
  },
  actions: {
    persist(user: PlatformUser) { this.user = user; localStorage.setItem(KEY, JSON.stringify(user)) },
    clear() { this.user = null; clearPlatformUser() },
    async signIn(username: string, password: string) { this.persist(await login(username, password)) },
    async restore() {
      try { const user = await me(); this.persist(user); return this.isLoggedIn } catch { this.clear(); return false }
    },
    async signOut() { try { await logout() } finally { this.clear() } },
  },
})
