<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  ArrowRight,
  Briefcase,
  Check,
  Connection,
  CopyDocument,
  Cpu,
  DataAnalysis,
  Document,
  Lock,
  Menu,
  Moon,
  Platform,
  Search,
  Sunny,
  User,
} from '@element-plus/icons-vue'
import logoDark from '@shared/assets/logo-small-dark.webp'
import logoLight from '@shared/assets/logo-small.webp'
import heroLight from '@/assets/hero-growth-light.avif'
import heroDark from '@/assets/hero-growth-dark.avif'
import aiLight from '@/assets/ai-agent-light.avif'
import aiDark from '@/assets/ai-agent-dark.avif'
import ctaWaves from '@/assets/cta-waves.avif'
import { content, type Locale } from '@/content'

type Theme = 'light' | 'dark'

const repositoryUrl = 'https://github.com/icecreamovo147/smart-recruit'
const docsUrl = `${repositoryUrl}#readme`
const issuesUrl = `${repositoryUrl}/issues`
const quickStartCommand = `git clone https://github.com/icecreamovo147/smart-recruit.git
cd smart-recruit
cp docker/.env.example docker/.env
./start-dev.sh`

const localeStorageKey = 'smart-recruit-homepage-locale'
const themeStorageKey = 'smart-recruit-homepage-theme'

const isLocale = (value: string | null): value is Locale => value === 'zh-CN' || value === 'en-US'
const isTheme = (value: string | null): value is Theme => value === 'light' || value === 'dark'

const getInitialLocale = (): Locale => {
  if (typeof window === 'undefined') return 'zh-CN'
  const stored = window.localStorage.getItem(localeStorageKey)
  return isLocale(stored) ? stored : 'zh-CN'
}

const getInitialTheme = (): Theme => {
  if (typeof window === 'undefined') return 'light'
  const stored = window.localStorage.getItem(themeStorageKey)
  if (isTheme(stored)) return stored
  return window.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

const locale = ref<Locale>(getInitialLocale())
const theme = ref<Theme>(getInitialTheme())
const mobileMenuOpen = ref(false)
const copied = ref(false)
let copiedTimer: number | undefined

const t = computed(() => content[locale.value])
const isDark = computed(() => theme.value === 'dark')
const brandLogo = computed(() => (isDark.value ? logoDark : logoLight))
const heroImage = computed(() => (isDark.value ? heroDark : heroLight))
const aiImage = computed(() => (isDark.value ? aiDark : aiLight))

const roleIcons = [Briefcase, User, Lock]
const technologyIcons = [Platform, Cpu, Connection, DataAnalysis, Lock, Search]

const applyDocumentPreferences = () => {
  document.documentElement.lang = locale.value
  document.documentElement.dataset.theme = theme.value
  document.documentElement.style.colorScheme = theme.value
  document
    .querySelector('meta[name="theme-color"]')
    ?.setAttribute('content', theme.value === 'dark' ? '#071224' : '#f7f9fd')
}

watch([locale, theme], () => {
  window.localStorage.setItem(localeStorageKey, locale.value)
  window.localStorage.setItem(themeStorageKey, theme.value)
  applyDocumentPreferences()
})

const toggleLocale = () => {
  locale.value = locale.value === 'zh-CN' ? 'en-US' : 'zh-CN'
  mobileMenuOpen.value = false
}

const toggleTheme = () => {
  theme.value = theme.value === 'light' ? 'dark' : 'light'
}

const closeMobileMenu = () => {
  mobileMenuOpen.value = false
}

const copyQuickStart = async () => {
  try {
    await navigator.clipboard.writeText(quickStartCommand)
    copied.value = true
    if (copiedTimer) window.clearTimeout(copiedTimer)
    copiedTimer = window.setTimeout(() => {
      copied.value = false
    }, 1800)
  } catch {
    copied.value = false
  }
}

const handleEscape = (event: KeyboardEvent) => {
  if (event.key === 'Escape') mobileMenuOpen.value = false
}

onMounted(async () => {
  applyDocumentPreferences()
  await nextTick()
  window.addEventListener('keydown', handleEscape)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleEscape)
  if (copiedTimer) window.clearTimeout(copiedTimer)
})
</script>

<template>
  <div class="site-shell">
    <a class="skip-link" href="#main-content">
      {{ locale === 'zh-CN' ? '跳到主要内容' : 'Skip to main content' }}
    </a>

    <header class="site-header">
      <div class="container header-inner">
        <a class="brand" href="#top" aria-label="Smart Recruit">
          <img :src="brandLogo" alt="" width="42" height="42" />
          <span>Smart Recruit</span>
        </a>

        <nav class="desktop-nav" :aria-label="locale === 'zh-CN' ? '主要导航' : 'Primary navigation'">
          <a href="#capabilities">{{ t.nav.capabilities }}</a>
          <a href="#ai">{{ t.nav.ai }}</a>
          <a href="#architecture">{{ t.nav.architecture }}</a>
          <a href="#open-source">{{ t.nav.openSource }}</a>
          <a :href="docsUrl" target="_blank" rel="noreferrer">{{ t.nav.docs }}</a>
        </nav>

        <div class="header-actions">
          <button
            class="icon-button"
            type="button"
            data-testid="theme-toggle"
            :aria-label="isDark ? 'Switch to light theme' : 'Switch to dark theme'"
            @click="toggleTheme"
          >
            <component :is="isDark ? Sunny : Moon" />
          </button>
          <button
            class="locale-button"
            type="button"
            data-testid="language-toggle"
            :aria-label="locale === 'zh-CN' ? 'Switch to English' : '切换到中文'"
            @click="toggleLocale"
          >
            <strong>{{ locale === 'zh-CN' ? '中' : 'EN' }}</strong>
            <span>/</span>
            <span>{{ locale === 'zh-CN' ? 'EN' : '中' }}</span>
          </button>
          <a class="button button--primary header-github" :href="repositoryUrl" target="_blank" rel="noreferrer">
            {{ t.common.github }}
          </a>
          <button
            class="icon-button mobile-menu-button"
            type="button"
            :aria-expanded="mobileMenuOpen"
            aria-controls="mobile-navigation"
            :aria-label="locale === 'zh-CN' ? '打开导航' : 'Open navigation'"
            @click="mobileMenuOpen = !mobileMenuOpen"
          >
            <Menu />
          </button>
        </div>
      </div>

      <nav
        v-if="mobileMenuOpen"
        id="mobile-navigation"
        class="mobile-nav"
        :aria-label="locale === 'zh-CN' ? '移动导航' : 'Mobile navigation'"
      >
        <a href="#capabilities" @click="closeMobileMenu">{{ t.nav.capabilities }}</a>
        <a href="#ai" @click="closeMobileMenu">{{ t.nav.ai }}</a>
        <a href="#architecture" @click="closeMobileMenu">{{ t.nav.architecture }}</a>
        <a href="#open-source" @click="closeMobileMenu">{{ t.nav.openSource }}</a>
        <button class="mobile-locale-button" type="button" @click="toggleLocale">
          {{ locale === 'zh-CN' ? 'English' : '中文' }}
        </button>
        <a :href="repositoryUrl" target="_blank" rel="noreferrer" @click="closeMobileMenu">
          {{ t.common.github }}
        </a>
      </nav>
    </header>

    <main id="main-content">
      <section id="top" class="hero-section">
        <div class="container hero-grid">
          <div class="hero-copy">
            <p class="eyebrow">{{ t.hero.eyebrow }}</p>
            <h1>
              <span class="hero-product-name">Smart Recruit</span>
              <span class="hero-headline">{{ t.hero.title }}</span>
            </h1>
            <p class="hero-subtitle">{{ t.hero.subtitle }}</p>
            <div class="hero-actions">
              <a class="button button--primary" :href="repositoryUrl" target="_blank" rel="noreferrer">
                {{ t.common.github }}
                <ArrowRight />
              </a>
              <a class="button button--secondary" href="#architecture">
                {{ t.hero.secondary }}
              </a>
            </div>
            <ul class="signal-list" aria-label="Highlights">
              <li v-for="signal in t.hero.signals" :key="signal">
                <Check />
                <span>{{ signal }}</span>
              </li>
            </ul>
          </div>
          <div class="hero-visual">
            <img
              :src="heroImage"
              width="1586"
              height="992"
              :alt="locale === 'zh-CN' ? '透明玻璃柱状图与向上箭头，象征招聘成长' : 'Glass bars and a rising arrow representing recruiting growth'"
            />
          </div>
        </div>
      </section>

      <section id="capabilities" class="section capabilities-section">
        <div class="container">
          <div class="section-heading">
            <p class="eyebrow">{{ t.capabilities.eyebrow }}</p>
            <h2>{{ t.capabilities.title }}</h2>
            <p>{{ t.capabilities.subtitle }}</p>
          </div>
          <div class="role-grid">
            <article v-for="(role, index) in t.capabilities.roles" :key="role.title" class="role-item">
              <span class="role-icon" aria-hidden="true">
                <component :is="roleIcons[index]" />
              </span>
              <h3>{{ role.title }}</h3>
              <p>{{ role.body }}</p>
              <a href="#ai">
                {{ role.link }}
                <ArrowRight />
              </a>
            </article>
          </div>
        </div>
      </section>

      <section id="ai" class="section ai-section">
        <div class="container ai-grid">
          <div class="ai-copy">
            <p class="eyebrow">{{ t.ai.eyebrow }}</p>
            <h2>{{ t.ai.title }}</h2>
            <p>{{ t.ai.body }}</p>
            <ul class="feature-list">
              <li v-for="bullet in t.ai.bullets" :key="bullet">
                <Check />
                <span>{{ bullet }}</span>
              </li>
            </ul>
          </div>
          <div class="ai-visual">
            <img
              :src="aiImage"
              width="1448"
              height="1086"
              :alt="locale === 'zh-CN' ? 'AI 助手与候选人评估面板' : 'AI assistant with candidate evaluation panels'"
            />
          </div>
        </div>
      </section>

      <section id="architecture" class="section architecture-section">
        <div class="container">
          <div class="architecture-intro">
            <p class="eyebrow">{{ t.architecture.eyebrow }}</p>
            <h2>{{ t.architecture.title }}</h2>
            <p>{{ t.architecture.body }}</p>
          </div>
          <div class="technology-grid">
            <article
              v-for="(technology, index) in t.architecture.technologies"
              :key="technology.name"
              class="technology-item"
            >
              <component :is="technologyIcons[index]" aria-hidden="true" />
              <h3>{{ technology.name }}</h3>
              <p>{{ technology.detail }}</p>
            </article>
          </div>

          <div id="open-source" class="quick-start">
            <div class="quick-start-copy">
              <p class="eyebrow">{{ t.architecture.quickStartEyebrow }}</p>
              <h2>{{ t.architecture.quickStartTitle }}</h2>
              <p>{{ t.architecture.quickStartBody }}</p>
              <a :href="docsUrl" target="_blank" rel="noreferrer">
                {{ t.architecture.docsLink }}
                <ArrowRight />
              </a>
            </div>
            <div class="code-panel">
              <div class="code-toolbar">
                <span>Terminal</span>
                <button type="button" :aria-label="t.common.copy" data-testid="copy-command" @click="copyQuickStart">
                  <Check v-if="copied" />
                  <CopyDocument v-else />
                  {{ copied ? t.common.copied : t.common.copy }}
                </button>
              </div>
              <pre><code>{{ quickStartCommand }}</code></pre>
            </div>
          </div>
        </div>
      </section>

      <section class="closing-cta" :style="{ backgroundImage: `url(${ctaWaves})` }">
        <div class="container closing-cta-inner">
          <p class="eyebrow">{{ t.cta.eyebrow }}</p>
          <h2>{{ t.cta.title }}</h2>
          <p>{{ t.cta.body }}</p>
          <a class="button button--light" :href="repositoryUrl" target="_blank" rel="noreferrer">
            {{ t.common.github }}
            <ArrowRight />
          </a>
        </div>
      </section>
    </main>

    <footer class="site-footer">
      <div class="container footer-grid">
        <div class="footer-brand">
          <a class="brand" href="#top">
            <img :src="brandLogo" alt="" width="38" height="38" />
            <span>Smart Recruit</span>
          </a>
          <p>{{ t.footer.summary }}</p>
        </div>
        <div class="footer-column">
          <strong>{{ t.footer.product }}</strong>
          <a href="#capabilities">{{ t.footer.capabilities }}</a>
          <a href="#ai">{{ t.footer.ai }}</a>
          <a href="#architecture">{{ t.footer.architecture }}</a>
        </div>
        <div class="footer-column">
          <strong>{{ t.footer.resources }}</strong>
          <a :href="repositoryUrl" target="_blank" rel="noreferrer">{{ t.footer.repository }}</a>
          <a :href="docsUrl" target="_blank" rel="noreferrer">{{ t.footer.documentation }}</a>
          <a :href="`${repositoryUrl}#许可证`" target="_blank" rel="noreferrer">{{ t.footer.license }}</a>
        </div>
        <div class="footer-column">
          <strong>{{ t.footer.community }}</strong>
          <a :href="issuesUrl" target="_blank" rel="noreferrer">{{ t.footer.issues }}</a>
          <a :href="`${repositoryUrl}/pulls`" target="_blank" rel="noreferrer">{{ t.footer.contribute }}</a>
        </div>
      </div>
      <div class="container footer-bottom">
        <span>© {{ new Date().getFullYear() }} Smart Recruit</span>
        <span>{{ t.footer.madeWith }}</span>
        <a href="https://recruit.jkghjk123.site">recruit.jkghjk123.site</a>
      </div>
    </footer>
  </div>
</template>
