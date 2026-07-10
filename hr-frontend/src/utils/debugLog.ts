type DebugDomain = 'HTTP' | 'RecruitingIntelligence' | 'MCP' | 'AgentSkill' | 'AgentTrace'

type DebugLevel = 'info' | 'warn' | 'error'

const PREFIX: Record<DebugDomain, string> = {
  HTTP: '[HR][HTTP]',
  RecruitingIntelligence: '[HR][RecruitingIntelligence]',
  MCP: '[HR][MCP]',
  AgentSkill: '[HR][AgentSkill]',
  AgentTrace: '[HR][AgentTrace]',
}

const ENV_MAP: Record<DebugDomain, string> = {
  HTTP: 'VITE_DEBUG_HTTP',
  RecruitingIntelligence: 'VITE_DEBUG_RECRUITING_INTELLIGENCE',
  MCP: 'VITE_DEBUG_MCP',
  AgentSkill: 'VITE_DEBUG_AGENT_SKILL',
  AgentTrace: 'VITE_DEBUG_AGENT_TRACE',
}

function isEnabled(domain: DebugDomain): boolean {
  const key = ENV_MAP[domain]
  const val = typeof import.meta !== 'undefined' && import.meta.env ? import.meta.env[key] : false
  return val === true || val === 'true' || val === '1'
}

function log(domain: DebugDomain, level: DebugLevel, message: string, data?: Record<string, unknown>): void {
  if (!isEnabled(domain)) return
  const prefix = PREFIX[domain]
  const timestamp = new Date().toISOString()
  const payload = data ? ` ${JSON.stringify(data)}` : ''
  const fn = level === 'error' ? console.error : level === 'warn' ? console.warn : console.log
  fn(`[${timestamp}] ${prefix} ${message}${payload}`)
}

export const debugLog = {
  http: {
    info: (msg: string, data?: Record<string, unknown>) => log('HTTP', 'info', msg, data),
    warn: (msg: string, data?: Record<string, unknown>) => log('HTTP', 'warn', msg, data),
    error: (msg: string, data?: Record<string, unknown>) => log('HTTP', 'error', msg, data),
  },
  ri: {
    info: (msg: string, data?: Record<string, unknown>) => log('RecruitingIntelligence', 'info', msg, data),
    warn: (msg: string, data?: Record<string, unknown>) => log('RecruitingIntelligence', 'warn', msg, data),
    error: (msg: string, data?: Record<string, unknown>) => log('RecruitingIntelligence', 'error', msg, data),
  },
  mcp: {
    info: (msg: string, data?: Record<string, unknown>) => log('MCP', 'info', msg, data),
    warn: (msg: string, data?: Record<string, unknown>) => log('MCP', 'warn', msg, data),
    error: (msg: string, data?: Record<string, unknown>) => log('MCP', 'error', msg, data),
  },
  skill: {
    info: (msg: string, data?: Record<string, unknown>) => log('AgentSkill', 'info', msg, data),
    warn: (msg: string, data?: Record<string, unknown>) => log('AgentSkill', 'warn', msg, data),
    error: (msg: string, data?: Record<string, unknown>) => log('AgentSkill', 'error', msg, data),
  },
  trace: {
    info: (msg: string, data?: Record<string, unknown>) => log('AgentTrace', 'info', msg, data),
    warn: (msg: string, data?: Record<string, unknown>) => log('AgentTrace', 'warn', msg, data),
    error: (msg: string, data?: Record<string, unknown>) => log('AgentTrace', 'error', msg, data),
  },
}
