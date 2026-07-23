import { computed, ref } from 'vue'

type ThemeMode = 'light' | 'dark'

const STORAGE_KEY = 'recruitment_theme_mode'
const OVERLAY_FADE_IN_MS = 280
const OVERLAY_HOLD_MS = 1500
const OVERLAY_FADE_OUT_MS = 280

const themeMode = ref<ThemeMode>('light')
const isThemeTransitioning = ref(false)

let enterTimer: ReturnType<typeof setTimeout> | null = null
let holdTimer: ReturnType<typeof setTimeout> | null = null
let fadeTimer: ReturnType<typeof setTimeout> | null = null
let overlayEl: HTMLDivElement | null = null
let transitionGeneration = 0

const isDark = computed(() => themeMode.value === 'dark')

const prefersReducedMotion = (): boolean => {
  try {
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches
  } catch {
    return false
  }
}

const clearTimers = (): void => {
  if (enterTimer) clearTimeout(enterTimer)
  if (holdTimer) clearTimeout(holdTimer)
  if (fadeTimer) clearTimeout(fadeTimer)
  enterTimer = null
  holdTimer = null
  fadeTimer = null
}

const ensureOverlay = (): HTMLDivElement => {
  if (overlayEl && document.body.contains(overlayEl)) return overlayEl
  const element = document.createElement('div')
  element.id = 'theme-switch-overlay'
  element.className = 'theme-switch-overlay'
  element.setAttribute('aria-hidden', 'true')
  element.innerHTML = `
    <div class="theme-switch-overlay__scrim" aria-hidden="true"></div>
    <div class="theme-switch-overlay__panel" role="status" aria-live="polite">
      <div class="theme-switch-overlay__spinner" aria-hidden="true">
        <span class="theme-switch-overlay__ring"></span>
        <span class="theme-switch-overlay__ring"></span>
        <span class="theme-switch-overlay__ring"></span>
      </div>
      <p class="theme-switch-overlay__text">正在切换主题…</p>
    </div>
  `
  document.body.appendChild(element)
  overlayEl = element
  return element
}

const hideOverlayImmediately = (): void => {
  if (!overlayEl) return
  overlayEl.classList.remove(
    'theme-switch-overlay--active',
    'theme-switch-overlay--solid',
    'theme-switch-overlay--leaving',
  )
  delete overlayEl.dataset.themeTarget
}

const applyThemeTokens = (mode: ThemeMode): void => {
  const root = document.documentElement
  root.dataset.theme = mode
  root.classList.toggle('dark', mode === 'dark')
}

const persist = (mode: ThemeMode): void => {
  try {
    localStorage.setItem(STORAGE_KEY, mode)
  } catch {
    // Storage can be unavailable in restricted browser contexts.
  }
}

const commitTheme = (mode: ThemeMode): void => {
  themeMode.value = mode
  applyThemeTokens(mode)
  persist(mode)
}

const applyTheme = (mode: ThemeMode, animate = false): void => {
  const root = document.documentElement
  clearTimers()
  const generation = ++transitionGeneration

  if (!animate || prefersReducedMotion()) {
    root.classList.remove('theme-switching')
    commitTheme(mode)
    hideOverlayImmediately()
    isThemeTransitioning.value = false
    return
  }

  const overlay = ensureOverlay()
  overlay.dataset.themeTarget = mode
  overlay.classList.remove(
    'theme-switch-overlay--active',
    'theme-switch-overlay--solid',
    'theme-switch-overlay--leaving',
  )
  void overlay.offsetWidth
  isThemeTransitioning.value = true

  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      if (generation !== transitionGeneration || !overlayEl) return
      overlay.classList.add('theme-switch-overlay--active')

      enterTimer = setTimeout(() => {
        if (generation !== transitionGeneration || !overlayEl) return
        overlay.classList.add('theme-switch-overlay--solid')
        void overlay.offsetWidth
        root.classList.remove('theme-switching')
        commitTheme(mode)
        enterTimer = null

        holdTimer = setTimeout(() => {
          if (generation !== transitionGeneration || !overlayEl) return
          overlay.classList.remove('theme-switch-overlay--solid')
          void overlay.offsetWidth
          overlay.classList.add('theme-switch-overlay--leaving')

          fadeTimer = setTimeout(() => {
            if (generation !== transitionGeneration) return
            hideOverlayImmediately()
            isThemeTransitioning.value = false
            holdTimer = null
            fadeTimer = null
          }, OVERLAY_FADE_OUT_MS)
        }, OVERLAY_HOLD_MS)
      }, OVERLAY_FADE_IN_MS)
    })
  })
}

const resolveSystemPreference = (): ThemeMode =>
  window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'

const readStored = (): ThemeMode | null => {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    return stored === 'dark' || stored === 'light' ? stored : null
  } catch {
    return null
  }
}

const initTheme = (): void => {
  const mode = readStored() || resolveSystemPreference()
  themeMode.value = mode
  applyThemeTokens(mode)
}

const setTheme = (mode: ThemeMode): void => {
  if (themeMode.value === mode || isThemeTransitioning.value) return
  applyTheme(mode, true)
}

const toggleTheme = (): void => {
  setTheme(themeMode.value === 'dark' ? 'light' : 'dark')
}

export const useTheme = () => ({
  themeMode,
  isDark,
  isThemeTransitioning,
  initTheme,
  toggleTheme,
  setTheme,
})
