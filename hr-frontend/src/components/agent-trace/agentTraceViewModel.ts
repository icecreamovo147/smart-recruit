import type {
  AgentRunDecision,
  AgentRunItem,
  AgentRunPlanJSON,
  AgentRunRecruitingPlan,
  AgentRunStepItem,
  ToolTraceItem,
} from '@/types/ai'

// ---- Status / type categories ----

export type TraceStatusCategory = 'failed' | 'warning' | 'succeeded' | 'active' | 'info'
export type TraceTypeCategory =
  | 'plan'
  | 'tool'
  | 'evidence'
  | 'memory'
  | 'prompt'
  | 'model'
  | 'fallback'
  | 'legacy'
  | 'other'

export type IssueSeverity = 'error' | 'warning' | 'info'
export type TagType = 'success' | 'warning' | 'danger' | 'info' | 'primary'

export interface PolicyDecisionInfo {
  decision: string
  reason: string
  policyId: number
  label: string
  tagType: TagType
  isIssue: boolean
  severity: IssueSeverity | null
}

export interface JsonDisplayMeta {
  raw: string
  formatted: string
  isValidJson: boolean
  isEmpty: boolean
  previewLength: number
  /** True when at least one nested JSON-encoded string was expanded for display. */
  nestedExpanded?: boolean
}

export interface TraceIssueItem {
  id: string
  severity: IssueSeverity
  label: string
  detail: string
  sourceKind: 'run' | 'step' | 'legacy'
  runId?: number
  stepId?: number
  traceId?: number
  anchorKey: string
}

export interface TraceStepVM {
  step: AgentRunStepItem
  title: string
  typeLabel: string
  typeCategory: TraceTypeCategory
  statusCategory: TraceStatusCategory
  statusLabel: string
  statusTagType: TagType
  isEvidence: boolean
  evidenceSummary: string
  policyDecision: PolicyDecisionInfo | null
  inputDisplay: JsonDisplayMeta
  outputDisplay: JsonDisplayMeta
  searchableText: string
  issues: TraceIssueItem[]
  anchorKey: string
}

export interface TraceRunVM {
  run: AgentRunItem
  agentLabel: string
  modelName: string
  intentLabel: string
  runtimeLabel: string
  statusCategory: TraceStatusCategory
  statusLabel: string
  statusTagType: TagType
  durationMs: number | null
  plan: AgentRunPlanJSON | null
  recruitingPlan: AgentRunRecruitingPlan | null
  riskFlags: string[]
  decision: AgentRunDecision | null
  decisionEntries: Array<{ key: string; value: string; warning: boolean }>
  confirmationRequired: boolean
  steps: TraceStepVM[]
  issues: TraceIssueItem[]
  searchableText: string
  hasStructuredPlan: boolean
  anchorKey: string
}

export interface TraceLegacyVM {
  trace: ToolTraceItem
  title: string
  statusCategory: TraceStatusCategory
  statusLabel: string
  statusTagType: TagType
  typeCategory: TraceTypeCategory
  policyDecision: PolicyDecisionInfo | null
  argsDisplay: JsonDisplayMeta
  resultDisplay: JsonDisplayMeta
  searchableText: string
  issues: TraceIssueItem[]
  anchorKey: string
}

export interface TraceOverviewVM {
  latestRunStatus: string
  latestRunStatusLabel: string
  latestRunStatusCategory: TraceStatusCategory | null
  modelName: string
  intent: string
  intentLabel: string
  runtime: string
  runtimeLabel: string
  runCount: number
  stepCount: number
  toolStepCount: number
  failureCount: number
  warningCount: number
  riskCount: number
  policyIssueCount: number
  legacyTraceCount: number
  durationMs: number | null
  confirmationRequired: boolean
  hasData: boolean
  activeLive: boolean
}

export interface TraceLiveState {
  active: boolean
  runId?: number
  status?: string
  processText?: string
  lastEventSeq?: number
  subscriptionWarning?: string
}

export interface TraceFilterState {
  keyword: string
  statusFilter: 'all' | TraceStatusCategory
  typeFilter: 'all' | TraceTypeCategory
  issueOnly: boolean
}

export interface TraceSessionVM {
  runs: TraceRunVM[]
  legacyTraces: TraceLegacyVM[]
  overview: TraceOverviewVM
  issues: TraceIssueItem[]
  live: TraceLiveState
}

export interface FilteredTraceSessionVM extends TraceSessionVM {
  filter: TraceFilterState
  isFilterEmpty: boolean
  hasActiveFilters: boolean
}

export const DEFAULT_JSON_PREVIEW_LENGTH = 240

export const DEFAULT_FILTER_STATE: TraceFilterState = {
  keyword: '',
  statusFilter: 'all',
  typeFilter: 'all',
  issueOnly: false,
}

export const DEFAULT_LIVE_STATE: TraceLiveState = {
  active: false,
}

// ---- Label maps (aligned with existing AgentTracePanel) ----

export const policyDecisionLabels: Record<string, string> = {
  allow: '允许',
  deny: '拒绝',
  confirmation_required: '需要确认',
  rate_limited: '限流',
  invalid_args: '参数不合规',
  no_policy: '无策略',
}

export const agentLabels: Record<string, string> = {
  hr_recruiting_agent: 'HR 招聘助手',
  hr: 'HR 招聘助手',
  candidate_assistant: '候选人助手',
  custom: '自定义智能体',
}

export const runtimeLabels: Record<string, string> = {
  adk: 'ADK 运行时',
  legacy: '兼容运行时',
  mock: '模拟运行时',
  fallback: '降级运行时',
}

export const intentLabels: Record<string, string> = {
  candidate_match_evaluation: '候选人匹配评估',
  candidate_comparison: '候选人对比',
  analytics: '招聘数据分析',
  status_change_proposal: '状态变更建议',
  interview_prep: '面试准备',
  offer_support: 'Offer 支持',
  unknown: '待澄清意图',
}

export const statusLabels: Record<string, string> = {
  succeeded: '成功',
  success: '成功',
  failed: '失败',
  error: '错误',
  partial: '部分完成',
  fallback: '降级',
  canceled: '已取消',
  cancelled: '已取消',
  running: '运行中',
  pending: '等待中',
  queued: '排队中',
  waiting_confirm: '等待确认',
}

export const stepTypeLabels: Record<string, string> = {
  plan: '规划',
  evidence: '证据',
  memory: '记忆',
  prompt: '技能',
  skill: '技能',
  tool: '工具',
  fallback: '降级',
  recovery: '恢复',
  status: '状态',
  model: '模型',
}

export const decisionKeyLabels: Record<string, string> = {
  intent: '意图',
  confirmation_required: '需要确认',
  confirmation_reason: '确认原因',
  required_tool_count: '所需工具数',
  required_data_count: '所需数据数',
  risk_flag_count: '风险检查数',
  unavailable_tool_risk: '工具不可用风险',
  requires_human_confirm: '需要人工确认',
  requires_evidence_citation: '需要证据引用',
  status: '状态',
  error_type: '错误类型',
  has_answer: '已生成回答',
  completed_at: '完成时间',
  partial: '部分完成',
  failed: '执行失败',
  canceled: '已取消',
  risk_flag_hit: '命中风险',
}

/** Aligned with AgentTracePanel risk chip labels. */
export const riskLabels: Record<string, string> = {
  verify_candidate_identity: '核验候选人身份',
  do_not_infer_from_missing_resume_text: '简历缺失时不做推断',
  cite_tool_returned_evidence: '引用工具返回的证据',
  verify_job_scope: '核验职位范围',
  avoid_unfair_attribute_comparisons: '避免不公平属性对比',
  explain_missing_evaluations: '说明缺失的评估数据',
  state_time_window: '明确统计时间窗口',
  do_not_mix_filtered_and_global_counts: '不要混用筛选数据与全局数据',
  call_tools_for_live_metrics: '实时指标必须调用工具查询',
  verify_application_id: '核验投递记录 ID',
  never_claim_database_updated: '不得声称已直接更新数据库',
  ask_clarifying_question_when_target_status_missing: '目标状态缺失时先追问',
  base_questions_on_resume_and_job_data: '面试题基于简历和职位数据',
  avoid_protected_attribute_questions: '避免受保护属性相关问题',
  ask_for_candidate_or_job_when_ambiguous: '候选人或职位不明确时先追问',
  do_not_send_or_create_offer_without_tool_support: '无工具支持时不得发送或创建 Offer',
  avoid_unverified_compensation_terms: '避免未经核验的薪酬承诺',
  verify_authorization_and_candidate_identity: '核验授权与候选人身份',
  do_not_claim_unavailable_tools: '不得声称使用了不可用工具',
  ask_for_missing_recruiting_intent_or_entities: '招聘意图或对象缺失时先追问',
  no_builtin_recruiting_tools_available: '内置招聘工具不可用',
}

export function riskLabel(value: string): string {
  return labelFrom(riskLabels, value)
}

const WARNING_DECISION_KEYS = new Set([
  'unavailable_tool_risk',
  'requires_human_confirm',
  'risk_flag_hit',
  'partial',
  'failed',
])

// ---- Pure utilities ----

export function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

export function parseJsonObject<T extends Record<string, unknown>>(json: string): T | null {
  if (!json) return null
  try {
    const parsed: unknown = JSON.parse(json)
    return isRecord(parsed) ? (parsed as T) : null
  } catch {
    return null
  }
}

const NESTED_JSON_EXPAND_DEPTH = 6

/**
 * Detect strings that are themselves JSON payloads (common in tool outputs
 * where `result` is a stringified object). Only attempts parse for values that
 * look like objects, arrays, or quoted JSON strings.
 */
export function tryParseEmbeddedJson(value: string): unknown | undefined {
  const trimmed = value.trim()
  if (trimmed.length < 2) return undefined
  const first = trimmed[0]
  if (first !== '{' && first !== '[' && first !== '"') return undefined
  try {
    return JSON.parse(trimmed)
  } catch {
    return undefined
  }
}

/**
 * Recursively expand JSON-encoded string leaves into structured values for
 * display. Does not mutate the input; pure transform with a depth cap.
 */
export function expandNestedJsonValues(
  value: unknown,
  depth: number = NESTED_JSON_EXPAND_DEPTH,
  state: { expanded: boolean } = { expanded: false },
): unknown {
  if (depth <= 0 || value === null || value === undefined) return value

  if (typeof value === 'string') {
    const parsed = tryParseEmbeddedJson(value)
    if (parsed === undefined) return value
    // Expand objects/arrays and further nested JSON strings; keep primitives as parsed.
    state.expanded = true
    if (isRecord(parsed) || Array.isArray(parsed) || typeof parsed === 'string') {
      return expandNestedJsonValues(parsed, depth - 1, state)
    }
    return parsed
  }

  if (Array.isArray(value)) {
    return value.map((item) => expandNestedJsonValues(item, depth - 1, state))
  }

  if (isRecord(value)) {
    const next: Record<string, unknown> = {}
    for (const [key, child] of Object.entries(value)) {
      next[key] = expandNestedJsonValues(child, depth - 1, state)
    }
    return next
  }

  return value
}

export function formatJsonContent(json: string): JsonDisplayMeta {
  const raw = json ?? ''
  if (!raw.trim()) {
    return {
      raw,
      formatted: raw,
      isValidJson: false,
      isEmpty: true,
      previewLength: DEFAULT_JSON_PREVIEW_LENGTH,
      nestedExpanded: false,
    }
  }
  try {
    const parsed: unknown = JSON.parse(raw)
    const expandState = { expanded: false }
    const expanded = expandNestedJsonValues(parsed, NESTED_JSON_EXPAND_DEPTH, expandState)
    const formatted = JSON.stringify(expanded, null, 2)
    return {
      raw,
      formatted,
      isValidJson: true,
      isEmpty: false,
      previewLength: DEFAULT_JSON_PREVIEW_LENGTH,
      nestedExpanded: expandState.expanded,
    }
  } catch {
    // Outer payload is not JSON; still try treating the whole string as embedded JSON text.
    const expandState = { expanded: false }
    const embedded = tryParseEmbeddedJson(raw)
    if (embedded !== undefined) {
      const expanded = expandNestedJsonValues(embedded, NESTED_JSON_EXPAND_DEPTH, expandState)
      return {
        raw,
        formatted: JSON.stringify(expanded, null, 2),
        isValidJson: true,
        isEmpty: false,
        previewLength: DEFAULT_JSON_PREVIEW_LENGTH,
        nestedExpanded: true,
      }
    }
    return {
      raw,
      formatted: raw,
      isValidJson: false,
      isEmpty: false,
      previewLength: DEFAULT_JSON_PREVIEW_LENGTH,
      nestedExpanded: false,
    }
  }
}

export function previewJsonContent(meta: JsonDisplayMeta, expanded: boolean): string {
  const text = meta.formatted
  return expanded ? text : text.slice(0, meta.previewLength)
}

export function shortText(value: unknown): string {
  if (value === null || value === undefined || value === '') return '未记录'
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

export function labelFrom(labels: Record<string, string>, value: unknown): string {
  const text = shortText(value)
  if (text === '未记录') return text
  return labels[text] || text
}

export function statusTagType(status: string): TagType {
  const category = classifyStatus(status)
  if (category === 'succeeded') return 'success'
  if (category === 'failed') return 'danger'
  if (category === 'warning') return 'warning'
  if (category === 'active') return 'primary'
  return 'info'
}

export function classifyStatus(status: string): TraceStatusCategory {
  const normalized = (status || '').toLowerCase()
  if (normalized === 'succeeded' || normalized === 'success') return 'succeeded'
  if (normalized === 'failed' || normalized === 'error') return 'failed'
  if (
    normalized === 'partial'
    || normalized === 'fallback'
    || normalized === 'waiting_confirm'
  ) {
    return 'warning'
  }
  if (
    normalized === 'running'
    || normalized === 'pending'
    || normalized === 'queued'
    || normalized === 'in_progress'
    || normalized === 'active'
  ) {
    return 'active'
  }
  if (normalized === 'canceled' || normalized === 'cancelled') return 'info'
  return 'info'
}

export function policyDecisionTagType(decision: string): TagType {
  if (decision === 'allow') return 'success'
  if (decision === 'deny' || decision === 'invalid_args') return 'danger'
  if (decision === 'confirmation_required' || decision === 'rate_limited') return 'warning'
  return 'info'
}

export function policyDecisionFromObject(value: Record<string, unknown> | null): PolicyDecisionInfo | null {
  if (!value) return null
  const decision = value.policy_decision
  if (typeof decision !== 'string' || !decision) return null
  const policyIdRaw = value.policy_id
  const policyId = typeof policyIdRaw === 'number'
    ? policyIdRaw
    : typeof policyIdRaw === 'string'
      ? Number(policyIdRaw)
      : 0
  const isIssue = decision !== 'allow' && decision !== 'no_policy'
  let severity: IssueSeverity | null = null
  if (decision === 'deny' || decision === 'invalid_args') severity = 'error'
  else if (decision === 'confirmation_required' || decision === 'rate_limited') severity = 'warning'
  else if (isIssue) severity = 'warning'

  return {
    decision,
    reason: typeof value.policy_reason === 'string' ? value.policy_reason : '',
    policyId: Number.isFinite(policyId) ? policyId : 0,
    label: policyDecisionLabels[decision] || decision,
    tagType: policyDecisionTagType(decision),
    isIssue,
    severity,
  }
}

export function policyDecisionFromJson(json: string): PolicyDecisionInfo | null {
  return policyDecisionFromObject(parseJsonObject<Record<string, unknown>>(json))
}

function parseNestedPlanner(value: unknown): AgentRunRecruitingPlan | null {
  if (!value) return null
  if (typeof value === 'string') return parseJsonObject<AgentRunRecruitingPlan>(value)
  return isRecord(value) ? (value as AgentRunRecruitingPlan) : null
}

export function parseRunPlan(run: AgentRunItem): AgentRunPlanJSON | null {
  return parseJsonObject<AgentRunPlanJSON>(run.plan_json)
}

export function parseRecruitingPlan(run: AgentRunItem): AgentRunRecruitingPlan | null {
  const plan = parseRunPlan(run)
  if (!plan) return null
  if (plan.recruiting_plan) return plan.recruiting_plan
  return parseNestedPlanner(plan.planner_json)
}

export function normalizeStringList(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  return value
    .map((item) => {
      if (typeof item === 'string') return item
      if (typeof item === 'number' || typeof item === 'boolean') return String(item)
      return ''
    })
    .filter((item) => item.length > 0)
}

export function normalizeNumberList(value: unknown): number[] {
  if (!Array.isArray(value)) return []
  return value
    .map((item) => {
      if (typeof item === 'number') return item
      if (typeof item === 'string') {
        const parsed = Number(item)
        return Number.isFinite(parsed) ? parsed : null
      }
      return null
    })
    .filter((item): item is number => item !== null)
}

export function computeDurationMs(startedAt?: string, completedAt?: string): number | null {
  if (!startedAt || !completedAt) return null
  const start = new Date(startedAt).getTime()
  const end = new Date(completedAt).getTime()
  if (!Number.isFinite(start) || !Number.isFinite(end) || end < start) return null
  return end - start
}

export function classifyStepType(step: AgentRunStepItem, isEvidence: boolean): TraceTypeCategory {
  if (isEvidence) return 'evidence'
  const type = (step.step_type || '').toLowerCase()
  if (type === 'plan') return 'plan'
  if (type === 'tool') return 'tool'
  if (type === 'memory') return 'memory'
  if (type === 'prompt' || type === 'skill') return 'prompt'
  if (type === 'model') return 'model'
  if (type === 'fallback' || type === 'recovery') return 'fallback'
  if (type === 'evidence') return 'evidence'
  return 'other'
}

export function isEvidenceStep(step: AgentRunStepItem): boolean {
  if (step.step_type === 'evidence') return true
  if (step.step_type !== 'tool') return false
  const output = parseJsonObject<Record<string, unknown>>(step.output_json)
  if (!output) return false
  return Array.isArray(output.evidence) || Array.isArray(output.citations) || Array.isArray(output.sources)
}

export function stepEvidenceSummary(step: AgentRunStepItem): string {
  const output = parseJsonObject<Record<string, unknown>>(step.output_json)
  if (!output) return ''
  const evidence = output.evidence
  if (Array.isArray(evidence)) return `证据 ${evidence.length} 条`
  const citations = output.citations
  if (Array.isArray(citations)) return `引用 ${citations.length} 条`
  const sources = output.sources
  if (Array.isArray(sources)) return `来源 ${sources.length} 条`
  return ''
}

export function stepTitle(step: AgentRunStepItem): string {
  if (step.tool_name) return step.tool_name
  if (step.capability_key) return step.capability_key
  return step.step_type || 'step'
}

function joinSearchParts(parts: Array<string | number | null | undefined>): string {
  return parts
    .map((part) => {
      if (part === null || part === undefined) return ''
      return String(part).trim()
    })
    .filter(Boolean)
    .join(' ')
    .toLowerCase()
}

function localizedValue(value: unknown): string {
  if (typeof value === 'boolean') return value ? '是' : '否'
  if (typeof value === 'string') {
    return statusLabels[value]
      || intentLabels[value]
      || runtimeLabels[value]
      || agentLabels[value]
      || value
  }
  return shortText(value)
}

export function extractRiskFlags(run: AgentRunItem, plan: AgentRunRecruitingPlan | null): string[] {
  const fromRun = normalizeStringList(parseRunPlan(run)?.risk_flags)
  return fromRun.length > 0 ? fromRun : normalizeStringList(plan?.risk_checks)
}

export function extractDecisionEntries(
  decision: AgentRunDecision | null | undefined,
): Array<{ key: string; value: string; warning: boolean }> {
  if (!decision || !isRecord(decision)) return []
  return Object.entries(decision)
    .filter(([, value]) => value !== undefined && value !== null && value !== '')
    .map(([key, value]) => ({
      key: decisionKeyLabels[key] || key,
      value: localizedValue(value),
      warning: value === true && WARNING_DECISION_KEYS.has(key),
    }))
}

export function buildStepVM(step: AgentRunStepItem, runId: number): TraceStepVM {
  const evidence = isEvidenceStep(step)
  const typeCategory = classifyStepType(step, evidence)
  const statusCategory = classifyStatus(step.status)
  const policyDecision = policyDecisionFromJson(step.output_json)
    || policyDecisionFromJson(step.input_json)
  const inputDisplay = formatJsonContent(step.input_json || '')
  const outputDisplay = formatJsonContent(step.output_json || '')
  const title = stepTitle(step)
  const issues: TraceIssueItem[] = []
  const anchorKey = `step-${runId}-${step.id}`

  if (statusCategory === 'failed' || step.error_message) {
    issues.push({
      id: `step-error-${step.id}`,
      severity: 'error',
      label: `步骤失败: ${title}`,
      detail: step.error_message || statusLabels[step.status] || step.status || '失败',
      sourceKind: 'step',
      runId,
      stepId: step.id,
      anchorKey,
    })
  }
  if (policyDecision?.isIssue && policyDecision.severity) {
    issues.push({
      id: `step-policy-${step.id}`,
      severity: policyDecision.severity,
      label: `MCP 策略: ${policyDecision.label}`,
      detail: policyDecision.reason || policyDecision.decision,
      sourceKind: 'step',
      runId,
      stepId: step.id,
      anchorKey,
    })
  }

  const searchableText = joinSearchParts([
    title,
    step.step_type,
    step.tool_name,
    step.capability_key,
    step.capability_source,
    step.status,
    step.error_message,
    policyDecision?.decision,
    policyDecision?.reason,
    policyDecision?.label,
    inputDisplay.formatted,
    outputDisplay.formatted,
  ])

  return {
    step,
    title,
    typeLabel: stepTypeLabels[step.step_type] || step.step_type || '其他',
    typeCategory,
    statusCategory,
    statusLabel: statusLabels[step.status] || step.status || '未记录',
    statusTagType: statusTagType(step.status),
    isEvidence: evidence,
    evidenceSummary: stepEvidenceSummary(step),
    policyDecision,
    inputDisplay,
    outputDisplay,
    searchableText,
    issues,
    anchorKey,
  }
}

export function buildRunVM(run: AgentRunItem): TraceRunVM {
  const plan = parseRunPlan(run)
  const recruitingPlan = parseRecruitingPlan(run)
  const decision = (plan?.decision as AgentRunDecision | undefined) || null
  const riskFlags = extractRiskFlags(run, recruitingPlan)
  const decisionEntries = extractDecisionEntries(decision)
  const steps = (run.steps || []).map((step) => buildStepVM(step, run.id))
  const statusCategory = classifyStatus(run.status)
  const durationMs = computeDurationMs(run.started_at, run.completed_at)
  const intentRaw = recruitingPlan?.intent || decision?.intent || ''
  const runtimeRaw = plan?.runtime || run.agent_type || ''
  const agentName = run.agent_name && run.agent_name !== run.agent_type
    ? run.agent_name
    : run.agent_type || plan?.agent || 'HR Agent'
  const confirmationRequired = Boolean(
    decision?.confirmation_required
    || decision?.requires_human_confirm
    || recruitingPlan?.confirmation_requirement?.required,
  )
  const anchorKey = `run-${run.id}`
  const issues: TraceIssueItem[] = []

  if (statusCategory === 'failed' || run.error_message) {
    issues.push({
      id: `run-error-${run.id}`,
      severity: 'error',
      label: `运行失败: ${labelFrom(agentLabels, agentName)}`,
      detail: run.error_message || run.error_type || statusLabels[run.status] || run.status,
      sourceKind: 'run',
      runId: run.id,
      anchorKey,
    })
  } else if (statusCategory === 'warning') {
    issues.push({
      id: `run-warning-${run.id}`,
      severity: 'warning',
      label: `运行告警: ${labelFrom(agentLabels, agentName)}`,
      detail: statusLabels[run.status] || run.status,
      sourceKind: 'run',
      runId: run.id,
      anchorKey,
    })
  }

  for (const flag of riskFlags) {
    const localizedRisk = riskLabel(flag)
    issues.push({
      id: `run-risk-${run.id}-${flag}`,
      severity: 'warning',
      label: '风险检查',
      detail: localizedRisk,
      sourceKind: 'run',
      runId: run.id,
      anchorKey,
    })
  }

  for (const entry of decisionEntries.filter((item) => item.warning)) {
    issues.push({
      id: `run-decision-${run.id}-${entry.key}`,
      severity: entry.key.includes('失败') ? 'error' : 'warning',
      label: entry.key,
      detail: entry.value,
      sourceKind: 'run',
      runId: run.id,
      anchorKey,
    })
  }

  if (confirmationRequired) {
    issues.push({
      id: `run-confirm-${run.id}`,
      severity: 'warning',
      label: '需要人工确认',
      detail: decision?.confirmation_reason
        || recruitingPlan?.confirmation_requirement?.reason
        || '该运行需要人工确认',
      sourceKind: 'run',
      runId: run.id,
      anchorKey,
    })
  }

  for (const step of steps) {
    issues.push(...step.issues)
  }

  const searchableText = joinSearchParts([
    agentName,
    run.model_name,
    run.status,
    run.error_message,
    run.error_type,
    run.final_answer,
    intentRaw,
    runtimeRaw,
    ...riskFlags,
    ...decisionEntries.map((entry) => `${entry.key} ${entry.value}`),
    ...steps.map((step) => step.searchableText),
  ])

  const hasStructuredPlan = Boolean(
    recruitingPlan
    || normalizeNumberList(plan?.selected_agent_skill_ids).length > 0
    || normalizeNumberList(plan?.selected_memory_ids).length > 0
    || decisionEntries.length > 0,
  )

  return {
    run,
    agentLabel: labelFrom(agentLabels, agentName),
    modelName: run.model_name || '未记录',
    intentLabel: intentRaw ? labelFrom(intentLabels, intentRaw) : '未记录',
    runtimeLabel: runtimeRaw ? labelFrom(runtimeLabels, runtimeRaw) : '未记录',
    statusCategory,
    statusLabel: statusLabels[run.status] || run.status || '未记录',
    statusTagType: statusTagType(run.status),
    durationMs,
    plan,
    recruitingPlan,
    riskFlags,
    decision,
    decisionEntries,
    confirmationRequired,
    steps,
    issues,
    searchableText,
    hasStructuredPlan,
    anchorKey,
  }
}

export function buildLegacyVM(trace: ToolTraceItem): TraceLegacyVM {
  const policyDecision = policyDecisionFromJson(trace.result_content)
  const argsDisplay = formatJsonContent(trace.args_json || '')
  const resultDisplay = formatJsonContent(trace.result_content || '')
  const hasError = Boolean(trace.error_msg)
  const statusCategory: TraceStatusCategory = hasError
    ? 'failed'
    : policyDecision?.isIssue
      ? (policyDecision.severity === 'error' ? 'failed' : 'warning')
      : 'succeeded'
  const statusLabel = hasError
    ? '失败'
    : policyDecision?.label || '成功'
  const anchorKey = `legacy-${trace.id}`
  const issues: TraceIssueItem[] = []

  if (hasError) {
    issues.push({
      id: `legacy-error-${trace.id}`,
      severity: 'error',
      label: `历史工具失败: ${trace.tool_name}`,
      detail: trace.error_msg,
      sourceKind: 'legacy',
      traceId: trace.id,
      anchorKey,
    })
  }
  if (policyDecision?.isIssue && policyDecision.severity) {
    issues.push({
      id: `legacy-policy-${trace.id}`,
      severity: policyDecision.severity,
      label: `MCP 策略: ${policyDecision.label}`,
      detail: policyDecision.reason || policyDecision.decision,
      sourceKind: 'legacy',
      traceId: trace.id,
      anchorKey,
    })
  }

  const searchableText = joinSearchParts([
    trace.tool_name,
    trace.error_msg,
    policyDecision?.decision,
    policyDecision?.reason,
    policyDecision?.label,
    argsDisplay.formatted,
    resultDisplay.formatted,
  ])

  return {
    trace,
    title: trace.tool_name || `trace-${trace.id}`,
    statusCategory,
    statusLabel,
    statusTagType: statusCategory === 'failed'
      ? 'danger'
      : statusCategory === 'warning'
        ? 'warning'
        : 'success',
    typeCategory: 'legacy',
    policyDecision,
    argsDisplay,
    resultDisplay,
    searchableText,
    issues,
    anchorKey,
  }
}

export function buildOverview(
  runs: TraceRunVM[],
  legacyTraces: TraceLegacyVM[],
  live: TraceLiveState = DEFAULT_LIVE_STATE,
): TraceOverviewVM {
  const latest = runs[0] || null
  const allSteps = runs.flatMap((run) => run.steps)
  const failureCount = runs.filter((run) => run.statusCategory === 'failed').length
    + allSteps.filter((step) => step.statusCategory === 'failed').length
    + legacyTraces.filter((trace) => trace.statusCategory === 'failed').length
  const warningCount = runs.filter((run) => run.statusCategory === 'warning').length
    + allSteps.filter((step) => step.statusCategory === 'warning').length
    + legacyTraces.filter((trace) => trace.statusCategory === 'warning').length
  const riskCount = runs.reduce((sum, run) => sum + run.riskFlags.length, 0)
  const policyIssueCount = [
    ...allSteps.map((step) => step.policyDecision),
    ...legacyTraces.map((trace) => trace.policyDecision),
  ].filter((policy) => policy?.isIssue).length

  const durationMs = latest?.durationMs
    ?? runs.reduce<number | null>((acc, run) => {
      if (run.durationMs == null) return acc
      return (acc ?? 0) + run.durationMs
    }, null)

  return {
    latestRunStatus: latest?.run.status || (live.active ? (live.status || 'running') : ''),
    latestRunStatusLabel: latest
      ? latest.statusLabel
      : live.active
        ? (statusLabels[live.status || 'running'] || live.status || '运行中')
        : '未记录',
    latestRunStatusCategory: latest?.statusCategory
      ?? (live.active ? 'active' : null),
    modelName: latest?.modelName || '未记录',
    intent: latest?.recruitingPlan?.intent || latest?.decision?.intent || '',
    intentLabel: latest?.intentLabel || '未记录',
    runtime: latest?.plan?.runtime || latest?.run.agent_type || '',
    runtimeLabel: latest?.runtimeLabel || '未记录',
    runCount: runs.length,
    stepCount: allSteps.length,
    toolStepCount: allSteps.filter((step) => step.typeCategory === 'tool' || step.step.tool_name).length,
    failureCount,
    warningCount,
    riskCount,
    policyIssueCount,
    legacyTraceCount: legacyTraces.length,
    durationMs,
    confirmationRequired: runs.some((run) => run.confirmationRequired),
    hasData: runs.length > 0 || legacyTraces.length > 0,
    activeLive: Boolean(live.active),
  }
}

export function buildTraceSessionVM(
  runs: AgentRunItem[],
  traces: ToolTraceItem[],
  live: TraceLiveState = DEFAULT_LIVE_STATE,
): TraceSessionVM {
  // Do not mutate source arrays; copy before mapping.
  const runVMs = runs.map((run) => buildRunVM(run))
  const legacyVMs = traces.map((trace) => buildLegacyVM(trace))
  const issues = [
    ...runVMs.flatMap((run) => run.issues),
    ...legacyVMs.flatMap((trace) => trace.issues),
  ]
  return {
    runs: runVMs,
    legacyTraces: legacyVMs,
    overview: buildOverview(runVMs, legacyVMs, live),
    issues,
    live: { ...live },
  }
}

function matchesKeyword(text: string, keyword: string): boolean {
  if (!keyword.trim()) return true
  return text.includes(keyword.trim().toLowerCase())
}

function stepMatchesFilters(step: TraceStepVM, filter: TraceFilterState): boolean {
  if (filter.issueOnly && step.issues.length === 0) return false
  if (filter.statusFilter !== 'all' && step.statusCategory !== filter.statusFilter) return false
  if (filter.typeFilter !== 'all' && step.typeCategory !== filter.typeFilter) return false
  if (!matchesKeyword(step.searchableText, filter.keyword)) return false
  return true
}

function runMatchesFilters(run: TraceRunVM, filter: TraceFilterState): boolean {
  if (filter.typeFilter === 'legacy') return false
  if (filter.issueOnly && run.issues.length === 0) return false

  const runTextMatch = matchesKeyword(run.searchableText, filter.keyword)
  const matchingSteps = run.steps.filter((step) => stepMatchesFilters(step, {
    ...filter,
    // When filtering by run-level status, still allow nested step search.
    statusFilter: filter.statusFilter === 'all' ? 'all' : filter.statusFilter,
  }))

  if (filter.typeFilter !== 'all') {
    // Type filters primarily target steps; keep run if any step matches type.
    return matchingSteps.length > 0
  }

  if (filter.statusFilter !== 'all') {
    const runStatusMatch = run.statusCategory === filter.statusFilter
    return (runStatusMatch && (runTextMatch || !filter.keyword.trim())) || matchingSteps.length > 0
  }

  if (filter.issueOnly) {
    return run.issues.length > 0 && (runTextMatch || matchingSteps.length > 0 || !filter.keyword.trim())
  }

  return runTextMatch || matchingSteps.length > 0
}

function legacyMatchesFilters(trace: TraceLegacyVM, filter: TraceFilterState): boolean {
  if (filter.typeFilter !== 'all' && filter.typeFilter !== 'legacy') return false
  if (filter.issueOnly && trace.issues.length === 0) return false
  if (filter.statusFilter !== 'all' && trace.statusCategory !== filter.statusFilter) return false
  if (!matchesKeyword(trace.searchableText, filter.keyword)) return false
  return true
}

export function hasActiveFilters(filter: TraceFilterState): boolean {
  return Boolean(
    filter.keyword.trim()
    || filter.statusFilter !== 'all'
    || filter.typeFilter !== 'all'
    || filter.issueOnly,
  )
}

export function applyTraceFilters(
  session: TraceSessionVM,
  filter: TraceFilterState = DEFAULT_FILTER_STATE,
): FilteredTraceSessionVM {
  const filteredRuns = session.runs
    .filter((run) => runMatchesFilters(run, filter))
    .map((run) => {
      if (filter.typeFilter === 'all' && filter.statusFilter === 'all' && !filter.issueOnly && !filter.keyword.trim()) {
        return run
      }
      const steps = run.steps.filter((step) => {
        // Keep steps that match; if only run-level keyword matched, keep all steps.
        if (filter.keyword.trim() && matchesKeyword(run.searchableText, filter.keyword) && !filter.issueOnly && filter.typeFilter === 'all' && filter.statusFilter === 'all') {
          return true
        }
        return stepMatchesFilters(step, filter) || (
          filter.keyword.trim()
          && matchesKeyword(run.searchableText, filter.keyword)
          && filter.typeFilter === 'all'
          && filter.statusFilter === 'all'
          && !filter.issueOnly
        )
      })
      // Prefer step-level filtering when any filter is active.
      const nextSteps = (filter.typeFilter !== 'all' || filter.issueOnly || filter.statusFilter !== 'all' || filter.keyword.trim())
        ? run.steps.filter((step) => {
          if (filter.typeFilter !== 'all' && step.typeCategory !== filter.typeFilter) {
            // When type filter is set, only matching steps.
            return false
          }
          if (filter.issueOnly && step.issues.length === 0) {
            // For issue-only, keep failed steps or those with issues; also keep empty if run has run-level issues only
            return false
          }
          if (filter.statusFilter !== 'all' && step.statusCategory !== filter.statusFilter) {
            // Allow run-level status filter to still show all steps of matching runs when keyword empty
            if (run.statusCategory === filter.statusFilter && !filter.keyword.trim() && !filter.issueOnly && filter.typeFilter === 'all') {
              return true
            }
            return false
          }
          if (filter.keyword.trim() && !matchesKeyword(step.searchableText, filter.keyword)) {
            // If run itself matched keyword, keep steps that also match or all steps of matching run
            if (matchesKeyword(run.searchableText, filter.keyword) && filter.typeFilter === 'all' && filter.statusFilter === 'all' && !filter.issueOnly) {
              return true
            }
            return false
          }
          return true
        })
        : run.steps

      return {
        ...run,
        steps: nextSteps.length > 0 || filter.typeFilter === 'all' ? (nextSteps.length > 0 ? nextSteps : run.steps) : [],
      }
    })
    .map((run) => {
      // Tighten step list for type/issue filters
      if (filter.typeFilter !== 'all') {
        return {
          ...run,
          steps: run.steps.filter((step) => step.typeCategory === filter.typeFilter),
        }
      }
      if (filter.issueOnly) {
        const issueSteps = run.steps.filter((step) => step.issues.length > 0)
        return {
          ...run,
          steps: issueSteps.length > 0 ? issueSteps : run.steps,
        }
      }
      return run
    })
    .filter((run) => {
      if (filter.typeFilter !== 'all') return run.steps.length > 0
      return true
    })

  const filteredLegacy = session.legacyTraces.filter((trace) => legacyMatchesFilters(trace, filter))
  const issues = [
    ...filteredRuns.flatMap((run) => run.issues),
    ...filteredLegacy.flatMap((trace) => trace.issues),
  ]
  const isFilterEmpty = hasActiveFilters(filter)
    && filteredRuns.length === 0
    && filteredLegacy.length === 0

  return {
    runs: filteredRuns,
    legacyTraces: filteredLegacy,
    overview: session.overview,
    issues,
    live: session.live,
    filter: { ...filter },
    isFilterEmpty,
    hasActiveFilters: hasActiveFilters(filter),
  }
}

export function resetTraceFilters(): TraceFilterState {
  return { ...DEFAULT_FILTER_STATE }
}

/** Frontend-only long-history page size (no backend pagination). */
export const DEFAULT_TRACE_PAGE_SIZE = 20

export interface TraceLazyPage<T> {
  items: T[]
  total: number
  visibleCount: number
  hasMore: boolean
  truncated: boolean
}

/**
 * Slice a list for progressive disclosure without mutating the source.
 * Used instead of backend pagination when client-side rendering is sufficient.
 */
export function paginateTraceItems<T>(
  items: readonly T[],
  visibleCount: number,
  pageSize: number = DEFAULT_TRACE_PAGE_SIZE,
): TraceLazyPage<T> {
  const total = items.length
  const safePage = Number.isFinite(pageSize) && pageSize > 0 ? Math.floor(pageSize) : DEFAULT_TRACE_PAGE_SIZE
  const safeVisible = Math.max(0, Math.min(total, Math.floor(visibleCount)))
  const effectiveVisible = safeVisible > 0 ? safeVisible : Math.min(total, safePage)
  const sliced = items.slice(0, effectiveVisible)
  return {
    items: sliced,
    total,
    visibleCount: sliced.length,
    hasMore: sliced.length < total,
    truncated: total > safePage,
  }
}

export function nextTraceVisibleCount(
  currentVisible: number,
  total: number,
  pageSize: number = DEFAULT_TRACE_PAGE_SIZE,
): number {
  const safePage = Number.isFinite(pageSize) && pageSize > 0 ? Math.floor(pageSize) : DEFAULT_TRACE_PAGE_SIZE
  return Math.min(total, Math.max(safePage, currentVisible) + safePage)
}
