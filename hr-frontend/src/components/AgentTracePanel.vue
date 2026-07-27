<script setup lang="ts">
import { t } from '@shared/i18n'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import DOMPurify from 'dompurify'
import MarkdownIt from 'markdown-it'
import { getAgentRuns, getToolTraces } from '@/api/ai'
import { getActiveAgentRun, subscribeAgentRunEvents } from '@/api/agentRun'
import { isTerminalAgentRunStatus, type AgentRunEvent } from '@shared/types/agentRun'
import type {
  AgentRunItem,
  AgentRunPlanJSON,
  AgentRunRecruitingPlan,
  AgentRunStepItem,
  ToolTraceItem,
} from '@/types/ai'
import { debugLog } from '@shared/utils/debugLog'
import { formatShanghaiDateTime } from '@shared/utils/format'
import TraceOverview from '@/components/agent-trace/TraceOverview.vue'
import TraceFilterBar from '@/components/agent-trace/TraceFilterBar.vue'
import TraceRunSection from '@/components/agent-trace/TraceRunSection.vue'
import TraceLegacySection from '@/components/agent-trace/TraceLegacySection.vue'
import TraceJsonBlock from '@/components/agent-trace/TraceJsonBlock.vue'
import {
  applyTraceFilters,
  buildTraceSessionVM,
  DEFAULT_FILTER_STATE,
  DEFAULT_LIVE_STATE,
  DEFAULT_TRACE_PAGE_SIZE,
  formatToolTitle,
  nextTraceVisibleCount,
  paginateTraceItems,
  resetTraceFilters,
  toolLabel as resolveToolLabel,
  type TraceFilterState,
  type TraceLiveState,
  type TraceRunVM,
  type TraceLegacyVM,
} from '@/components/agent-trace/agentTraceViewModel'

const props = defineProps<{
  sessionId: number | null
  visible: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
}>()

const traces = ref<ToolTraceItem[]>([])
const runs = ref<AgentRunItem[]>([])
const loading = ref(false)
const expandedArgs = ref<Set<number>>(new Set())
const expandedResult = ref<Set<number>>(new Set())
const expandedStepInput = ref<Set<number>>(new Set())
const expandedStepOutput = ref<Set<number>>(new Set())
const liveState = ref<TraceLiveState>({ ...DEFAULT_LIVE_STATE })
const liveWarning = ref('')
let liveAbort: AbortController | null = null
let liveSubscribeToken = 0
let liveReconnectTimer: ReturnType<typeof setTimeout> | null = null
let liveReconnectAttempts = 0

const LIVE_RECONNECT_MAX_ATTEMPTS = 5
const LIVE_RECONNECT_BASE_MS = 400
const LIVE_RECONNECT_MAX_MS = 8000

const sessionVM = computed(() => buildTraceSessionVM(runs.value, traces.value, liveState.value))
const hasTraceData = computed(() => sessionVM.value.overview.hasData || liveState.value.active)

const filterState = ref<TraceFilterState>({ ...DEFAULT_FILTER_STATE })
const activeLayer = ref<'overview' | 'steps' | 'raw' | 'legacy'>('overview')
const visibleRunCount = ref(DEFAULT_TRACE_PAGE_SIZE)
const visibleLegacyCount = ref(DEFAULT_TRACE_PAGE_SIZE)

const filteredVM = computed(() => applyTraceFilters(sessionVM.value, filterState.value))
const visibleRuns = computed(() => filteredVM.value.runs)
const visibleLegacy = computed(() => filteredVM.value.legacyTraces)
const isFilterEmpty = computed(() => filteredVM.value.isFilterEmpty)

const pagedRuns = computed(() => paginateTraceItems(visibleRuns.value, visibleRunCount.value))
const pagedLegacy = computed(() => paginateTraceItems(visibleLegacy.value, visibleLegacyCount.value))
const historyTruncationHint = computed(() => {
  const runTotal = visibleRuns.value.length
  const legacyTotal = visibleLegacy.value.length
  if (runTotal <= DEFAULT_TRACE_PAGE_SIZE && legacyTotal <= DEFAULT_TRACE_PAGE_SIZE) return ''
  return `长历史已分段展示：运行 ${Math.min(visibleRunCount.value, runTotal)}/${runTotal}，兼容轨迹 ${Math.min(visibleLegacyCount.value, legacyTotal)}/${legacyTotal}（前端懒加载，未请求后端分页）`
})

const loadMoreRuns = () => {
  visibleRunCount.value = nextTraceVisibleCount(visibleRunCount.value, visibleRuns.value.length)
}
const loadMoreLegacy = () => {
  visibleLegacyCount.value = nextTraceVisibleCount(visibleLegacyCount.value, visibleLegacy.value.length)
}

watch(filterState, () => {
  visibleRunCount.value = DEFAULT_TRACE_PAGE_SIZE
  visibleLegacyCount.value = DEFAULT_TRACE_PAGE_SIZE
}, { deep: true })

// Wider on desktop; Element Plus accepts CSS length. Mobile uses near-full width via 92vw/100%.
const drawerSize = computed(() => 'min(860px, 100vw)')

const resetFilters = () => {
  filterState.value = resetTraceFilters()
}

/** Map filtered run VMs back to original runs but with filtered steps for timeline rendering. */
const filteredRunItems = computed(() => {
  return pagedRuns.value.items.map((runVM: TraceRunVM) => {
    const stepIds = new Set(runVM.steps.map((s) => s.step.id))
    return {
      ...runVM.run,
      steps: (runVM.run.steps || []).filter((step) => stepIds.has(step.id) || runVM.steps.length === 0),
    } as AgentRunItem
  })
})

const filteredLegacyItems = computed(() =>
  pagedLegacy.value.items.map((item: TraceLegacyVM) => item.trace),
)

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
})

const renderMarkdown = (content: string): string => {
  const raw = DOMPurify.sanitize(md.render(content || ''), {
    ALLOWED_TAGS: [
      'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
      'p', 'br', 'hr',
      'strong', 'b', 'em', 'i', 'u', 's', 'del',
      'ul', 'ol', 'li',
      'code', 'pre',
      'a',
      'table', 'thead', 'tbody', 'tr', 'th', 'td',
      'blockquote',
    ],
    ALLOWED_ATTR: ['href', 'title', 'target'],
    ALLOWED_URI_REGEXP: /^(?:(?:https?|mailto|tel):|[^a-z]|[a-z+.-]+(?:[^a-z+\.\-:]|$))/i,
  })
  return raw.replace(/<a\s/g, '<a rel="noopener noreferrer" ')
}

const loadTraces = async () => {
  if (!props.sessionId) {
    traces.value = []
    runs.value = []
    return
  }
  debugLog.trace.info('loadTraces_started', { session_id: props.sessionId })
  loading.value = true
  try {
    const [runData, traceData] = await Promise.all([
      getAgentRuns(props.sessionId).catch(() => ({ list: [] as AgentRunItem[] })),
      getToolTraces(props.sessionId),
    ])
    runs.value = runData.list || []
    traces.value = traceData.list || []
    debugLog.trace.info('loadTraces_finished', {
      session_id: props.sessionId,
      run_count: runs.value.length,
      trace_count: traces.value.length,
    })
  } catch (e) {
    // Clear both lists so a failed refresh cannot leave stale runs from a prior session.
    runs.value = []
    traces.value = []
    debugLog.trace.error('loadTraces_failed', { session_id: props.sessionId, error: (e as Error)?.message })
    ElMessage.error(t('frontend.operation_failed'))
  } finally {
    loading.value = false
  }
}

const clearLiveReconnectTimer = () => {
  if (liveReconnectTimer != null) {
    clearTimeout(liveReconnectTimer)
    liveReconnectTimer = null
  }
}

const resetLiveReconnect = () => {
  liveReconnectAttempts = 0
  clearLiveReconnectTimer()
}

const stopLiveSubscription = () => {
  clearLiveReconnectTimer()
  if (liveAbort) {
    liveAbort.abort()
    liveAbort = null
  }
  liveSubscribeToken += 1
}

const applyLiveEvent = (event: AgentRunEvent) => {
  const status = event.status || liveState.value.status || 'running'
  const processText = event.snapshot_text
    || event.delta
    || event.display_message
    || event.event_message
    || event.error_message
    || (event.tool_name ? `工具: ${formatToolTitle(event.tool_name)}` : liveState.value.processText)
  liveState.value = {
    active: !isTerminalAgentRunStatus(status),
    runId: event.run_id || liveState.value.runId,
    status,
    processText: processText || '',
    lastEventSeq: event.seq ?? liveState.value.lastEventSeq,
    subscriptionWarning: liveWarning.value || undefined,
  }
}

const refreshDurableRunsQuietly = async () => {
  if (!props.sessionId) return
  try {
    const runData = await getAgentRuns(props.sessionId)
    runs.value = runData.list || []
  } catch {
    // Keep existing durable data on refresh failure.
  }
}

const shouldReconnectLiveSubscription = (runId: number, token: number): boolean => {
  const status = liveState.value.status || ''
  return props.visible
    && Boolean(props.sessionId)
    && token === liveSubscribeToken
    && liveState.value.active
    && liveState.value.runId === runId
    && !isTerminalAgentRunStatus(status)
}

const scheduleLiveReconnect = (runId: number, token: number) => {
  if (!shouldReconnectLiveSubscription(runId, token)) return
  if (liveReconnectAttempts >= LIVE_RECONNECT_MAX_ATTEMPTS) {
    liveWarning.value = liveWarning.value || '实时状态订阅中断，请关闭后重新打开轨迹面板'
    liveState.value = {
      ...liveState.value,
      subscriptionWarning: liveWarning.value,
    }
    return
  }

  const attempt = liveReconnectAttempts
  liveReconnectAttempts += 1
  const delay = Math.min(LIVE_RECONNECT_MAX_MS, LIVE_RECONNECT_BASE_MS * 2 ** attempt)
  clearLiveReconnectTimer()
  liveReconnectTimer = setTimeout(() => {
    liveReconnectTimer = null
    if (!shouldReconnectLiveSubscription(runId, token)) return
    const afterSeq = liveState.value.lastEventSeq || 0
    debugLog.trace.info('traceSubscription_reconnect_scheduled', {
      run_id: runId,
      after_seq: afterSeq,
      attempt: liveReconnectAttempts,
    })
    void startLiveSubscription(runId, afterSeq, { reconnect: true })
  }, delay)
}

const startLiveSubscription = async (
  runId: number,
  afterSeq = 0,
  options: { reconnect?: boolean } = {},
) => {
  stopLiveSubscription()
  const token = ++liveSubscribeToken
  const controller = new AbortController()
  liveAbort = controller
  let terminalSeen = false
  if (!options.reconnect) {
    liveReconnectAttempts = 0
  }
  liveWarning.value = ''
  liveState.value = {
    active: true,
    runId,
    status: liveState.value.status || 'running',
    processText: liveState.value.processText || '实时监听中…',
    lastEventSeq: afterSeq,
  }
  debugLog.trace.info('traceSubscription_started', { run_id: runId, after_seq: afterSeq })
  try {
    await subscribeAgentRunEvents(
      runId,
      afterSeq,
      {
        onEvent: (event) => {
          if (token !== liveSubscribeToken) return
          liveReconnectAttempts = 0
          applyLiveEvent(event)
          if (
            isTerminalAgentRunStatus(event.status)
            || event.event_type === 'run.completed'
            || event.event_type === 'run.canceled'
            || event.event_type === 'run.error'
          ) {
            terminalSeen = true
            void refreshDurableRunsQuietly().finally(() => {
              if (token === liveSubscribeToken) {
                liveState.value = {
                  ...liveState.value,
                  active: false,
                }
              }
            })
          }
        },
        onError: (err) => {
          if (token !== liveSubscribeToken) return
          liveWarning.value = err?.message || '实时状态订阅失败，历史轨迹仍可查看'
          liveState.value = {
            ...liveState.value,
            subscriptionWarning: liveWarning.value,
          }
          debugLog.trace.error('traceSubscription_failed', {
            run_id: runId,
            error: err?.message,
          })
        },
        onDone: () => {
          if (token !== liveSubscribeToken) return
          debugLog.trace.info('traceSubscription_closed', { run_id: runId })
          if (!terminalSeen) {
            scheduleLiveReconnect(runId, token)
          }
        },
      },
      { signal: controller.signal },
    )
  } catch (e) {
    if (token !== liveSubscribeToken) return
    if ((e as Error)?.name === 'AbortError') return
    liveWarning.value = (e as Error)?.message || '实时状态订阅失败，历史轨迹仍可查看'
    liveState.value = {
      ...liveState.value,
      active: Boolean(liveState.value.runId),
      subscriptionWarning: liveWarning.value,
    }
    debugLog.trace.error('traceSubscription_failed', {
      run_id: runId,
      error: (e as Error)?.message,
    })
    scheduleLiveReconnect(runId, token)
  }
}

const checkActiveRun = async () => {
  if (!props.sessionId) {
    stopLiveSubscription()
    resetLiveReconnect()
    liveState.value = { ...DEFAULT_LIVE_STATE }
    liveWarning.value = ''
    return
  }
  debugLog.trace.info('activeRun_check_started', { session_id: props.sessionId })
  try {
    const active = await getActiveAgentRun(props.sessionId, { silentError: true })
    const run = active?.run
    const status = run?.status || ''
    if (run?.run_id && !isTerminalAgentRunStatus(status)) {
      liveState.value = {
        active: true,
        runId: run.run_id,
        status,
        processText: run.process_text || '执行中…',
        lastEventSeq: run.last_event_seq || 0,
      }
      debugLog.trace.info('activeRun_check_finished', {
        session_id: props.sessionId,
        run_id: run.run_id,
        status,
      })
      await startLiveSubscription(run.run_id, run.last_event_seq || 0)
    } else {
      stopLiveSubscription()
      resetLiveReconnect()
      liveState.value = { ...DEFAULT_LIVE_STATE }
      debugLog.trace.info('activeRun_check_finished', {
        session_id: props.sessionId,
        active: false,
      })
    }
  } catch (e) {
    // Non-blocking: keep historical traces.
    liveWarning.value = '无法检查进行中的运行，已展示历史轨迹'
    liveState.value = {
      ...liveState.value,
      subscriptionWarning: liveWarning.value,
    }
    debugLog.trace.error('activeRun_check_finished', {
      session_id: props.sessionId,
      error: (e as Error)?.message,
    })
  }
}

const close = () => {
  stopLiveSubscription()
  emit('update:visible', false)
}

const toggleArgs = (id: number) => {
  const next = new Set(expandedArgs.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedArgs.value = next
}

const toggleResult = (id: number) => {
  const next = new Set(expandedResult.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedResult.value = next
}

const toggleStepInput = (id: number) => {
  const next = new Set(expandedStepInput.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedStepInput.value = next
}

const toggleStepOutput = (id: number) => {
  const next = new Set(expandedStepOutput.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedStepOutput.value = next
}

const formatTime = (iso: string): string => {
  return formatShanghaiDateTime(iso, '')
}

const formatJson = (json: string): string => {
  try {
    return JSON.stringify(JSON.parse(json), null, 2)
  } catch {
    return json
  }
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  value !== null && typeof value === 'object' && !Array.isArray(value)

const parseJsonObject = <T extends Record<string, unknown>>(json: string): T | null => {
  if (!json) return null
  try {
    const parsed: unknown = JSON.parse(json)
    return isRecord(parsed) ? parsed as T : null
  } catch {
    return null
  }
}

interface PolicyDecisionInfo {
  decision: string
  reason: string
  policyId: number
}

const policyDecisionLabels: Record<string, string> = {
  allow: '允许',
  deny: '拒绝',
  confirmation_required: '需要确认',
  rate_limited: '限流',
  invalid_args: '参数不合规',
  no_policy: '无策略',
}


const agentLabels: Record<string, string> = {
  hr_recruiting_agent: 'HR 招聘助手',
  hr: 'HR 招聘助手',
  candidate_assistant: '候选人助手',
  custom: '自定义智能体',
}

const runtimeLabels: Record<string, string> = {
  'native-hr-runtime': 'HR 招聘运行时',
  hr_recruiting_agent: 'HR 招聘助手',
  candidate_ai_assistant: '候选人 AI 助手',
  candidate_assistant: '候选人 AI 助手',
  adk: 'ADK 运行时',
  legacy: '兼容运行时',
  mock: '模拟运行时',
  fallback: '降级运行时',
}

const intentLabels: Record<string, string> = {
  candidate_match_evaluation: '候选人匹配评估',
  candidate_comparison: '候选人对比',
  candidate_lookup: '候选人查询',
  job_listing: '职位列表查询',
  job_detail: '职位详情查询',
  application_listing: '投递列表查询',
  analytics: '招聘数据分析',
  status_change_proposal: '状态变更建议',
  interview_prep: '面试准备',
  offer_support: 'Offer 支持',
  unknown: '待澄清意图',
}

const dataLabels: Record<string, string> = {
  application_id: '投递记录 ID',
  resume_text: '简历文本',
  job_requirements: '职位要求',
  candidate_match_evaluation: '候选人匹配评估',
  job_id: '职位 ID',
  candidate_match_rankings: '候选人匹配排名',
  time_range: '时间范围',
  job_filter_optional: '职位筛选条件（可选）',
  application_counts: '投递数量',
  status_distribution: '状态分布',
  trend: '趋势数据',
  target_status: '目标状态',
  candidate_identity: '候选人身份',
  resume_profile: '简历画像',
  interview_focus_areas: '面试关注点',
  job_details: '职位详情',
  compensation_constraints_optional: '薪酬约束（可选）',
  clarifying_question: '澄清问题',
}

const riskLabels: Record<string, string> = {
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

const outputFieldLabels: Record<string, string> = {
  candidate_match_evaluation: '候选人匹配评估',
  summary: '摘要',
  score: '评分',
  evidence: '证据',
  risks: '风险',
  next_steps: '下一步',
  candidate_comparison: '候选人对比',
  job: '职位',
  ranked_candidates: '候选人排名',
  tradeoffs: '取舍分析',
  recommended_follow_up: '建议跟进',
  analytics: '数据分析',
  metrics: '指标',
  filters: '筛选条件',
  observations: '观察结论',
  caveats: '注意事项',
  status_change_proposal: '状态变更建议',
  candidate: '候选人',
  current_status: '当前状态',
  target_status: '目标状态',
  confirmation_prompt: '确认提示',
  interview_prep: '面试准备',
  candidate_context: '候选人背景',
  focus_areas: '关注领域',
  questions: '面试问题',
  evaluation_rubric: '评估标准',
  offer_support: 'Offer 支持',
  offer_inputs: 'Offer 输入信息',
  draft_points: '草稿要点',
  approval_or_confirmation_needed: '所需审批或确认',
  clarifying_question: '澄清问题',
  known_constraints: '已知约束',
}

const decisionKeyLabels: Record<string, string> = {
  intent: '意图',
  confirmation_required: '需要确认',
  confirmation_reason: '确认原因',
  required_tool_count: '所需工具数',
  required_data_count: '所需数据数',
  risk_flag_count: '风险检查数',
  runtime_warning: '运行告警',
  warning_count: '告警数',
  warning_messages: '告警信息',
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

const statusLabels: Record<string, string> = {
  succeeded: '成功',
  success: '成功',
  failed: '失败',
  error: '错误',
  partial: '部分完成',
  fallback: '降级',
  canceled: '已取消',
  running: '运行中',
  pending: '等待中',
}

const policyDecisionTagType = (decision: string): 'success' | 'warning' | 'danger' | 'info' | 'primary' => {
  if (decision === 'allow') return 'success'
  if (decision === 'deny' || decision === 'invalid_args') return 'danger'
  if (decision === 'confirmation_required' || decision === 'rate_limited') return 'warning'
  return 'info'
}

const policyDecisionFromObject = (value: Record<string, unknown> | null): PolicyDecisionInfo | null => {
  if (!value) return null
  const decision = value.policy_decision
  if (typeof decision !== 'string' || !decision) return null
  const policyIdRaw = value.policy_id
  const policyId = typeof policyIdRaw === 'number'
    ? policyIdRaw
    : typeof policyIdRaw === 'string'
      ? Number(policyIdRaw)
      : 0
  return {
    decision,
    reason: typeof value.policy_reason === 'string' ? value.policy_reason : '',
    policyId: Number.isFinite(policyId) ? policyId : 0,
  }
}

const policyDecisionFromJson = (json: string): PolicyDecisionInfo | null =>
  policyDecisionFromObject(parseJsonObject<Record<string, unknown>>(json))

const policyDecisionFromTrace = (trace: ToolTraceItem): PolicyDecisionInfo | null =>
  policyDecisionFromJson(trace.result_content)

const parseNestedPlanner = (value: unknown): AgentRunRecruitingPlan | null => {
  if (!value) return null
  if (typeof value === 'string') return parseJsonObject<AgentRunRecruitingPlan>(value)
  return isRecord(value) ? value as AgentRunRecruitingPlan : null
}

const runPlan = (run: AgentRunItem): AgentRunPlanJSON | null =>
  parseJsonObject<AgentRunPlanJSON>(run.plan_json)

const runModelDisplayName = (run: AgentRunItem): string => {
  const fromRun = String(run.model_name || '').trim()
  if (fromRun) return fromRun
  const plan = runPlan(run)
  const fromPlan = typeof plan?.model === 'string' ? plan.model.trim() : ''
  if (fromPlan) return fromPlan
  return run.model_id > 0 ? `模型 #${run.model_id}` : '默认模型'
}

const recruitingPlan = (run: AgentRunItem): AgentRunRecruitingPlan | null => {
  const plan = runPlan(run)
  if (!plan) return null
  if (plan.recruiting_plan) return plan.recruiting_plan
  return parseNestedPlanner(plan.planner_json)
}

const normalizeStringList = (value: unknown): string[] => {
  if (!Array.isArray(value)) return []
  return value
    .map((item) => {
      if (typeof item === 'string') return item
      if (typeof item === 'number' || typeof item === 'boolean') return String(item)
      return ''
    })
    .filter((item) => item.length > 0)
}

const normalizeNumberList = (value: unknown): number[] => {
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

const shortText = (value: unknown): string => {
  if (value === null || value === undefined || value === '') return '未记录'
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  return JSON.stringify(value)
}

const booleanLabel = (value: boolean): string => value ? '是' : '否'

const labelFrom = (labels: Record<string, string>, value: unknown): string => {
  const text = shortText(value)
  return labels[text] || text
}

const agentLabel = (value: unknown): string => labelFrom(agentLabels, value)
const runtimeLabel = (value: unknown): string => labelFrom(runtimeLabels, value)
const intentLabel = (value: unknown): string => labelFrom(intentLabels, value)
const toolLabel = (value: string): string => resolveToolLabel(value)
const dataLabel = (value: string): string => labelFrom(dataLabels, value)
const riskLabel = (value: string): string => labelFrom(riskLabels, value)
const outputFieldLabel = (value: string): string => labelFrom(outputFieldLabels, value)
const decisionKeyLabel = (value: string): string => labelFrom(decisionKeyLabels, value)

const localizedValue = (value: unknown): string => {
  if (typeof value === 'boolean') return booleanLabel(value)
  if (typeof value === 'string') {
    return statusLabels[value]
      || intentLabels[value]
      || runtimeLabels[value]
      || agentLabels[value]
      || value
  }
  return shortText(value)
}

const outputSchemaFields = (plan: AgentRunRecruitingPlan | null): string[] => {
  if (!plan?.output_schema || !isRecord(plan.output_schema)) return []
  const properties = plan.output_schema.properties
  if (isRecord(properties)) return Object.keys(properties)
  return Object.keys(plan.output_schema)
}

const capabilityItems = (run: AgentRunItem): string[] => {
  const capabilities = runPlan(run)?.capabilities
  if (!capabilities) return []
  if (Array.isArray(capabilities)) return normalizeStringList(capabilities)
  if (!isRecord(capabilities)) return [shortText(capabilities)]
  return Object.entries(capabilities)
    .map(([key, value]) => `${key}: ${shortText(value)}`)
    .filter((item) => item.length > 0)
}

const selectedSkillIds = (run: AgentRunItem): number[] =>
  normalizeNumberList(runPlan(run)?.selected_agent_skill_ids)

const selectedMemoryIds = (run: AgentRunItem): number[] =>
  normalizeNumberList(runPlan(run)?.selected_memory_ids)

const selectedSkillNames = (plan: AgentRunRecruitingPlan | null): string[] =>
  normalizeStringList(plan?.selected_skills)

const selectedMemoryNames = (plan: AgentRunRecruitingPlan | null): string[] =>
  normalizeStringList(plan?.selected_memories)

interface SelectionChipItem {
  key: string
  label: string
  meta: string
  title: string
}

interface ParsedSelectionLabel {
  id: number | null
  label: string
  title: string
}

const parseSelectionLabel = (raw: string): ParsedSelectionLabel => {
  const text = raw.trim()
  const match = text.match(/^(.+?)\s*\(\s*id\s*:\s*([^,\)]+).*?\)\s*$/i)
  if (!match) {
    return {
      id: null,
      label: text,
      title: text,
    }
  }

  const parsedId = Number(match[2])
  return {
    id: Number.isFinite(parsedId) ? parsedId : null,
    label: match[1].trim() || text,
    title: text,
  }
}

const selectedSelectionItems = (ids: number[], names: string[], keyPrefix: string): SelectionChipItem[] => {
  const byId = new Map<number, SelectionChipItem>()
  const looseItems: SelectionChipItem[] = []

  names.forEach((name, index) => {
    const parsed = parseSelectionLabel(name)
    const item: SelectionChipItem = {
      key: parsed.id !== null ? `${keyPrefix}-${parsed.id}` : `${keyPrefix}-name-${index}-${parsed.label}`,
      label: parsed.label,
      meta: parsed.id !== null ? `#${parsed.id}` : '',
      title: parsed.title,
    }
    if (parsed.id !== null) byId.set(parsed.id, item)
    else looseItems.push(item)
  })

  ids.forEach((id) => {
    if (!byId.has(id)) {
      byId.set(id, {
        key: `${keyPrefix}-${id}`,
        label: `#${id}`,
        meta: '',
        title: `#${id}`,
      })
    }
  })

  return [...byId.values(), ...looseItems]
}

const selectedSkillItems = (run: AgentRunItem, plan: AgentRunRecruitingPlan | null): SelectionChipItem[] =>
  selectedSelectionItems(selectedSkillIds(run), selectedSkillNames(plan), 'skill')

const selectedMemoryItems = (run: AgentRunItem, plan: AgentRunRecruitingPlan | null): SelectionChipItem[] =>
  selectedSelectionItems(selectedMemoryIds(run), selectedMemoryNames(plan), 'memory')

const riskFlags = (run: AgentRunItem, plan: AgentRunRecruitingPlan | null): string[] => {
  const fromRun = normalizeStringList(runPlan(run)?.risk_flags)
  return fromRun.length > 0 ? fromRun : normalizeStringList(plan?.risk_checks)
}

const runAgentName = (run: AgentRunItem): string => {
  if (run.agent_name && run.agent_name !== run.agent_type) return agentLabel(run.agent_name)
  return agentLabel(run.agent_type || runPlan(run)?.agent || 'HR Agent')
}

const decisionEntries = (run: AgentRunItem): Array<{ key: string; value: string; warning: boolean }> => {
  const decision = runPlan(run)?.decision
  if (!decision || !isRecord(decision)) return []
  return Object.entries(decision)
    .filter(([, value]) => value !== undefined && value !== null && value !== '')
    .map(([key, value]) => ({
      key: decisionKeyLabel(key),
      value: localizedValue(value),
      warning: value === true && ['unavailable_tool_risk', 'requires_human_confirm', 'risk_flag_hit', 'partial', 'failed'].includes(key),
    }))
}

const hasStructuredRunPlan = (run: AgentRunItem): boolean => {
  const plan = recruitingPlan(run)
  return !!plan || selectedSkillIds(run).length > 0 || selectedMemoryIds(run).length > 0 || decisionEntries(run).length > 0
}

const statusTagType = (status: string): 'success' | 'warning' | 'danger' | 'info' | 'primary' => {
  if (status === 'succeeded' || status === 'success') return 'success'
  if (status === 'failed' || status === 'error') return 'danger'
  if (status === 'partial' || status === 'fallback') return 'warning'
  if (status === 'canceled') return 'info'
  return 'primary'
}

const stepTagType = (step: AgentRunStepItem): 'success' | 'warning' | 'danger' | 'info' | 'primary' => {
  if (step.status === 'failed') return 'danger'
  if (isEvidenceStep(step)) return 'warning'
  return statusTagType(step.status)
}

const stepTitle = (step: AgentRunStepItem): string => {
  if (step.tool_name) return formatToolTitle(step.tool_name)
  if (step.capability_key) return step.capability_key
  return step.step_type
}

const stepTypeLabel = (step: AgentRunStepItem): string => {
  const labels: Record<string, string> = {
    plan: '规划',
    evidence: '证据',
    memory: '记忆',
    prompt: '技能',
    tool: '工具',
    fallback: '降级',
    recovery: '恢复',
    status: '状态',
    model: '模型',
  }
  return labels[step.step_type] || step.step_type
}

const stepOutputObject = (step: AgentRunStepItem): Record<string, unknown> | null =>
  parseJsonObject<Record<string, unknown>>(step.output_json)

const isEvidenceStep = (step: AgentRunStepItem): boolean => {
  if (step.step_type === 'evidence') return true
  if (step.step_type !== 'tool') return false
  const output = stepOutputObject(step)
  if (!output) return false
  return Array.isArray(output.evidence) || Array.isArray(output.citations) || Array.isArray(output.sources)
}

const stepSummary = (step: AgentRunStepItem): string => {
  const output = stepOutputObject(step)
  if (!output) return ''
  const evidence = output.evidence
  if (Array.isArray(evidence)) return `证据 ${evidence.length} 条`
  const citations = output.citations
  if (Array.isArray(citations)) return `引用 ${citations.length} 条`
  const sources = output.sources
  if (Array.isArray(sources)) return `来源 ${sources.length} 条`
  return ''
}

const previewJson = (json: string, expanded: boolean): string => {
  const text = formatJson(json)
  return expanded ? text : text.slice(0, 240)
}

// Reload when drawer opens with a valid session
watch(() => props.visible, async (val) => {
  if (val) {
    await loadTraces()
    await checkActiveRun()
  } else {
    stopLiveSubscription()
    resetLiveReconnect()
    liveState.value = { ...DEFAULT_LIVE_STATE }
    liveWarning.value = ''
  }
})

// Reload (or clear) when session changes while the panel is open.
// loadTraces() already clears runs/traces when sessionId is null.
watch(() => props.sessionId, async () => {
  stopLiveSubscription()
  resetLiveReconnect()
  liveState.value = { ...DEFAULT_LIVE_STATE }
  liveWarning.value = ''
  if (props.visible) {
    await loadTraces()
    await checkActiveRun()
  }
})

onBeforeUnmount(() => {
  stopLiveSubscription()
  resetLiveReconnect()
})
</script>

<template>
  <el-drawer
    :model-value="visible"
    @update:model-value="(val: boolean) => emit('update:visible', val)"
    title="Agent 执行轨迹"
    :size="drawerSize"
    class="agent-trace-drawer"
    :close-on-click-modal="true"
  >
    <div class="trace-panel" v-loading="loading">
      <template v-if="!hasTraceData && !loading">
        <el-empty description="本次会话暂无 Agent 执行记录" />
      </template>

      <template v-if="hasTraceData">
        <TraceOverview :overview="sessionVM.overview" />
        <div v-if="liveState.active || liveWarning" class="live-status" data-testid="trace-live-status">
          <el-alert
            v-if="liveState.active"
            :title="`实时执行中${liveState.status ? ' · ' + (statusLabels[liveState.status] || liveState.status) : ''}`"
            :description="liveState.processText || '正在接收执行事件'"
            type="info"
            :closable="false"
            show-icon
          />
          <el-alert
            v-if="liveWarning"
            class="live-status__warning"
            :title="liveWarning"
            type="warning"
            :closable="false"
            show-icon
          />
        </div>
        <TraceFilterBar v-model="filterState" @reset="resetFilters" />
        <el-alert
          v-if="historyTruncationHint"
          class="history-truncation"
          data-testid="trace-history-truncation"
          :title="historyTruncationHint"
          type="info"
          :closable="false"
          show-icon
        />

        <div v-if="isFilterEmpty" class="filter-empty" data-testid="trace-filter-empty">
          <el-empty description="没有符合当前筛选条件的轨迹">
            <el-button type="primary" @click="resetFilters">重置筛选</el-button>
          </el-empty>
        </div>

        <el-tabs v-else v-model="activeLayer" class="trace-layers" data-testid="trace-layers">
          <el-tab-pane label="概览/计划" name="overview">
            <TraceRunSection :is-empty="filteredRunItems.length === 0" empty-text="当前筛选下没有运行计划">
              <div v-if="filteredRunItems.length > 0" class="run-list">

        <section
          v-for="run in filteredRunItems"
          :id="'run-' + run.id"
          :key="run.id"
          class="run-item"
        >
          <div class="run-item__header">
            <div>
              <div class="run-item__title">{{ runAgentName(run) }}</div>
              <div class="run-item__meta">
                {{ runModelDisplayName(run) }} · {{ formatTime(run.started_at || run.created_at) }}
              </div>
            </div>
            <el-tag :type="statusTagType(run.status)" size="small">
              {{ statusLabels[run.status] || run.status }}
            </el-tag>
          </div>

          <div v-if="run.plan_json" class="run-summary">
            <template v-if="hasStructuredRunPlan(run)">
              <div class="summary-grid">
                <div class="summary-cell">
                  <span class="summary-cell__label">意图</span>
                  <strong>{{ intentLabel(recruitingPlan(run)?.intent || runPlan(run)?.decision?.intent || '未记录') }}</strong>
                </div>
                <div class="summary-cell">
                  <span class="summary-cell__label">运行时</span>
                  <strong>{{ runtimeLabel(runPlan(run)?.runtime || run.agent_type || '未记录') }}</strong>
                </div>
                <div class="summary-cell">
                  <span class="summary-cell__label">应用</span>
                  <strong>{{ runPlan(run)?.application_bound ? `#${runPlan(run)?.application_id || '-'}` : '未绑定' }}</strong>
                </div>
              </div>

              <div v-if="recruitingPlan(run)?.required_tools?.length" class="compact-section">
                <div class="compact-section__title">计划工具</div>
                <div class="chip-list">
                  <el-tag
                    v-for="tool in recruitingPlan(run)?.required_tools"
                    :key="tool"
                    size="small"
                    effect="plain"
                  >
                    {{ toolLabel(tool) }}
                  </el-tag>
                </div>
              </div>

              <div v-if="recruitingPlan(run)?.required_data?.length" class="compact-section">
                <div class="compact-section__title">所需数据</div>
                <div class="chip-list">
                  <el-tag
                    v-for="dataKey in recruitingPlan(run)?.required_data"
                    :key="dataKey"
                    size="small"
                    type="info"
                    effect="plain"
                  >
                    {{ dataLabel(dataKey) }}
                  </el-tag>
                </div>
              </div>

              <div v-if="outputSchemaFields(recruitingPlan(run)).length" class="compact-section">
                <div class="compact-section__title">输出结构</div>
                <div class="chip-list">
                  <el-tag
                    v-for="field in outputSchemaFields(recruitingPlan(run))"
                    :key="field"
                    size="small"
                    type="success"
                    effect="plain"
                  >
                    {{ outputFieldLabel(field) }}
                  </el-tag>
                </div>
              </div>

              <div v-if="capabilityItems(run).length" class="compact-section">
                <div class="compact-section__title">能力选择</div>
                <div class="compact-list">
                  <span v-for="item in capabilityItems(run)" :key="item">{{ item }}</span>
                </div>
              </div>

              <div
                v-if="selectedSkillItems(run, recruitingPlan(run)).length || selectedMemoryItems(run, recruitingPlan(run)).length"
                class="selection-grid"
              >
                <div class="selection-box">
                  <span class="selection-box__label">Skill</span>
                  <div class="chip-list">
                    <el-tag
                      v-for="skill in selectedSkillItems(run, recruitingPlan(run))"
                      :key="skill.key"
                      class="selection-chip"
                      size="small"
                      type="warning"
                      effect="plain"
                      :title="skill.title"
                    >
                      <span class="selection-chip__text">{{ skill.label }}</span>
                      <span v-if="skill.meta" class="selection-chip__meta">{{ skill.meta }}</span>
                    </el-tag>
                    <span v-if="!selectedSkillItems(run, recruitingPlan(run)).length" class="muted">未选择</span>
                  </div>
                </div>
                <div class="selection-box">
                  <span class="selection-box__label">Memory</span>
                  <div class="chip-list">
                    <el-tag
                      v-for="memory in selectedMemoryItems(run, recruitingPlan(run))"
                      :key="memory.key"
                      class="selection-chip"
                      size="small"
                      type="primary"
                      effect="plain"
                      :title="memory.title"
                    >
                      <span class="selection-chip__text">{{ memory.label }}</span>
                      <span v-if="memory.meta" class="selection-chip__meta">{{ memory.meta }}</span>
                    </el-tag>
                    <span v-if="!selectedMemoryItems(run, recruitingPlan(run)).length" class="muted">未选择</span>
                  </div>
                </div>
              </div>

              <div v-if="riskFlags(run, recruitingPlan(run)).length" class="compact-section">
                <div class="compact-section__title">风险检查</div>
                <div class="chip-list">
                  <el-tag
                    v-for="risk in riskFlags(run, recruitingPlan(run))"
                    :key="risk"
                    size="small"
                    type="danger"
                    effect="plain"
                  >
                    {{ riskLabel(risk) }}
                  </el-tag>
                </div>
              </div>

              <div v-if="decisionEntries(run).length" class="compact-section">
                <div class="compact-section__title">决策</div>
                <div class="decision-list">
                  <span
                    v-for="entry in decisionEntries(run)"
                    :key="entry.key"
                    :class="{ 'decision-list__item--warning': entry.warning }"
                  >
                    {{ entry.key }}: {{ entry.value }}
                  </span>
                </div>
              </div>
            </template>
            <div v-else class="trace-item__section">
              <div class="trace-item__label">Plan：</div>
              <div class="trace-item__code">
                <pre>{{ formatJson(run.plan_json) }}</pre>
              </div>
            </div>
          </div>

          <el-alert
            v-if="run.error_message"
            class="run-item__alert"
            :title="run.error_message"
            :type="run.status === 'partial' ? 'warning' : 'error'"
            :closable="false"
            show-icon
          />

          <div v-if="run.final_answer" class="final-answer">
            <div class="compact-section__title">最终回答</div>
            <div class="final-answer__content md-content" v-html="renderMarkdown(run.final_answer)"></div>
          </div>

          <!-- steps moved to 执行步骤 tab; keep compact error only in overview -->
          <el-timeline v-if="false" class="run-steps">
            <el-timeline-item
              v-for="step in run.steps"
              :key="step.id"
              :color="step.status === 'failed' ? 'var(--el-color-danger)' : isEvidenceStep(step) ? 'var(--el-color-warning)' : 'var(--el-color-primary)'"
              :timestamp="formatTime(step.started_at || step.created_at)"
            >
              <div
                :id="'step-' + run.id + '-' + step.id"
                class="trace-item"
                :class="{ 'trace-item--evidence': isEvidenceStep(step) }"
              >
                <div class="trace-item__header">
                  <span class="trace-item__name" :class="{ 'trace-item__name--error': step.status === 'failed' }">
                    {{ stepTitle(step) }}
                  </span>
                  <el-tag :type="stepTagType(step)" size="small" effect="plain">
                    {{ stepTypeLabel(step) }} · {{ statusLabels[step.status] || step.status }}
                  </el-tag>
                  <el-tag v-if="isEvidenceStep(step)" type="warning" size="small" effect="dark">
                    Evidence
                  </el-tag>
                </div>
                <div class="trace-item__duration">
                  耗时：<strong>{{ step.duration_ms || 0 }}</strong> ms
                  <span v-if="step.capability_source || step.capability_key">
                    · {{ step.capability_source }} {{ step.capability_key }}
                  </span>
                  <span v-if="stepSummary(step)"> · {{ stepSummary(step) }}</span>
                </div>

                <div v-if="policyDecisionFromJson(step.output_json)" class="policy-decision">
                  <el-tag
                    :type="policyDecisionTagType(policyDecisionFromJson(step.output_json)?.decision || '')"
                    size="small"
                    effect="plain"
                  >
                    MCP 策略：{{ policyDecisionLabels[policyDecisionFromJson(step.output_json)?.decision || ''] || policyDecisionFromJson(step.output_json)?.decision }}
                  </el-tag>
                  <span v-if="policyDecisionFromJson(step.output_json)?.policyId" class="policy-decision__meta">
                    #{{ policyDecisionFromJson(step.output_json)?.policyId }}
                  </span>
                  <span v-if="policyDecisionFromJson(step.output_json)?.reason" class="policy-decision__reason">
                    {{ policyDecisionFromJson(step.output_json)?.reason }}
                  </span>
                </div>

                <div v-if="step.input_json" class="trace-item__section">
                  <div class="trace-item__label">输入：</div>
                  <div class="trace-item__code">
                    <pre>{{ previewJson(step.input_json, expandedStepInput.has(step.id)) }}</pre>
                  </div>
                  <button
                    v-if="formatJson(step.input_json).length > 240"
                    class="trace-item__toggle"
                    @click="toggleStepInput(step.id)"
                  >
                    {{ expandedStepInput.has(step.id) ? '收起' : '展开全部' }}
                  </button>
                </div>

                <div v-if="step.output_json" class="trace-item__section">
                  <div class="trace-item__label">输出：</div>
                  <div class="trace-item__code">
                    <pre>{{ previewJson(step.output_json, expandedStepOutput.has(step.id)) }}</pre>
                  </div>
                  <button
                    v-if="formatJson(step.output_json).length > 240"
                    class="trace-item__toggle"
                    @click="toggleStepOutput(step.id)"
                  >
                    {{ expandedStepOutput.has(step.id) ? '收起' : '展开全部' }}
                  </button>
                </div>

                <el-alert
                  v-if="step.error_message"
                  :title="step.error_message"
                  type="error"
                  :closable="false"
                  show-icon
                />
              </div>
            </el-timeline-item>
          </el-timeline>
        </section>
              </div>
            </TraceRunSection>
            <div v-if="pagedRuns.hasMore" class="lazy-more">
              <el-button data-testid="trace-load-more-runs" @click="loadMoreRuns">
                加载更多运行（{{ pagedRuns.visibleCount }}/{{ pagedRuns.total }}）
              </el-button>
            </div>
          </el-tab-pane>

          <el-tab-pane label="执行步骤" name="steps">
            <TraceRunSection :is-empty="filteredRunItems.length === 0" empty-text="当前筛选下没有执行步骤">
              <div v-if="filteredRunItems.length > 0" class="run-list">
                <section
                  v-for="run in filteredRunItems"
                  :key="'steps-' + run.id"
                  class="run-item"
                >
                  <div class="run-item__header">
                    <div>
                      <div class="run-item__title">{{ runAgentName(run) }}</div>
                      <div class="run-item__meta">
                        {{ runModelDisplayName(run) }} · {{ formatTime(run.started_at || run.created_at) }}
                      </div>
                    </div>
                    <el-tag :type="statusTagType(run.status)" size="small">
                      {{ statusLabels[run.status] || run.status }}
                    </el-tag>
                  </div>
                  <el-timeline class="run-steps">
                    <el-timeline-item
                      v-for="step in run.steps"
                      :key="step.id"
                      :color="step.status === 'failed' ? 'var(--el-color-danger)' : isEvidenceStep(step) ? 'var(--el-color-warning)' : 'var(--el-color-primary)'"
                      :timestamp="formatTime(step.started_at || step.created_at)"
                    >
                      <div
                        :id="'step-' + run.id + '-' + step.id"
                        class="trace-item"
                        :class="{ 'trace-item--evidence': isEvidenceStep(step) }"
                      >
                        <div class="trace-item__header">
                          <span class="trace-item__name" :class="{ 'trace-item__name--error': step.status === 'failed' }">
                            {{ stepTitle(step) }}
                          </span>
                          <el-tag :type="stepTagType(step)" size="small" effect="plain">
                            {{ stepTypeLabel(step) }} · {{ statusLabels[step.status] || step.status }}
                          </el-tag>
                          <el-tag v-if="isEvidenceStep(step)" type="warning" size="small" effect="dark">
                            Evidence
                          </el-tag>
                        </div>
                        <div class="trace-item__duration">
                          耗时：<strong>{{ step.duration_ms || 0 }}</strong> ms
                          <span v-if="step.capability_source || step.capability_key">
                            · {{ step.capability_source }} {{ step.capability_key }}
                          </span>
                          <span v-if="stepSummary(step)"> · {{ stepSummary(step) }}</span>
                        </div>
                        <div v-if="policyDecisionFromJson(step.output_json)" class="policy-decision">
                          <el-tag
                            :type="policyDecisionTagType(policyDecisionFromJson(step.output_json)?.decision || '')"
                            size="small"
                            effect="plain"
                          >
                            MCP 策略：{{ policyDecisionLabels[policyDecisionFromJson(step.output_json)?.decision || ''] || policyDecisionFromJson(step.output_json)?.decision }}
                          </el-tag>
                          <span v-if="policyDecisionFromJson(step.output_json)?.reason" class="policy-decision__reason">
                            {{ policyDecisionFromJson(step.output_json)?.reason }}
                          </span>
                        </div>
                        <el-alert
                          v-if="step.error_message"
                          :title="step.error_message"
                          type="error"
                          :closable="false"
                          show-icon
                        />
                      </div>
                    </el-timeline-item>
                  </el-timeline>
                </section>
              </div>
            </TraceRunSection>
            <div v-if="pagedRuns.hasMore" class="lazy-more">
              <el-button data-testid="trace-load-more-runs-steps" @click="loadMoreRuns">
                加载更多运行（{{ pagedRuns.visibleCount }}/{{ pagedRuns.total }}）
              </el-button>
            </div>
          </el-tab-pane>

          <el-tab-pane label="原始数据" name="raw">
            <TraceRunSection :is-empty="filteredRunItems.every(r => !(r.steps || []).length)" empty-text="当前筛选下没有原始数据">
              <div class="run-list">
                <section v-for="run in filteredRunItems" :key="'raw-' + run.id" class="run-item">
                  <div class="run-item__title">{{ runAgentName(run) }}</div>
                  <div class="raw-step-list">
                    <div
                      v-for="step in run.steps"
                      :key="'raw-step-' + step.id"
                      class="trace-item raw-step"
                    >
                      <div class="raw-step__header">
                        <div class="trace-item__name">{{ stepTitle(step) }}</div>
                      </div>
                      <div class="raw-step__body">
                        <div v-if="step.input_json" class="trace-item__section">
                          <TraceJsonBlock :content="step.input_json" label="输入" />
                        </div>
                        <div v-if="step.output_json" class="trace-item__section">
                          <TraceJsonBlock :content="step.output_json" label="输出" />
                        </div>
                      </div>
                    </div>
                  </div>
                </section>
              </div>
            </TraceRunSection>
            <div v-if="pagedRuns.hasMore" class="lazy-more">
              <el-button @click="loadMoreRuns">
                加载更多运行（{{ pagedRuns.visibleCount }}/{{ pagedRuns.total }}）
              </el-button>
            </div>
          </el-tab-pane>

          <el-tab-pane label="兼容轨迹" name="legacy">
            <TraceLegacySection :is-empty="filteredLegacyItems.length === 0">
              <div v-if="filteredLegacyItems.length > 0" class="legacy-traces">
        <el-timeline>
          <el-timeline-item
            v-for="item in filteredLegacyItems"
            :key="item.id"
            :color="item.error_msg ? 'var(--el-color-danger)' : 'var(--el-color-primary)'"
            :timestamp="formatTime(item.created_at)"
          >
            <div :id="'legacy-' + item.id" class="trace-item">
              <!-- Tool name -->
              <div class="trace-item__header">
                <span class="trace-item__name" :class="{ 'trace-item__name--error': !!item.error_msg }">
                  {{ formatToolTitle(item.tool_name) }}
                </span>
                <el-tag
                  v-if="item.error_msg"
                  type="danger"
                  size="small"
                  effect="dark"
                >失败</el-tag>
                <el-tag
                  v-else
                  type="success"
                  size="small"
                  effect="plain"
                >成功</el-tag>
              </div>

              <!-- Duration -->
              <div class="trace-item__duration">
                耗时：<strong>{{ item.duration_ms }}</strong> ms
              </div>

              <div v-if="policyDecisionFromTrace(item)" class="policy-decision">
                <el-tag
                  :type="policyDecisionTagType(policyDecisionFromTrace(item)?.decision || '')"
                  size="small"
                  effect="plain"
                >
                  MCP 策略：{{ policyDecisionLabels[policyDecisionFromTrace(item)?.decision || ''] || policyDecisionFromTrace(item)?.decision }}
                </el-tag>
                <span v-if="policyDecisionFromTrace(item)?.policyId" class="policy-decision__meta">
                  #{{ policyDecisionFromTrace(item)?.policyId }}
                </span>
                <span v-if="policyDecisionFromTrace(item)?.reason" class="policy-decision__reason">
                  {{ policyDecisionFromTrace(item)?.reason }}
                </span>
              </div>

              <!-- Args -->
              <div v-if="item.args_json" class="trace-item__section">
                <TraceJsonBlock :content="item.args_json" label="入参" />
              </div>

              <!-- Result -->
              <div v-if="item.result_content" class="trace-item__section">
                <TraceJsonBlock :content="item.result_content" label="结果" />
              </div>

              <!-- Error message -->
              <div v-if="item.error_msg" class="trace-item__error">
                <el-alert
                  :title="item.error_msg"
                  type="error"
                  :closable="false"
                  show-icon
                />
              </div>
            </div>
          </el-timeline-item>
        </el-timeline>
              </div>
            </TraceLegacySection>
            <div v-if="pagedLegacy.hasMore" class="lazy-more">
              <el-button data-testid="trace-load-more-legacy" @click="loadMoreLegacy">
                加载更多兼容轨迹（{{ pagedLegacy.visibleCount }}/{{ pagedLegacy.total }}）
              </el-button>
            </div>
          </el-tab-pane>
        </el-tabs>
      </template>
    </div>
  </el-drawer>
</template>

<style scoped>
.trace-panel {
  height: 100%;
  overflow-y: auto;
}

.run-list {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.run-item {
  border-bottom: 1px solid var(--el-border-color-lighter);
  padding-bottom: 16px;
}

.run-item__header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.run-item__title {
  font-size: 15px;
  font-weight: 650;
  color: var(--el-text-color-primary);
}

.run-item__meta {
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.run-item__alert {
  margin-bottom: 10px;
}

.run-summary {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 10px;
  margin-bottom: 10px;
  background: var(--el-fill-color-extra-light);
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  margin-bottom: 8px;
}

.summary-cell {
  min-width: 0;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
  padding: 6px 8px;
  background: var(--el-bg-color);
}

.summary-cell__label,
.selection-box__label,
.compact-section__title {
  display: block;
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
}

.summary-cell strong {
  display: block;
  font-size: 13px;
  color: var(--el-text-color-primary);
  overflow-wrap: anywhere;
}

.compact-section {
  margin-top: 8px;
}

.chip-list {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  min-width: 0;
}

.selection-chip {
  max-width: 100%;
  min-width: 0;
}

.selection-chip :deep(.el-tag__content) {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  max-width: 100%;
}

.selection-chip__text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.selection-chip__meta {
  flex: 0 0 auto;
  color: var(--el-text-color-secondary);
}

.compact-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  color: var(--el-text-color-regular);
  font-size: 12px;
}

.compact-list span,
.decision-list span {
  overflow-wrap: anywhere;
}

.selection-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-top: 8px;
}

.selection-box {
  min-width: 0;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
  padding: 8px;
  background: var(--el-bg-color);
}

.muted {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.decision-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
  color: var(--el-text-color-regular);
}

.decision-list__item--warning {
  color: var(--el-color-warning-dark-2);
  font-weight: 600;
}

.final-answer {
  border-left: 3px solid var(--el-color-success);
  border-radius: 0 6px 6px 0;
  padding: 8px 10px;
  margin-bottom: 12px;
  /* color-mix 随 surface 自适应，避免 success-light-9 在未加载 EP dark css-vars 时仍为浅底 */
  background: color-mix(in srgb, var(--el-color-success) 12%, var(--surface));
  color: var(--text-primary);
}

.final-answer__content {
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-primary);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.final-answer__content.md-content {
  white-space: normal;
  color: inherit;
}

.final-answer__content.md-content :deep(h1),
.final-answer__content.md-content :deep(h2),
.final-answer__content.md-content :deep(h3),
.final-answer__content.md-content :deep(strong),
.final-answer__content.md-content :deep(b) {
  color: var(--text-primary);
}

.final-answer__content.md-content :deep(a) {
  color: var(--brand);
}

.final-answer__content.md-content :deep(a:hover) {
  color: var(--brand-strong);
}

.final-answer__content.md-content :deep(code) {
  background: var(--surface-muted);
  color: var(--text-primary);
}

.final-answer__content.md-content :deep(th),
.final-answer__content.md-content :deep(td) {
  border-color: var(--border);
}

.final-answer__content.md-content :deep(th) {
  background: var(--surface-muted);
  color: var(--text-primary);
}

.final-answer__content.md-content :deep(blockquote) {
  background: var(--brand-soft);
  color: var(--text-primary);
  border-left-color: var(--brand);
}

.run-steps {
  margin-top: 8px;
}

.raw-step-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-top: 12px;
}

.raw-step {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
  overflow: hidden;
}

.raw-step__header {
  padding: 10px 12px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  background: var(--el-fill-color-extra-light);
}

.raw-step__header .trace-item__name {
  margin: 0;
}

.raw-step__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
}

.raw-step__body .trace-item__section {
  margin-bottom: 0;
}

.legacy-traces {
  margin-top: 18px;
}

.legacy-traces__title {
  margin-bottom: 10px;
  font-size: 13px;
  font-weight: 650;
  color: var(--el-text-color-primary);
}

.trace-item {
  font-size: 13px;
  line-height: 1.5;
}

/* Evidence：轻量 callout，避免黄色横向渐变在亮/暗色下都显脏 */
.trace-item--evidence {
  margin-top: 2px;
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, var(--el-color-warning) 22%, var(--border));
  border-left: 3px solid var(--el-color-warning);
  border-radius: 8px;
  background: color-mix(in srgb, var(--el-color-warning) 8%, var(--surface));
  box-shadow: inset 0 1px 0 color-mix(in srgb, var(--el-color-warning) 10%, transparent);
}

.trace-item--evidence .trace-item__code,
.trace-item--evidence .trace-item__result {
  background: color-mix(in srgb, var(--surface-muted) 88%, var(--el-color-warning) 12%);
  border: 1px solid color-mix(in srgb, var(--border) 80%, var(--el-color-warning) 20%);
}

.trace-item__header {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 4px;
}

.trace-item__name {
  font-weight: 600;
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.trace-item__name--error {
  color: var(--el-color-danger);
}

.trace-item__duration {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  margin-bottom: 8px;
}

.policy-decision {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.policy-decision__meta {
  color: var(--el-text-color-placeholder);
}

.policy-decision__reason {
  overflow-wrap: anywhere;
}

.trace-item__section {
  margin-bottom: 6px;
}

.trace-item__label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 2px;
}

.trace-item__code {
  background: var(--el-fill-color-light);
  border-radius: 4px;
  padding: 6px 8px;
  overflow-x: auto;
}

.trace-item__code pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: 'SF Mono', 'Fira Code', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.4;
  color: var(--el-text-color-regular);
}

.trace-item__code--collapsed pre {
  max-height: 80px;
  overflow: hidden;
}

.trace-item__result {
  background: var(--el-fill-color-light);
  border-radius: 4px;
  padding: 6px 8px;
  font-size: 12px;
  color: var(--el-text-color-regular);
  word-break: break-all;
}

.trace-item__result--collapsed {
  max-height: 80px;
  overflow: hidden;
}

.trace-item__toggle {
  background: none;
  border: none;
  color: var(--el-color-primary);
  cursor: pointer;
  font-size: 12px;
  padding: 2px 0;
  margin-top: 2px;
}

.trace-item__toggle:hover {
  color: var(--el-color-primary-light-3);
}

.trace-item__error {
  margin-top: 6px;
}

@media (max-width: 640px) {
  .summary-grid,
  .selection-grid {
    grid-template-columns: 1fr;
  }
}

.agent-trace-drawer :deep(.el-drawer) {
  max-width: 100vw;
}

.agent-trace-drawer :deep(.el-drawer__body) {
  overflow-x: hidden;
}

@media (max-width: 767px) {
  .summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .selection-grid {
    grid-template-columns: 1fr;
  }
}

.live-status {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}

.live-status__warning {
  margin-top: 0;
}

.history-truncation {
  margin-bottom: 10px;
}

.lazy-more {
  display: flex;
  justify-content: center;
  margin: 12px 0 4px;
}
</style>
