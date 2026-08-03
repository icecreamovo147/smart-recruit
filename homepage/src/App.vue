<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ArrowRight, CopyDocument, Menu, Moon, Sunny } from '@element-plus/icons-vue'
import logoDark from '@shared/assets/logo-small-dark.webp'
import logoLight from '@shared/assets/logo-small.webp'
import miuraHero from '@/assets/miura-deploy-hero.webp'
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
const roleTags = ['TALENT OPS', 'CANDIDATE', 'PLATFORM']

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
const prefersReducedMotion = ref(false)
let copiedTimer: number | undefined
let revealObserver: IntersectionObserver | undefined

const t = computed(() => content[locale.value])
const isDark = computed(() => theme.value === 'dark')
const brandLogo = computed(() => (isDark.value ? logoDark : logoLight))
const brandName = computed(() => (locale.value === 'zh-CN' ? '智联招聘' : 'Smart Recruit'))

const applyDocumentPreferences = () => {
	document.documentElement.lang = locale.value
	document.documentElement.dataset.theme = theme.value
	document.documentElement.style.colorScheme = theme.value
	document.title =
		locale.value === 'zh-CN' ? '智联招聘 · 智能招聘平台' : 'Smart Recruit · Intelligent Recruiting Platform'
	document
		.querySelector('meta[name="theme-color"]')
		?.setAttribute('content', theme.value === 'dark' ? '#111713' : '#f5f4ef')
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

const observeReveals = () => {
	const targets = document.querySelectorAll('.reveal')
	if (prefersReducedMotion.value || !('IntersectionObserver' in window)) {
		for (const element of targets) element.classList.add('is-visible')
		return
	}
	revealObserver = new IntersectionObserver(
		(entries) => {
			for (const entry of entries) {
				if (entry.isIntersecting) {
					entry.target.classList.add('is-visible')
					revealObserver?.unobserve(entry.target)
				}
			}
		},
		{ threshold: 0.12, rootMargin: '0px 0px -6% 0px' },
	)
	for (const element of targets) revealObserver.observe(element)
}

onMounted(async () => {
	prefersReducedMotion.value = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false
	applyDocumentPreferences()
	await nextTick()
	document.documentElement.classList.add('js')
	observeReveals()
	window.addEventListener('keydown', handleEscape)
})

onBeforeUnmount(() => {
	window.removeEventListener('keydown', handleEscape)
	revealObserver?.disconnect()
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
				<a class="brand" href="#top" :aria-label="brandName">
					<img :src="brandLogo" alt="" width="38" height="38" />
					<span>{{ brandName }}</span>
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
						<Sunny v-if="isDark" />
						<Moon v-else />
					</button>
					<button
						class="locale-button"
						type="button"
						data-testid="language-toggle"
						:aria-label="locale === 'zh-CN' ? 'Switch to English' : '切换到中文'"
						@click="toggleLocale"
					>
						{{ locale === 'zh-CN' ? 'EN' : '中' }}
					</button>
					<a class="header-github" :href="repositoryUrl" target="_blank" rel="noreferrer">
						<span>{{ t.common.github }}</span>
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
				<a :href="docsUrl" target="_blank" rel="noreferrer" @click="closeMobileMenu">{{ t.nav.docs }}</a>
				<button type="button" @click="toggleLocale">{{ locale === 'zh-CN' ? 'English' : '中文' }}</button>
				<a :href="repositoryUrl" target="_blank" rel="noreferrer" @click="closeMobileMenu">
					{{ t.common.github }}
				</a>
			</nav>
		</header>

		<main id="main-content">
			<section id="top" class="hero-section">
				<div class="hero-crease" aria-hidden="true"></div>
				<div class="container hero-grid">
					<div class="hero-copy">
						<h1>
							<span class="hero-product-name">{{ brandName }}</span>
							<span class="hero-headline">{{ t.hero.title }}</span>
						</h1>
						<p class="hero-subtitle">{{ t.hero.subtitle }}</p>
						<div class="hero-actions">
							<a class="button button--primary" :href="repositoryUrl" target="_blank" rel="noreferrer">
								{{ t.common.github }} <ArrowRight aria-hidden="true" />
							</a>
							<a class="button button--secondary" href="#architecture">
								<span>{{ t.hero.secondary }}</span> <ArrowRight aria-hidden="true" />
							</a>
						</div>
					</div>

					<div class="hero-visual" role="img" :aria-label="locale === 'zh-CN' ? '招聘平台从折叠状态展开为招聘流程、AI 治理和开放架构' : 'The recruiting platform unfolds into hiring workflow, AI governance, and open architecture'">
						<img :src="miuraHero" alt="" width="1800" height="1200" />
						<div class="hero-visual-label label--journey">01 / {{ locale === 'zh-CN' ? '人才旅程' : 'TALENT JOURNEY' }}</div>
						<div class="hero-visual-label label--governance">02 / {{ locale === 'zh-CN' ? '可信治理' : 'GOVERNED AI' }}</div>
						<div class="hero-visual-label label--architecture">03 / {{ locale === 'zh-CN' ? '开放架构' : 'OPEN STACK' }}</div>
					</div>
				</div>

				<div class="container signal-strip" aria-label="Highlights">
					<div v-for="(signal, index) in t.hero.signals" :key="signal" class="signal-item">
						<span>{{ String(index + 1).padStart(2, '0') }}</span>
						<strong>{{ signal }}</strong>
					</div>
				</div>
			</section>

			<section id="capabilities" class="section capabilities-section">
				<div class="container capabilities-layout">
					<div class="section-heading reveal">
						<h2>{{ t.capabilities.title }}</h2>
						<p>{{ t.capabilities.subtitle }}</p>
					</div>
					<div class="role-assembly reveal">
						<article v-for="(role, index) in t.capabilities.roles" :key="role.title" class="role-item">
							<div class="role-index">
								<span>{{ String(index + 1).padStart(2, '0') }}</span>
								<small>{{ roleTags[index] }}</small>
							</div>
							<div class="role-copy">
								<h3>{{ role.title }}</h3>
								<p>{{ role.body }}</p>
							</div>
							<a href="#ai" :aria-label="role.link"><ArrowRight aria-hidden="true" /></a>
						</article>
					</div>
				</div>
			</section>

			<section id="ai" class="section ai-section">
				<div class="container ai-layout">
					<div class="ai-copy reveal">
						<h2>{{ t.ai.title }}</h2>
						<p>{{ t.ai.body }}</p>
						<ul class="feature-list">
							<li v-for="(bullet, index) in t.ai.bullets" :key="bullet">
								<span>{{ String(index + 1).padStart(2, '0') }}</span>
								{{ bullet }}
							</li>
						</ul>
					</div>

					<div class="evidence-fold reveal" role="img" :aria-label="t.ai.title">
						<div class="fold-panel fold-panel--candidate">
							<span class="fold-label">01 / CONTEXT</span>
							<svg viewBox="0 0 240 200" aria-hidden="true">
								<path d="M30 100h45m28 0h36m28 0h43" />
								<circle cx="88" cy="100" r="16" />
								<circle cx="152" cy="100" r="16" />
								<path d="M88 84V45m64 39V45M88 116v39m64-39v39" />
								<circle cx="88" cy="35" r="6" /><circle cx="152" cy="35" r="6" />
								<circle cx="88" cy="165" r="6" /><circle cx="152" cy="165" r="6" />
							</svg>
							<strong>{{ locale === 'zh-CN' ? '理解候选人与岗位' : 'Understand role and candidate' }}</strong>
						</div>
						<div class="fold-panel fold-panel--evidence">
							<span class="fold-label">02 / EVIDENCE</span>
							<svg viewBox="0 0 240 200" aria-hidden="true">
								<polygon points="120,32 177,65 177,131 120,164 63,131 63,65" />
								<circle cx="120" cy="98" r="23" />
								<path d="M120 32v43m0 46v43M63 65l37 21m40 23 37 22M63 131l37-22m40-23 37-21" />
							</svg>
							<strong>{{ locale === 'zh-CN' ? '让判断有据可循' : 'Make decisions explainable' }}</strong>
						</div>
						<div class="fold-panel fold-panel--policy">
							<span class="fold-label">03 / POLICY</span>
							<svg viewBox="0 0 240 200" aria-hidden="true">
								<path d="M120 27 177 49v53c0 37-23 61-57 75-34-14-57-38-57-75V49Z" />
								<path d="m94 102 18 18 38-43" />
							</svg>
							<strong>{{ locale === 'zh-CN' ? '策略、配额与审计' : 'Policy, quota, and audit' }}</strong>
						</div>
					</div>
				</div>
			</section>

			<section id="architecture" class="section architecture-section">
				<div class="container architecture-layout">
					<div class="architecture-intro reveal">
						<h2>{{ t.architecture.title }}</h2>
						<p>{{ t.architecture.body }}</p>
					</div>
					<div class="stack-fold reveal">
						<div v-for="(technology, index) in t.architecture.technologies" :key="technology.name" class="stack-cell">
							<span>{{ String(index + 1).padStart(2, '0') }}</span>
							<strong>{{ technology.name }}</strong>
							<small>{{ technology.detail }}</small>
						</div>
					</div>

					<div id="open-source" class="quick-start reveal">
						<div class="quick-start-copy">
							<h2>{{ t.architecture.quickStartTitle }}</h2>
							<p>{{ t.architecture.quickStartBody }}</p>
							<a :href="docsUrl" target="_blank" rel="noreferrer">
								{{ t.architecture.docsLink }} <ArrowRight aria-hidden="true" />
							</a>
						</div>
						<div class="code-panel">
							<div class="code-toolbar">
								<span>~/smart-recruit</span>
								<button type="button" :aria-label="t.common.copy" data-testid="copy-command" @click="copyQuickStart">
									<CopyDocument aria-hidden="true" />
									{{ copied ? t.common.copied : t.common.copy }}
								</button>
							</div>
							<pre><code>{{ quickStartCommand }}</code></pre>
						</div>
					</div>
				</div>
			</section>

			<section class="closing-cta">
				<div class="container closing-cta-inner reveal">
					<div>
						<h2>{{ t.cta.title }}</h2>
						<p>{{ t.cta.body }}</p>
					</div>
					<a class="button button--foil" :href="repositoryUrl" target="_blank" rel="noreferrer">
						{{ t.common.github }} <ArrowRight aria-hidden="true" />
					</a>
				</div>
			</section>
		</main>

		<footer class="site-footer">
			<div class="container footer-grid">
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
				<span>© {{ new Date().getFullYear() }} {{ brandName }}</span>
				<span>{{ t.footer.madeWith }}</span>
				<a href="https://recruit.jkghjk123.site">recruit.jkghjk123.site</a>
			</div>
		</footer>
	</div>
</template>
