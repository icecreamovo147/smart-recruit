import type { User } from '@shared/types/domain'

const USER_KEY = 'recruitment_interviewer_user'

export const getUser = (): User | null => {
  try {
    const raw = localStorage.getItem(USER_KEY)
    if (!raw) return null
    return JSON.parse(raw) as User
  } catch {
    return null
  }
}

export const setUser = (user: User): void => {
  try {
    localStorage.setItem(USER_KEY, JSON.stringify(user))
  } catch {
    // localStorage unavailable
  }
}

export const removeUser = (): void => {
  try {
    localStorage.removeItem(USER_KEY)
  } catch {
    // localStorage unavailable
  }
}

export const clearLocalAuthCache = (): void => {
  removeUser()
}
