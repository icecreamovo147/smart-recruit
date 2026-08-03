import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  configureLocale,
  getLocale,
  initializeLocale,
  localizedBackendText,
  t,
} from '@shared/i18n'

describe('shared runtime i18n', () => {
  beforeEach(() => {
    configureLocale('zh-CN')
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('switches the generated catalog without cross-language fallback', () => {
    expect(t('common.success')).toBe('成功')
    configureLocale('en-US')
    expect(t('common.success')).toBe('Success')
  })

  it('loads the deployment locale before application mount', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: { locale: 'en-US' } }),
    }))

    await expect(initializeLocale()).resolves.toBe('en-US')
    expect(getLocale()).toBe('en-US')
  })

  it('uses the Chinese default when runtime configuration is unavailable', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')))
    await expect(initializeLocale()).resolves.toBe('zh-CN')
  })

  it('localizes known backend message keys and preserves unknown diagnostic text', () => {
    expect(localizedBackendText('common.operation_failed')).toBe('操作没有成功，请稍后再试')
    expect(localizedBackendText('upstream provider disconnected')).toBe('upstream provider disconnected')
    expect(localizedBackendText(undefined)).toBe('')
  })
})
