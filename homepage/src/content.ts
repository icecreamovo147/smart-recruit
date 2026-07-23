export type Locale = 'zh-CN' | 'en-US'

export interface LocalizedContent {
  nav: {
    capabilities: string
    ai: string
    architecture: string
    openSource: string
    docs: string
  }
  common: {
    github: string
    learnMore: string
    copied: string
    copy: string
  }
  hero: {
    eyebrow: string
    title: string
    subtitle: string
    secondary: string
    signals: string[]
  }
  capabilities: {
    eyebrow: string
    title: string
    subtitle: string
    roles: Array<{ title: string; body: string; link: string }>
  }
  ai: {
    eyebrow: string
    title: string
    body: string
    bullets: string[]
  }
  architecture: {
    eyebrow: string
    title: string
    body: string
    technologies: Array<{ name: string; detail: string }>
    quickStartEyebrow: string
    quickStartTitle: string
    quickStartBody: string
    docsLink: string
  }
  cta: {
    eyebrow: string
    title: string
    body: string
  }
  footer: {
    summary: string
    product: string
    resources: string
    community: string
    capabilities: string
    ai: string
    architecture: string
    repository: string
    documentation: string
    license: string
    issues: string
    contribute: string
    madeWith: string
  }
}

export const content: Record<Locale, LocalizedContent> = {
  'zh-CN': {
    nav: {
      capabilities: '产品',
      ai: 'AI 能力',
      architecture: '架构',
      openSource: '开源',
      docs: '文档',
    },
    common: {
      github: '在 GitHub 查看',
      learnMore: '了解更多',
      copied: '已复制',
      copy: '复制命令',
    },
    hero: {
      eyebrow: 'OPEN SOURCE · AI-NATIVE RECRUITING',
      title: '连接人才与机会，让招聘更智能',
      subtitle:
        '面向招聘团队与开发者的开源智能招聘平台。以 AI Agent 与可信治理为核心，覆盖招聘全流程，帮助组织更高效地发现、评估与成就优秀人才。',
      secondary: '阅读架构',
      signals: ['完全开源', '可信治理', '模块化设计', '社区驱动'],
    },
    capabilities: {
      eyebrow: 'ONE PLATFORM · THREE PERSPECTIVES',
      title: '为不同角色创造价值',
      subtitle: '从招聘协作到平台治理，智能能力贯穿每一步。',
      roles: [
        {
          title: '招聘团队',
          body: '统一管理岗位、候选人、面试与 Offer，让协作节奏清晰，让招聘决策更有依据。',
          link: '探索招聘工作台',
        },
        {
          title: '候选人',
          body: '获得清晰透明的求职旅程，管理简历、投递、面试与 Offer，及时掌握每个进展。',
          link: '了解候选人体验',
        },
        {
          title: '平台运营者',
          body: '通过租户治理、权限审计、套餐与配额管理，构建安全、可控、可观测的平台。',
          link: '查看平台治理',
        },
      ],
    },
    ai: {
      eyebrow: 'AI AGENT',
      title: '与招聘场景深度融合的 AI Agent',
      body:
        'Smart Recruit 将检索、评估与上下文记忆嵌入真实招聘流程，让 AI 从通用问答走向可追踪、可解释、可治理的招聘协作。',
      bullets: [
        '多 Agent 协同，覆盖候选人研究、职位理解与面试辅助',
        '结合组织知识和招聘数据，提供有依据的智能建议',
        '保留运行轨迹与工具调用，让关键判断可以解释',
        '通过配额、审计与策略控制，让智能能力安全落地',
      ],
    },
    architecture: {
      eyebrow: 'ENGINEERED FOR FLEXIBILITY',
      title: '现代化、可扩展、可观测',
      body:
        '以 Vue 3 前端、Go 微服务与 gRPC 契约构建清晰边界，结合消息驱动、权限治理与完整可观测能力，为持续演进留足空间。',
      technologies: [
        { name: 'Vue 3', detail: '三套业务前端与共享组件体系' },
        { name: 'Go', detail: '独立构建的领域微服务' },
        { name: 'gRPC', detail: '强类型跨服务契约' },
        { name: 'Eino / ADK', detail: '可治理的 Agent 运行时' },
        { name: '安全治理', detail: 'RBAC、审计与内部鉴权' },
        { name: '可观测性', detail: '日志、指标、追踪与健康检查' },
      ],
      quickStartEyebrow: 'QUICK START',
      quickStartTitle: '几分钟启动本地环境',
      quickStartBody: '完整代码、部署编排和开发文档都在同一个仓库中。',
      docsLink: '查看完整文档',
    },
    cta: {
      eyebrow: 'OPEN SOURCE · COMMUNITY DRIVEN',
      title: '与我们一起，构建更智能的招聘未来',
      body: '探索代码、参与讨论，或为下一次招聘体验贡献你的想法。',
    },
    footer: {
      summary: '开源智能招聘平台，连接人才与机会，让组织和开发者共同塑造招聘未来。',
      product: '产品',
      resources: '资源',
      community: '社区',
      capabilities: '产品能力',
      ai: 'AI 能力',
      architecture: '技术架构',
      repository: 'GitHub 仓库',
      documentation: '项目文档',
      license: 'MIT License',
      issues: '问题反馈',
      contribute: '参与贡献',
      madeWith: '为招聘团队与开源社区共同打造',
    },
  },
  'en-US': {
    nav: {
      capabilities: 'Product',
      ai: 'AI',
      architecture: 'Architecture',
      openSource: 'Open Source',
      docs: 'Docs',
    },
    common: {
      github: 'View on GitHub',
      learnMore: 'Learn more',
      copied: 'Copied',
      copy: 'Copy command',
    },
    hero: {
      eyebrow: 'OPEN SOURCE · AI-NATIVE RECRUITING',
      title: 'Connect talent with opportunity, intelligently',
      subtitle:
        'An open-source recruiting platform for talent teams and developers. Smart Recruit brings AI Agents and trusted governance into the full hiring journey—from discovery to decision.',
      secondary: 'Explore architecture',
      signals: ['Open source', 'Trusted governance', 'Modular by design', 'Community driven'],
    },
    capabilities: {
      eyebrow: 'ONE PLATFORM · THREE PERSPECTIVES',
      title: 'Built around every participant',
      subtitle: 'One connected experience for hiring, candidate progress, and platform governance.',
      roles: [
        {
          title: 'Talent teams',
          body: 'Coordinate jobs, candidates, interviews, and offers with a clear operating rhythm and better-informed decisions.',
          link: 'Explore the HR workspace',
        },
        {
          title: 'Candidates',
          body: 'Manage profiles, applications, interviews, and offers through a transparent and supportive candidate journey.',
          link: 'See the candidate experience',
        },
        {
          title: 'Platform operators',
          body: 'Govern tenants, access, audit trails, plans, and quotas through a secure and observable control plane.',
          link: 'Discover platform governance',
        },
      ],
    },
    ai: {
      eyebrow: 'AI AGENT',
      title: 'AI Agents grounded in real recruiting work',
      body:
        'Smart Recruit connects retrieval, evaluation, and contextual memory to hiring workflows—moving beyond generic chat toward traceable, explainable, and governable assistance.',
      bullets: [
        'Multi-agent collaboration for candidate research, role understanding, and interview support',
        'Evidence-aware recommendations grounded in organizational knowledge and recruiting data',
        'Visible execution traces and tool activity for explainable decisions',
        'Quotas, audits, and policy controls that make AI safe to operate',
      ],
    },
    architecture: {
      eyebrow: 'ENGINEERED FOR FLEXIBILITY',
      title: 'Modern, extensible, and observable',
      body:
        'Vue 3 experiences, independently deployable Go services, and typed gRPC contracts create clear boundaries—supported by event-driven workflows, access governance, and full-stack observability.',
      technologies: [
        { name: 'Vue 3', detail: 'Three product surfaces and a shared UI layer' },
        { name: 'Go', detail: 'Independently buildable domain services' },
        { name: 'gRPC', detail: 'Strongly typed service contracts' },
        { name: 'Eino / ADK', detail: 'A governable Agent runtime' },
        { name: 'Security', detail: 'RBAC, audit, and internal auth' },
        { name: 'Observability', detail: 'Logs, metrics, traces, and health' },
      ],
      quickStartEyebrow: 'QUICK START',
      quickStartTitle: 'Run the stack in minutes',
      quickStartBody: 'Source, deployment orchestration, and developer documentation live in one repository.',
      docsLink: 'Read the full documentation',
    },
    cta: {
      eyebrow: 'OPEN SOURCE · COMMUNITY DRIVEN',
      title: 'Help shape the future of intelligent recruiting',
      body: 'Explore the code, join the discussion, or contribute your perspective to the next hiring experience.',
    },
    footer: {
      summary: 'Open-source intelligent recruiting, connecting talent and opportunity through a community-built platform.',
      product: 'Product',
      resources: 'Resources',
      community: 'Community',
      capabilities: 'Capabilities',
      ai: 'AI Agents',
      architecture: 'Architecture',
      repository: 'GitHub repository',
      documentation: 'Documentation',
      license: 'MIT License',
      issues: 'Report an issue',
      contribute: 'Contribute',
      madeWith: 'Built for talent teams and the open-source community',
    },
  },
}
