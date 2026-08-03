# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

- Talent teams (recruiters, recruiting admins, interviewers, HR leaders) who run hiring workflows: jobs, candidates, interviews, offers.
- Developers and open-source contributors who evaluate the project's architecture (Vue 3, Go microservices, gRPC), AI agent runtime, and self-hosting story, and who may contribute code.
- The homepage addresses both audiences at once (confirmed by the owner); no audience may be sacrificed to the other.

## Product Purpose

Smart Recruit is an open-source intelligent recruiting platform that connects talent with opportunity. It covers the full hiring journey, from job posting and candidate tracking through interviews, offers, and platform governance, and brings AI Agents into real recruiting workflows in a traceable, explainable, governable way.

## Positioning

An open-source, AI-native recruiting platform with trusted governance: AI assistance (candidate research, role understanding, interview support) is grounded in organizational knowledge and recruiting data, with visible execution traces, quotas, audits, and policy controls. Unlike closed SaaS recruiting suites, the full stack is open: Vue 3 experiences, independently deployable Go services, typed gRPC contracts, and Docker Compose deployment live in one repository.

## Operating Context

- HR workspace (hr-frontend): dashboard, jobs, candidate ledger with status-machine workflow, interviews, offers, AI data assistant, analytics, notifications, admin/RBAC capabilities.
- Candidate portal (user-frontend): job search and details, profile and resume upload, application status tracking, interviews, offers, AI job assistant.
- Platform console (platform-frontend): tenant management, plan/rate-card/quota management, audit logs, platform user administration.
- Interviewer workspace: interview scheduling, candidate materials, and feedback submission bound to enterprise member sessions and permissions.
- Self-hosting via Docker Compose; development via `./start-dev.sh`.
- Identity via JWT + Refresh Token, RBAC roles (Candidate / Recruiter / Recruiting Admin / System Admin / Interviewer), audit logging.

## Capabilities and Constraints

- The homepage is a static Vue 3 + Vite single-page site (`homepage/`), plain CSS in `src/styles.css`, Element Plus icons, i18n (zh-CN / en-US), light/dark themes persisted in localStorage.
- Feature anchors the homepage must keep working: theme toggle, language toggle, mobile navigation, quick-start command copy, in-page anchors, skip link, `data-testid` attributes.
- Visual assets on hand: a generated Miura deployment illustration (`src/assets/miura-deploy-hero.webp`) and a restrained gold-foil texture (`src/assets/gold-foil.webp`). No real product screenshots exist in the repo, and the owner confirmed illustrations are to be used rather than captured product UI.
- Domain: `https://recruit.jkghjk123.site`; repository: `https://github.com/icecreamovo147/smart-recruit`; MIT License.
- No additional hard constraints beyond the defaults listed in the design brief (brand name, logo, repo links, MIT license mention, bilingual copy, dual themes, anchors, test ids, domain).

## Brand Commitments

- Name: Smart Recruit; Chinese display name: 智联招聘 (category: 智能招聘平台; hero concept: 智能连接人才与机会). Chinese-mode interfaces must use 智联招聘 instead of Smart Recruit.
- Logo assets in `packages/shared/src/assets/`: `logo-small.webp`, `logo-small-dark.webp`, `logo-full.webp`, `logo-full-dark.webp`.
- Voice is professional, open, trustworthy; copy must remain factual (open source, Go/Vue/gRPC stack, AI agent capabilities) and must not invent claims, customers, or benchmarks.
- Existing copy register: calm, engineering-oriented, bilingual (zh-CN primary, en-US mirror).

## Evidence on Hand

- `homepage/design-qa/source-design.png`: the reference design the current implementation was accepted against (light theme, blue/white, split hero).
- `homepage/design-qa/`: comparison and implementation screenshots (desktop light zh, desktop dark en, mobile).
- `homepage/src/content.ts`: the full bilingual marketing copy inventory.
- README.md: feature and architecture facts used for copy.
- Absences: no real customer testimonials, no usage statistics, no product screenshots; these must not be fabricated.

## Product Principles

1. Serve both audiences in one surface: HR teams must see a trustworthy, efficient hiring product; developers must see a credible, open, well-engineered project. Neither audience gets a decorative afterthought.
2. Prove, don't claim: show the product's actual mechanism (AI-grounded assistance, governed recruiting workflow, open architecture) through the interface, not through hype adjectives.
3. Trust is the product: governance, auditability, and explainability are first-class values, reflected in tone, structure, and detail.
4. Openness is a feature: the code, the deployment story, and the community path are as prominent as the product capabilities.
5. Preserve the existing functional contract (bilingual, dual theme, anchors, copy-to-clipboard, skip link) while replacing the visual world.

## Accessibility & Inclusion

- The incumbent page honors `prefers-reduced-motion`, provides a skip link, visible focus rings, aria labels, and semantic landmarks; the redesign must not regress these.
- Bilingual (zh-CN / en-US) content must remain complete and parallel in both locales.
