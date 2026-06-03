import type { User } from '../types/domain'

const USER_KEY = 'recruitment_user'

export const getUser = (): User | null => {
  try {
    const raw = localStorage.getItem(USER_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw)
    const userId = Number(parsed?.user_id)
    const role = Number(parsed?.role)
    if (!Number.isInteger(userId) || !Number.isInteger(role) || !parsed?.username) {
      removeUser()
      return null
    }
    return {
      user_id: userId,
      role,
      username: String(parsed.username),
      account_type: parsed.account_type ? String(parsed.account_type) : undefined,
      roles: Array.isArray(parsed.roles) ? parsed.roles.map(String) : undefined,
      permissions: Array.isArray(parsed.permissions) ? parsed.permissions.map(String) : undefined,
    } as User
  } catch {
    removeUser()
    return null
  }
}

export const setUser = (user: User): void =>
  localStorage.setItem(USER_KEY, JSON.stringify(user))

export const removeUser = (): void => localStorage.removeItem(USER_KEY)

// removeLegacyToken clears any stale localStorage token from the legacy httpOnly-less era.
// Kept only for cleanup; the current system uses httpOnly cookies for auth.
export const removeLegacyToken = (): void => localStorage.removeItem('recruitment_token')

// clearLocalAuthCache wipes all local auth state — call on logout and refresh failure.
export const clearLocalAuthCache = (): void => {
  removeLegacyToken()
  removeUser()
}
