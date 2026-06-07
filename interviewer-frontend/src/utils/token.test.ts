import { describe, it, expect, beforeEach } from 'vitest'
import { getUser, setUser, removeUser, clearLocalAuthCache } from '@/utils/token'

describe('interviewer token utils', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  describe('getUser', () => {
    it('returns null when no user stored', () => {
      expect(getUser()).toBeNull()
    })

    it('returns user when valid data stored', () => {
      const user = { user_id: 10, role: 4, username: 'interviewer1' } as any
      localStorage.setItem('recruitment_interviewer_user', JSON.stringify(user))
      const result = getUser()
      expect(result).not.toBeNull()
      expect(result!.user_id).toBe(10)
      expect(result!.username).toBe('interviewer1')
    })

    it('returns null for corrupt JSON', () => {
      localStorage.setItem('recruitment_interviewer_user', 'invalid-json')
      expect(getUser()).toBeNull()
    })
  })

  describe('setUser', () => {
    it('stores user in localStorage', () => {
      const user = { user_id: 5, role: 4, username: 'test' } as any
      setUser(user)
      const stored = JSON.parse(localStorage.getItem('recruitment_interviewer_user')!)
      expect(stored.username).toBe('test')
    })
  })

  describe('removeUser', () => {
    it('removes user from localStorage', () => {
      localStorage.setItem('recruitment_interviewer_user', JSON.stringify({ user_id: 1, role: 4, username: 'x' }))
      removeUser()
      expect(localStorage.getItem('recruitment_interviewer_user')).toBeNull()
    })
  })

  describe('clearLocalAuthCache', () => {
    it('clears interviewer auth state', () => {
      localStorage.setItem('recruitment_interviewer_user', JSON.stringify({ user_id: 1, role: 4, username: 'x' }))
      clearLocalAuthCache()
      expect(localStorage.getItem('recruitment_interviewer_user')).toBeNull()
    })
  })
})
