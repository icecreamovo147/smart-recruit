import { describe, expect, it } from 'vitest'
import type { ContextUsageInfo } from '@/types/ai'
import {
  contextBudgetProgressRatio,
  contextBudgetRatio,
  contextBudgetSeverity,
  contextGuardCodeFrom,
  contextGuardMessage,
  currentEffectiveContextTokens,
  formatCompactTokens,
  inputBudgetTokens,
  contextUsageBelongsToSession,
  resolveSessionContextUsage,
} from './contextUsage'

const usage = (patch: Partial<ContextUsageInfo> = {}): ContextUsageInfo => ({
  model_id: 1,
  model_name: 'model',
  context_window_tokens: 8_192,
  max_output_tokens: 1_024,
  prompt_tokens_estimated: 851,
  prompt_tokens_actual: 0,
  completion_tokens_actual: 0,
  total_tokens_actual: 0,
  remaining_tokens_estimated: 5_907,
  usage_ratio: 0.1,
  estimated: true,
  source: 'conservative_estimator',
  stage: 'pre_generation',
  input_budget_tokens: 6_758,
  safety_margin_tokens: 410,
  ...patch,
})

describe('context usage view model', () => {
  it('uses effective input over input budget (not total window)', () => {
    const value = usage()
    expect(currentEffectiveContextTokens(value)).toBe(851)
    expect(inputBudgetTokens(value)).toBe(6_758)
    expect(`${formatCompactTokens(currentEffectiveContextTokens(value))} / ${formatCompactTokens(inputBudgetTokens(value))}`).toBe('851 / 6.8K')
  })

  it('prefers provider actual input and falls back to estimate', () => {
    expect(currentEffectiveContextTokens(usage({ prompt_tokens_actual: 777 }))).toBe(777)
    expect(currentEffectiveContextTokens(usage({ prompt_tokens_actual: Number.NaN, prompt_tokens_estimated: 123 }))).toBe(123)
  })

  it('does not invent a denominator or ratio for unknown configuration', () => {
    const value = usage({ context_window_tokens: 0, input_budget_tokens: 0, budget_status: 'unknown_config' })
    expect(inputBudgetTokens(value)).toBeNull()
    expect(contextBudgetRatio(value)).toBeNull()
    expect(formatCompactTokens(inputBudgetTokens(value))).toBe('—')
    expect(contextBudgetSeverity(value)).toBe('unknown')
  })

  it('uses backend status and applies 60/75 percent fallback boundaries', () => {
    expect(contextBudgetSeverity(usage({ budget_usage_ratio: 0.59 }))).toBe('normal')
    expect(contextBudgetSeverity(usage({ budget_usage_ratio: 0.6 }))).toBe('caution')
    expect(contextBudgetSeverity(usage({ budget_usage_ratio: 0.75 }))).toBe('warning')
    expect(contextBudgetSeverity(usage({ budget_status: 'within_budget', budget_usage_ratio: 0.99 }))).toBe('normal')
    expect(contextBudgetSeverity(usage({ budget_status: 'invalid_config' }))).toBe('danger')
  })

  it('keeps over-100 text ratio but clamps progress ratio', () => {
    const value = usage({ budget_usage_ratio: 1.4 })
    expect(contextBudgetRatio(value)).toBe(1.4)
    expect(contextBudgetProgressRatio(value)).toBe(1)
  })

  it('maps stable guard codes without exposing backend details', () => {
    const code = contextGuardCodeFrom('wrapped: AI_CONTEXT_BUDGET_EXCEEDED internal prompt')
    expect(code).toBe('AI_CONTEXT_BUDGET_EXCEEDED')
    expect(contextGuardMessage(code)).toContain('缩短输入')
    expect(contextGuardMessage('AI_CONTEXT_CONFIGURATION_INVALID')).toContain('联系管理员')
  })

  it('restores the active session snapshot and rejects stale SSE from another session', () => {
    const sessionSnapshot = usage({ prompt_tokens_estimated: 222 })
    const messageSnapshot = usage({ prompt_tokens_estimated: 111 })
    expect(resolveSessionContextUsage(
      { latest_context_usage: sessionSnapshot },
      [{ context_usage: messageSnapshot }],
    )).toBe(sessionSnapshot)
    expect(resolveSessionContextUsage({}, [])).toBeNull()
    expect(contextUsageBelongsToSession(2, 1)).toBe(false)
    expect(contextUsageBelongsToSession(1, 1)).toBe(true)
  })
})
