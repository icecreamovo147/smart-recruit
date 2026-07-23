import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useTheme } from './useTheme'

describe('platform theme', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.classList.remove('dark')
    delete document.documentElement.dataset.theme
    vi.stubGlobal('matchMedia', vi.fn().mockReturnValue({ matches: true }))
  })

  it('restores, applies, and persists the selected theme', () => {
    localStorage.setItem('recruitment_theme_mode', 'dark')
    const theme = useTheme()

    theme.initTheme()
    expect(theme.isDark.value).toBe(true)
    expect(document.documentElement.dataset.theme).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)

    theme.toggleTheme()
    expect(theme.isDark.value).toBe(false)
    expect(document.documentElement.dataset.theme).toBe('light')
    expect(localStorage.getItem('recruitment_theme_mode')).toBe('light')
  })
})
