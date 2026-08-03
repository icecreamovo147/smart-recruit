import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App.vue'

const matchMediaMock = (matches: boolean) =>
  vi.fn().mockImplementation((query: string) => ({
    matches: query === '(prefers-color-scheme: dark)' ? matches : false,
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))

describe('homepage', () => {
  beforeEach(() => {
    window.localStorage.clear()
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: matchMediaMock(false),
    })
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
    })
    document.documentElement.removeAttribute('data-theme')
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('defaults to Chinese and persists an English selection', async () => {
    const wrapper = mount(App)

    expect(wrapper.find('h1').text()).toContain('连接人才与机会')
    expect(wrapper.find('.brand span').text()).toBe('智联招聘')
    expect(wrapper.find('.hero-product-name').text()).toBe('智联招聘')
    expect(wrapper.find('.footer-brand').exists()).toBe(false)
    expect(document.title).toBe('智联招聘 · 智能招聘平台')
    await wrapper.get('[data-testid="language-toggle"]').trigger('click')

    expect(wrapper.find('h1').text()).toContain('Connect talent with opportunity')
    expect(wrapper.find('.brand span').text()).toBe('Smart Recruit')
    expect(wrapper.find('.hero-product-name').text()).toBe('Smart Recruit')
    expect(document.title).toBe('Smart Recruit · Intelligent Recruiting Platform')
    expect(window.localStorage.getItem('smart-recruit-homepage-locale')).toBe('en-US')
    expect(document.documentElement.lang).toBe('en-US')
  })

  it('follows system dark mode once and persists an explicit theme', async () => {
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: matchMediaMock(true),
    })

    const wrapper = mount(App)
    expect(document.documentElement.dataset.theme).toBe('dark')

    await wrapper.get('[data-testid="theme-toggle"]').trigger('click')
    expect(document.documentElement.dataset.theme).toBe('light')
    expect(window.localStorage.getItem('smart-recruit-homepage-theme')).toBe('light')
  })

  it('exposes stable navigation anchors and copies the quick-start command', async () => {
    const wrapper = mount(App)

    expect(wrapper.find('a[href="#capabilities"]').exists()).toBe(true)
    expect(wrapper.find('a[href="#ai"]').exists()).toBe(true)
    expect(wrapper.find('a[href="#architecture"]').exists()).toBe(true)
    expect(wrapper.find('#open-source').exists()).toBe(true)

    await wrapper.get('[data-testid="copy-command"]').trigger('click')
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(expect.stringContaining('./start-dev.sh'))
  })
})
