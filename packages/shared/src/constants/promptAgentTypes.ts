/**
 * Prompt 绑定的 agent_type 目录。
 *
 * - conversation：聊天助手用（可在 Agent 管理中配置）
 * - structured_task：后台自动任务用（只维护 Prompt，不要求 Agent 配置）
 *
 * 代码/运行时用 value 做查找 key；管理台用 label / group 展示。
 */

export type PromptAgentTypeKind = 'conversation' | 'structured_task'

export interface PromptAgentTypeOption {
  value: string
  label: string
  kind: PromptAgentTypeKind
  /** 简短说明，用于表单提示 */
  description: string
}

/** 合法可选类型（创建/筛选用；legacy hr_agent 仅展示映射，不作为新建选项） */
export const PROMPT_AGENT_TYPE_OPTIONS: PromptAgentTypeOption[] = [
  {
    value: 'hr_recruiting_agent',
    label: 'HR 招聘助手',
    kind: 'conversation',
    description: '给 HR 聊天助手用：用户在 AI 对话里提问时加载',
  },
  {
    value: 'candidate_assistant',
    label: '候选人 AI 助手',
    kind: 'conversation',
    description: '给候选人聊天助手用：候选人端 AI 对话时加载',
  },
  {
    value: 'resume_profile_extractor',
    label: '简历画像抽取',
    kind: 'structured_task',
    description: '给系统后台用：自动解析简历、生成结构化画像时加载',
  },
  {
    value: 'job_requirement_extractor',
    label: '岗位要求抽取',
    kind: 'structured_task',
    description: '给系统后台用：自动从 JD 抽取任职要求时加载',
  },
  {
    value: 'candidate_match_evaluator',
    label: '人岗匹配评估',
    kind: 'structured_task',
    description: '给系统后台用：自动评估候选人与岗位匹配度时加载',
  },
]

const LABEL_BY_VALUE: Record<string, string> = {
  hr_agent: 'HR 招聘助手', // legacy 别名
  ...Object.fromEntries(PROMPT_AGENT_TYPE_OPTIONS.map((item) => [item.value, item.label])),
}

const KIND_BY_VALUE: Record<string, PromptAgentTypeKind> = {
  hr_agent: 'conversation',
  ...Object.fromEntries(PROMPT_AGENT_TYPE_OPTIONS.map((item) => [item.value, item.kind])),
}

export const promptAgentTypeLabel = (value: string): string =>
  LABEL_BY_VALUE[value] || value || '-'

export const promptAgentTypeKind = (value: string): PromptAgentTypeKind | 'unknown' =>
  KIND_BY_VALUE[value] || 'unknown'

/** 列表 tag / 分组标题 */
export const promptAgentTypeKindLabel = (value: string): string => {
  const kind = promptAgentTypeKind(value)
  if (kind === 'conversation') return '对话助手'
  if (kind === 'structured_task') return '系统内置任务'
  return '其他'
}

export const PROMPT_AGENT_TYPE_GROUP_LABEL = {
  conversation: '对话助手',
  structured_task: '系统内置任务',
} as const

export const normalizePromptAgentType = (value: string): string =>
  value === 'hr_agent' ? 'hr_recruiting_agent' : value

export const conversationPromptAgentTypes = PROMPT_AGENT_TYPE_OPTIONS.filter(
  (item) => item.kind === 'conversation',
)

export const structuredTaskPromptAgentTypes = PROMPT_AGENT_TYPE_OPTIONS.filter(
  (item) => item.kind === 'structured_task',
)

export const findPromptAgentTypeOption = (value: string): PromptAgentTypeOption | undefined => {
  const normalized = normalizePromptAgentType(value)
  return PROMPT_AGENT_TYPE_OPTIONS.find((item) => item.value === normalized)
}
