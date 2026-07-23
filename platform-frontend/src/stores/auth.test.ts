import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { logout } from '@/api/auth'
import { PLATFORM_PERMISSIONS } from '@/permissions'
import { useAuthStore } from './auth'
import type { PlatformUser } from '@/types'

vi.mock('@/api/auth', () => ({
  login: vi.fn(),
  logout: vi.fn(),
  me: vi.fn(),
}))

const platformUser = (overrides: Partial<PlatformUser> = {}): PlatformUser => ({
  user_id: 1,
  username: 'operator',
  account_type: 'platform',
  roles: ['platform_operator'],
  permissions: [PLATFORM_PERMISSIONS.TENANT_READ],
  client_app: 'platform',
  available_apps: ['platform'],
  ...overrides,
})

describe('platform auth store', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.mocked(logout).mockReset()
    setActivePinia(createPinia())
  })

  it('admits only a platform principal with a platform role', () => {
    const store = useAuthStore()
    store.persist(platformUser())
    expect(store.isLoggedIn).toBe(true)

    store.persist(platformUser({ account_type: 'staff', client_app: 'staff' }))
    expect(store.isLoggedIn).toBe(false)
  })

  it('uses explicit permissions instead of role hierarchy', () => {
    const store = useAuthStore()
    store.persist(platformUser())
    expect(store.can(PLATFORM_PERMISSIONS.TENANT_READ)).toBe(true)
    expect(store.can(PLATFORM_PERMISSIONS.USER_MANAGE)).toBe(false)
    expect(store.can(PLATFORM_PERMISSIONS.PLAN_PUBLISH)).toBe(false)
  })

  it('persists and clears the platform session snapshot', () => {
    const store = useAuthStore()
    store.persist(platformUser({ username: 'auditor', roles: ['platform_auditor'] }))
    expect(JSON.parse(localStorage.getItem('recruitment_platform_user') || '{}').username).toBe('auditor')
    store.clear()
    expect(store.user).toBeNull()
    expect(localStorage.getItem('recruitment_platform_user')).toBeNull()
  })

  it('clears the session only after server logout succeeds', async () => {
    vi.mocked(logout).mockResolvedValue(undefined)
    const store = useAuthStore()
    store.persist(platformUser())

    await store.signOut()

    expect(store.user).toBeNull()
    expect(store.consumeLoginRestoreSuppression()).toBe(true)
    expect(store.consumeLoginRestoreSuppression()).toBe(false)
  })

  it('keeps the current session when server logout fails', async () => {
    vi.mocked(logout).mockRejectedValue(new Error('network unavailable'))
    const store = useAuthStore()
    store.persist(platformUser())

    await expect(store.signOut()).rejects.toThrow('network unavailable')

    expect(store.user?.username).toBe('operator')
    expect(localStorage.getItem('recruitment_platform_user')).not.toBeNull()
    expect(store.consumeLoginRestoreSuppression()).toBe(false)
  })
})
