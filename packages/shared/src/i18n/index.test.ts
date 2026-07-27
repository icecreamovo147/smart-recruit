import { beforeEach, describe, expect, it, vi } from 'vitest'
import { configureLocale, getLocale, initializeLocale, t } from './index'

describe('shared i18n', () => {
  beforeEach(() => {
    configureLocale('zh-CN')
    vi.restoreAllMocks()
  })

  it('translates registered keys', () => {
    expect(t('common.success')).toBe('成功')
    configureLocale('en-US')
    expect(t('common.success')).toBe('Success')
  })

  it('loads locale from gateway runtime config', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: { locale: 'en-US' } }),
    }))
    await expect(initializeLocale()).resolves.toBe('en-US')
    expect(getLocale()).toBe('en-US')
  })

  it('falls back to Chinese when runtime config is unavailable', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')))
    await expect(initializeLocale()).resolves.toBe('zh-CN')
  })
})
