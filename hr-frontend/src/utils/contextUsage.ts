import type { ContextUsageInfo, Session } from '@/types/ai'

export type ContextBudgetSeverity = 'normal' | 'caution' | 'warning' | 'danger' | 'unknown'

const finiteNonNegative = (value: unknown): number | null => {
  if (value == null || value === '') return null
  const numeric = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(numeric) && numeric >= 0 ? numeric : null
}

const positive = (value: unknown): number | null => {
  const numeric = finiteNonNegative(value)
  return numeric != null && numeric > 0 ? numeric : null
}

export const currentEffectiveContextTokens = (usage: ContextUsageInfo | null | undefined): number => {
  if (!usage) return 0
  return positive(usage.prompt_tokens_actual)
    ?? finiteNonNegative(usage.prompt_tokens_estimated)
    ?? 0
}

export const inputBudgetTokens = (usage: ContextUsageInfo | null | undefined): number | null =>
  positive(usage?.input_budget_tokens)

export const contextWindowTokens = (usage: ContextUsageInfo | null | undefined): number | null =>
  positive(usage?.context_window_tokens)

export const contextWindowRatio = (usage: ContextUsageInfo | null | undefined): number | null => {
  const window = contextWindowTokens(usage)
  if (!usage || window == null) return null
  return currentEffectiveContextTokens(usage) / window
}

export const contextWindowProgressRatio = (usage: ContextUsageInfo | null | undefined): number => {
  const ratio = contextWindowRatio(usage) ?? 0
  return Math.min(Math.max(ratio, 0), 1)
}

export const resolveLiveContextUsage = (
  current: ContextUsageInfo | null | undefined,
  incoming: ContextUsageInfo,
): ContextUsageInfo => {
  const isConfigurationOnly = incoming.stage === 'model_selected'
    && current != null
    && currentEffectiveContextTokens(incoming) === 0
    && currentEffectiveContextTokens(current) > 0
  return isConfigurationOnly ? current : incoming
}

export const contextBudgetRatio = (usage: ContextUsageInfo | null | undefined): number | null => {
  if (!usage || inputBudgetTokens(usage) == null) return null
  const backendRatio = finiteNonNegative(usage.budget_usage_ratio)
  if (backendRatio != null) return backendRatio
  return currentEffectiveContextTokens(usage) / (inputBudgetTokens(usage) as number)
}

export const contextBudgetProgressRatio = (usage: ContextUsageInfo | null | undefined): number => {
  const ratio = contextBudgetRatio(usage) ?? 0
  return Math.min(Math.max(ratio, 0), 1)
}

export const contextBudgetSeverity = (usage: ContextUsageInfo | null | undefined): ContextBudgetSeverity => {
  if (!usage) return 'unknown'
  const status = (usage.budget_status || '').trim().toLowerCase()
  if (status === 'invalid_config' || status === 'over_budget') return 'danger'
  if (status === 'unknown_config' || inputBudgetTokens(usage) == null) return 'unknown'
  if (status === 'over_target') return 'warning'
  if (status === 'approaching_target') return 'caution'
  if (status === 'within_budget') return 'normal'
  const ratio = contextBudgetRatio(usage)
  if (ratio == null) return 'unknown'
  if (ratio >= 0.75) return 'warning'
  if (ratio >= 0.6) return 'caution'
  return 'normal'
}

export const formatCompactTokens = (value: number | null | undefined): string => {
  const tokens = finiteNonNegative(value)
  if (tokens == null) return '—'
  if (tokens >= 1_000_000) return `${Number((tokens / 1_000_000).toFixed(1))}M`
  if (tokens >= 1_000) return `${Number((tokens / 1_000).toFixed(1))}K`
  return String(Math.round(tokens))
}

export const contextUsageSourceLabel = (usage: ContextUsageInfo): string => {
  if (positive(usage.prompt_tokens_actual) != null || usage.source === 'provider_actual') return 'Provider 实际值'
  return '保守估算值'
}

export const contextUsageStageLabel = (stage: string | undefined): string => {
  if (stage === 'model_preview') return '当前模型预览'
  if (stage === 'post_turn') return '当前会话'
  if (stage === 'final') return '最终值'
  if (stage === 'pre_generation') return '生成前'
  return '实时值'
}

export const CONTEXT_BUDGET_EXCEEDED = 'AI_CONTEXT_BUDGET_EXCEEDED'
export const CONTEXT_CONFIGURATION_INVALID = 'AI_CONTEXT_CONFIGURATION_INVALID'

export const contextGuardMessage = (code: string | null | undefined): string | null => {
  const normalized = (code || '').trim().toUpperCase()
  if (normalized === CONTEXT_BUDGET_EXCEEDED) {
    return '固定指令或当前输入超出模型可用上下文预算。请缩短输入、减少附件，或新建会话后重试。'
  }
  if (normalized === CONTEXT_CONFIGURATION_INVALID) {
    return '模型上下文配置无效，请联系管理员检查上下文窗口与最大输出上限。'
  }
  return null
}

export const contextGuardCodeFrom = (...values: unknown[]): string | null => {
  for (const value of values) {
    if (typeof value !== 'string') continue
    const upper = value.toUpperCase()
    if (upper.includes(CONTEXT_BUDGET_EXCEEDED)) return CONTEXT_BUDGET_EXCEEDED
    if (upper.includes(CONTEXT_CONFIGURATION_INVALID)) return CONTEXT_CONFIGURATION_INVALID
  }
  return null
}

export const resolveSessionContextUsage = (
  session: Pick<Session, 'latest_context_usage' | 'latestContextUsage'>,
  messages: Array<{ context_usage?: ContextUsageInfo; contextUsage?: ContextUsageInfo }>,
): ContextUsageInfo | null => {
  const sessionUsage = session.latest_context_usage || session.latestContextUsage
  if (sessionUsage) return sessionUsage
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const usage = messages[index]?.context_usage || messages[index]?.contextUsage
    if (usage) return usage
  }
  return null
}

export const contextUsageBelongsToSession = (
  eventSessionId: number | null | undefined,
  activeSessionId: number | null | undefined,
): boolean => {
  if (!activeSessionId || !eventSessionId) return true
  return eventSessionId === activeSessionId
}
