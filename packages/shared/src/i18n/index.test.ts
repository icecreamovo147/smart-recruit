import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { configureLocale, getLocale, initializeLocale, t } from './index'

describe('shared i18n', () => {
  beforeEach(() => {
    configureLocale('zh-CN')
    vi.restoreAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('translates registered keys', () => {
    expect(t('common.success')).toBe('成功')
    configureLocale('en-US')
    expect(t('common.success')).toBe('Success')
  })

  it('interpolates the effective model in fallback messages for both locales', () => {
    expect(t('ai.model_fallback', { model: 'GPT-4.1' })).toBe('请求的模型不可用，已自动切换至 GPT-4.1')
    configureLocale('en-US')
    expect(t('ai.model_fallback', { model: 'GPT-4.1' })).toBe(
      'The requested model is unavailable. Switched automatically to GPT-4.1.',
    )
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

  it('aborts a hanging runtime config request and falls back within the timeout', async () => {
    vi.useFakeTimers()
    const onAbort = vi.fn()
    vi.stubGlobal('fetch', vi.fn((_input: RequestInfo | URL, init?: RequestInit) => {
      init?.signal?.addEventListener('abort', onAbort)
      return new Promise<Response>(() => undefined)
    }))

    const result = initializeLocale()
    await vi.advanceTimersByTimeAsync(5_000)

    await expect(result).resolves.toBe('zh-CN')
    expect(onAbort).toHaveBeenCalledOnce()
  })
})
