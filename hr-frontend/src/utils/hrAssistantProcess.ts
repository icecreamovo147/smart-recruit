const hiddenAssistantProcessLines = new Set([
  'Agent run 已开始',
  'Agent run 已完成',
  'Agent run 已取消',
  'Agent run 已部分完成',
  'Agent run 失败',
  'Agent run 保存失败',
  '已选择可用能力',
])

const isHiddenAssistantProcessLine = (line: string): boolean => {
  const text = line.trim()
  return hiddenAssistantProcessLines.has(text)
}

const formatNativeRuntimeProcessJson = (text: string): string => {
  const raw = text.trim()
  if (!raw.startsWith('{')) return text
  try {
    const payload = JSON.parse(raw) as {
      runtime?: string
      fallback_used?: boolean
      tool_count?: number
      context_usage?: {
        prompt_tokens_estimated?: number
        remaining_tokens_estimated?: number
        stage?: string
      }
      tool_results?: Array<{ tool_name?: string; status?: string }>
      display_summary?: string[]
    }
    const runtime = String(payload?.runtime || '').trim()
    const isNativeRuntime = ['native-hr-runtime', 'hr-agent-runtime', 'adk', 'legacy'].includes(runtime)
    if (!payload || !isNativeRuntime) return text
    if (Array.isArray(payload.display_summary) && payload.display_summary.length > 0) {
      return payload.display_summary.filter(Boolean).join('\n')
    }
    const toolNames = (payload.tool_results || []).map((tool) => tool.tool_name || '').filter(Boolean)
    const hasTool = (name: string, status?: string) =>
      (payload.tool_results || []).some((tool) => tool.tool_name === name && (!status || tool.status === status))
    const sentences: string[] = []
    if (toolNames.length) {
      if (hasTool('get_application_snapshot') || hasTool('get_candidate_detail')) {
        sentences.push('我先读取了当前投递记录和候选人详情，确认分析对象与岗位上下文。')
      }
      if (hasTool('get_candidate_match_evaluation', 'error') && hasTool('evaluate_candidate_match', 'success')) {
        sentences.push('没有找到可直接复用的历史匹配评估后，我基于真实简历和岗位要求重新生成了匹配评估。')
      } else if (hasTool('get_candidate_match_evaluation', 'success')) {
        sentences.push('随后我读取了已有匹配评估，用它作为匹配度结论的主要依据。')
      } else if (hasTool('evaluate_candidate_match', 'success')) {
        sentences.push('随后我基于真实简历和岗位要求生成了新的匹配评估。')
      }
      if (hasTool('get_job_detail')) {
        sentences.push('最后我补充读取了岗位详情，用岗位职责和任职要求校验匹配结论。')
      }
      const remaining = toolNames.filter((name) => ![
        'get_application_snapshot',
        'get_candidate_detail',
        'get_candidate_match_evaluation',
        'evaluate_candidate_match',
        'get_job_detail',
      ].includes(name))
      if (sentences.length === 0 && remaining.length > 0) {
        sentences.push(`我调用了 ${remaining.length} 个招聘数据工具，整理实时结果后再生成回复。`)
      }
    } else if (payload.tool_count != null) {
      sentences.push(`我调用了 ${payload.tool_count} 个招聘数据工具，整理实时结果后再生成回复。`)
    }
    if (payload.context_usage?.prompt_tokens_estimated || payload.context_usage?.remaining_tokens_estimated) {
      const prompt = payload.context_usage.prompt_tokens_estimated ?? 0
      const remaining = payload.context_usage.remaining_tokens_estimated ?? 0
      sentences.push(`本次上下文约使用 ${prompt} tokens，剩余约 ${remaining} tokens，可以支撑后续归纳。`)
    }
    if (payload.fallback_used) {
      sentences.push('模型调用异常时，我已使用保守兜底逻辑避免编造数据。')
    }
    return sentences.join('\n') || ''
  } catch {
    return text
  }
}

export const sanitizeAssistantProcessText = (text = ''): string => {
  if (!text) return ''
  const formatted = formatNativeRuntimeProcessJson(text)
  const lines = formatted.split(/\r?\n/)
  const filtered = lines.filter((line) => !isHiddenAssistantProcessLine(line))
  let result = filtered.join('\n')
  if (formatted.endsWith('\n') && result && !result.endsWith('\n')) {
    result += '\n'
  }
  return result
}
