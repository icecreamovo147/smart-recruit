import { describe, expect, it } from 'vitest'
import { friendlyStreamMsg } from './ai'
import { friendlyAgentRunStreamMsg } from './agentRun'
import { friendlyBusinessMessage } from './request'

describe('Context guard error UX', () => {
  const cases = [
    {
      code: 'AI_CONTEXT_BUDGET_EXCEEDED',
      expected: '请缩短输入、减少附件，或新建会话后重试',
    },
    {
      code: 'AI_CONTEXT_CONFIGURATION_INVALID',
      expected: '请联系管理员检查上下文窗口与最大输出上限',
    },
  ]

  it.each(cases)('uses the same safe message for SSE, durable SSE and non-stream errors: $code', ({ code, expected }) => {
    const legacySSE = friendlyStreamMsg(503, `${code}: private prompt must not leak`)
    const durableSSE = friendlyAgentRunStreamMsg(503, `${code}: private prompt must not leak`)
    const nonStream = friendlyBusinessMessage(503, `${code}: private prompt must not leak`)
    expect(legacySSE).toBe(durableSSE)
    expect(durableSSE).toBe(nonStream)
    expect(nonStream).toContain(expected)
    expect(nonStream).not.toContain('private prompt')
  })

  it('uses purchase guidance for exhausted AI credits on every HR transport', () => {
    const expected = 'AI 套餐额度不足，请购买套餐或加量包后重试'
    expect(friendlyStreamMsg(40201, '')).toBe(expected)
    expect(friendlyAgentRunStreamMsg(40201, '')).toBe(expected)
    expect(friendlyBusinessMessage(40201, '')).toBe(expected)
  })
})
