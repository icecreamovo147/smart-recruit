import { computed, ref } from 'vue'

type ThemeMode = 'light' | 'dark'

const STORAGE_KEY = 'recruitment_theme_mode'
/** Fade-in duration (must match CSS `.theme-switch-overlay` transition). */
const OVERLAY_FADE_IN_MS = 280
/** Fully visible hold after theme is committed under a solid mask. */
const OVERLAY_HOLD_MS = 1500
/** Fade-out duration (must match CSS). */
const OVERLAY_FADE_OUT_MS = 280

const themeMode = ref<ThemeMode>('light')
const isThemeTransitioning = ref(false)

let enterTimer: ReturnType<typeof setTimeout> | null = null
let holdTimer: ReturnType<typeof setTimeout> | null = null
let fadeTimer: ReturnType<typeof setTimeout> | null = null
let overlayEl: HTMLDivElement | null = null
/** Bumped on every applyTheme to cancel stale rAF / timeout chains. */
let transitionGen = 0

const isDark = computed(() => themeMode.value === 'dark')

const prefersReducedMotion = (): boolean => {
  try {
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches
  } catch {
    return false
  }
}

const clearTimers = (): void => {
  if (enterTimer) {
    clearTimeout(enterTimer)
    enterTimer = null
  }
  if (holdTimer) {
    clearTimeout(holdTimer)
    holdTimer = null
  }
  if (fadeTimer) {
    clearTimeout(fadeTimer)
    fadeTimer = null
  }
}

const ensureOverlay = (): HTMLDivElement => {
  if (overlayEl && document.body.contains(overlayEl)) {
    return overlayEl
  }
  const el = document.createElement('div')
  el.id = 'theme-switch-overlay'
  el.className = 'theme-switch-overlay'
  el.setAttribute('aria-hidden', 'true')
  el.innerHTML = `
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
  document.body.appendChild(el)
  overlayEl = el
  return el
}

const hideOverlayImmediate = (): void => {
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
  if (mode === 'dark') {
    root.classList.add('dark')
  } else {
    root.classList.remove('dark')
  }
}

/**
 * Commit theme under a fully opaque mask.
 * Vue reactive mode, DOM tokens, and localStorage all flip together here —
 * never during a translucent fade frame.
 */
const commitTheme = (mode: ThemeMode): void => {
  themeMode.value = mode
  applyThemeTokens(mode)
  persist(mode)
}

const applyTheme = (mode: ThemeMode, animate = false): void => {
  const root = document.documentElement
  clearTimers()
  const gen = ++transitionGen

  const useOverlay = animate && !prefersReducedMotion()
  if (!useOverlay) {
    root.classList.remove('theme-switching')
    commitTheme(mode)
    hideOverlayImmediate()
    isThemeTransitioning.value = false
    return
  }

  const overlay = ensureOverlay()
  overlay.dataset.themeTarget = mode
  // Reset to translucent baseline so enter can interpolate 0 → 1.
  overlay.classList.remove(
    'theme-switch-overlay--active',
    'theme-switch-overlay--solid',
    'theme-switch-overlay--leaving',
  )
  void overlay.offsetWidth

  isThemeTransitioning.value = true

  // Double rAF: paint baseline, then start enter fade (page still on OLD theme).
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      if (gen !== transitionGen || !overlayEl) return
      overlay.classList.add('theme-switch-overlay--active')

      enterTimer = setTimeout(() => {
        if (gen !== transitionGen || !overlayEl) return

        // Lock fully opaque BEFORE any theme mutation so repaint cannot show through.
        overlay.classList.add('theme-switch-overlay--solid')
        void overlay.offsetWidth

        // Instant token swap under solid mask (no theme-switching soft fade —
        // that would only be visible if the mask leaked).
        root.classList.remove('theme-switching')
        commitTheme(mode)
        enterTimer = null

        // Keep solid cover, then fade out to reveal the already-switched page.
        holdTimer = setTimeout(() => {
          if (gen !== transitionGen || !overlayEl) return

          // Unlock opacity transition for leave.
          overlay.classList.remove('theme-switch-overlay--solid')
          void overlay.offsetWidth
          overlay.classList.add('theme-switch-overlay--leaving')

          fadeTimer = setTimeout(() => {
            if (gen !== transitionGen) return
            hideOverlayImmediate()
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
  applyThemeTokens(mode)
}

const setTheme = (mode: ThemeMode): void => {
  if (themeMode.value === mode) return
  if (isThemeTransitioning.value) return
  // Do NOT flip themeMode here — that would re-render logo/icons under a
  // translucent mask. commitTheme runs only after the mask is solid.
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
