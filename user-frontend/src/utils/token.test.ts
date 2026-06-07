import { describe, it, expect, beforeEach } from 'vitest'
import { getUser, setUser, removeUser, clearLocalAuthCache } from '@shared/utils/token'

describe('token utils (user-frontend)', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('returns null when no user stored', () => {
    expect(getUser()).toBeNull()
  })

  it('stores and retrieves a candidate user', () => {
    const user = { user_id: 7, role: 1, username: 'candidate1' } as any
    setUser(user)
    const result = getUser()
    expect(result).not.toBeNull()
    expect(result!.user_id).toBe(7)
    expect(result!.role).toBe(1)
    expect(result!.username).toBe('candidate1')
  })

  it('clears storage on invalid data', () => {
    localStorage.setItem('recruitment_user', JSON.stringify({ bad: true }))
    expect(getUser()).toBeNull()
  })

  it('removeUser clears the stored user', () => {
    setUser({ user_id: 1, role: 1, username: 'x' } as any)
    removeUser()
    expect(getUser()).toBeNull()
  })

  it('clearLocalAuthCache removes all auth state', () => {
    setUser({ user_id: 1, role: 1, username: 'x' } as any)
    localStorage.setItem('recruitment_token', 'old')
    clearLocalAuthCache()
    expect(getUser()).toBeNull()
    expect(localStorage.getItem('recruitment_token')).toBeNull()
  })
})
