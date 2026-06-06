import { computed, ref } from 'vue'

type ThemeMode = 'light' | 'dark'

const STORAGE_KEY = 'recruitment_interviewer_theme_mode'
const THEME_TRANSITION_MS = 500

const themeMode = ref<ThemeMode>('light')
let transitionTimer: ReturnType<typeof setTimeout> | null = null

const isDark = computed(() => themeMode.value === 'dark')

const applyTheme = (mode: ThemeMode, animate = false): void => {
  const root = document.documentElement
  if (transitionTimer) clearTimeout(transitionTimer)
  if (animate) {
    root.classList.add('theme-switching')
    void root.offsetWidth
  }

  document.documentElement.dataset.theme = mode
  if (mode === 'dark') {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }

  if (animate) {
    transitionTimer = setTimeout(() => {
      root.classList.remove('theme-switching')
      transitionTimer = null
    }, THEME_TRANSITION_MS)
  } else {
    root.classList.remove('theme-switching')
  }
}

const resolveSystemPreference = (): ThemeMode =>
  window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'

const readStored = (): ThemeMode | null => {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw === 'dark' || raw === 'light') return raw
  } catch {
    // localStorage unavailable
  }
  return null
}

const persist = (mode: ThemeMode): void => {
  try {
    localStorage.setItem(STORAGE_KEY, mode)
  } catch {
    // localStorage unavailable
  }
}

const initTheme = (): void => {
  const mode = readStored() || resolveSystemPreference()
  themeMode.value = mode
  applyTheme(mode)
}

const setTheme = (mode: ThemeMode): void => {
  const shouldAnimate = themeMode.value !== mode
  themeMode.value = mode
  applyTheme(mode, shouldAnimate)
  persist(mode)
}

const toggleTheme = (): void => {
  setTheme(themeMode.value === 'dark' ? 'light' : 'dark')
}

export const useTheme = () => ({
  themeMode,
  isDark,
  initTheme,
  toggleTheme,
  setTheme,
})
