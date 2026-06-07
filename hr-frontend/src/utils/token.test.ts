import { describe, it, expect, beforeEach } from 'vitest'
import { getUser, setUser, removeUser, clearLocalAuthCache } from '@shared/utils/token'

describe('token utils', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  describe('getUser', () => {
    it('returns null when no user stored', () => {
      expect(getUser()).toBeNull()
    })

    it('returns user when valid data stored', () => {
      const user = { user_id: 1, role: 2, username: 'testuser' }
      localStorage.setItem('recruitment_user', JSON.stringify(user))
      const result = getUser()
      expect(result).not.toBeNull()
      expect(result!.user_id).toBe(1)
      expect(result!.role).toBe(2)
      expect(result!.username).toBe('testuser')
    })

    it('returns null and clears storage for invalid data', () => {
      localStorage.setItem('recruitment_user', JSON.stringify({ foo: 'bar' }))
      expect(getUser()).toBeNull()
      expect(localStorage.getItem('recruitment_user')).toBeNull()
    })

    it('returns null for corrupt JSON', () => {
      localStorage.setItem('recruitment_user', 'not-json{')
      expect(getUser()).toBeNull()
    })

    it('parses numeric user_id from string', () => {
      const user = { user_id: '42', role: '1', username: 'alice' }
      localStorage.setItem('recruitment_user', JSON.stringify(user))
      const result = getUser()
      expect(result!.user_id).toBe(42)
      expect(result!.role).toBe(1)
    })

    it('preserves optional roles and permissions arrays', () => {
      const user = {
        user_id: 1,
        role: 3,
        username: 'admin',
        account_type: 'staff',
        roles: ['recruiter', 'recruiting_admin'],
        permissions: ['job.read', 'job.create'],
      }
      localStorage.setItem('recruitment_user', JSON.stringify(user))
      const result = getUser()
      expect(result!.account_type).toBe('staff')
      expect(result!.roles).toEqual(['recruiter', 'recruiting_admin'])
      expect(result!.permissions).toEqual(['job.read', 'job.create'])
    })
  })

  describe('setUser', () => {
    it('stores user in localStorage', () => {
      const user = { user_id: 1, role: 1, username: 'bob' } as any
      setUser(user)
      const stored = JSON.parse(localStorage.getItem('recruitment_user')!)
      expect(stored.username).toBe('bob')
    })
  })

  describe('removeUser', () => {
    it('removes user from localStorage', () => {
      localStorage.setItem('recruitment_user', JSON.stringify({ user_id: 1, role: 1, username: 'x' }))
      removeUser()
      expect(localStorage.getItem('recruitment_user')).toBeNull()
    })
  })

  describe('clearLocalAuthCache', () => {
    it('removes both user and legacy token', () => {
      localStorage.setItem('recruitment_user', JSON.stringify({ user_id: 1, role: 1, username: 'x' }))
      localStorage.setItem('recruitment_token', 'stale-token')
      clearLocalAuthCache()
      expect(localStorage.getItem('recruitment_user')).toBeNull()
      expect(localStorage.getItem('recruitment_token')).toBeNull()
    })
  })
})
