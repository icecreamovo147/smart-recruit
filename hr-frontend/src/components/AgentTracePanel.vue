<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getAgentRuns, getToolTraces } from '@/api/ai'
import type {
  AgentRunItem,
  AgentRunPlanJSON,
  AgentRunRecruitingPlan,
  AgentRunStepItem,
  ToolTraceItem,
} from '@/types/ai'

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

const loadTraces = async () => {
  if (!props.sessionId) {
    traces.value = []
    runs.value = []
    return
  }
  loading.value = true
  try {
    const [runData, traceData] = await Promise.all([
      getAgentRuns(props.sessionId).catch(() => ({ list: [] as AgentRunItem[] })),
      getToolTraces(props.sessionId),
    ])
    runs.value = runData.list || []
    traces.value = traceData.list || []
  } catch (e) {
    traces.value = []
    ElMessage.error('加载执行轨迹失败，请稍后重试')
  } finally {
    loading.value = false
  }
}

const close = () => {
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
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return iso
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
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

const riskFlags = (run: AgentRunItem, plan: AgentRunRecruitingPlan | null): string[] => {
  const fromRun = normalizeStringList(runPlan(run)?.risk_flags)
  return fromRun.length > 0 ? fromRun : normalizeStringList(plan?.risk_checks)
}

const decisionEntries = (run: AgentRunItem): Array<{ key: string; value: string; warning: boolean }> => {
  const decision = runPlan(run)?.decision
  if (!decision || !isRecord(decision)) return []
  return Object.entries(decision)
    .filter(([, value]) => value !== undefined && value !== null && value !== '')
    .map(([key, value]) => ({
      key,
      value: shortText(value),
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
  if (step.tool_name) return step.tool_name
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
watch(() => props.visible, (val) => {
  if (val) {
    loadTraces()
  }
})

// Reload when session changes (panel already open)
watch(() => props.sessionId, () => {
  if (props.visible && props.sessionId) {
    loadTraces()
  }
})
</script>

<template>
  <el-drawer
    :model-value="visible"
    @update:model-value="(val: boolean) => emit('update:visible', val)"
    title="Agent 执行轨迹"
    size="560px"
    :close-on-click-modal="false"
  >
    <div class="trace-panel" v-loading="loading">
      <template v-if="runs.length === 0 && traces.length === 0 && !loading">
        <el-empty description="本次会话暂无 Agent 执行记录" />
      </template>

      <div v-if="runs.length > 0" class="run-list">
        <section
          v-for="run in runs"
          :key="run.id"
          class="run-item"
        >
          <div class="run-item__header">
            <div>
              <div class="run-item__title">{{ run.agent_name || 'HR Agent' }}</div>
              <div class="run-item__meta">
                {{ run.model_name || '未记录模型' }} · {{ formatTime(run.started_at || run.created_at) }}
              </div>
            </div>
            <el-tag :type="statusTagType(run.status)" size="small">
              {{ run.status }}
            </el-tag>
          </div>

          <div v-if="run.plan_json" class="run-summary">
            <template v-if="hasStructuredRunPlan(run)">
              <div class="summary-grid">
                <div class="summary-cell">
                  <span class="summary-cell__label">意图</span>
                  <strong>{{ recruitingPlan(run)?.intent || runPlan(run)?.decision?.intent || '未记录' }}</strong>
                </div>
                <div class="summary-cell">
                  <span class="summary-cell__label">运行时</span>
                  <strong>{{ runPlan(run)?.runtime || run.agent_type || '未记录' }}</strong>
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
                    {{ tool }}
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
                    {{ dataKey }}
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
                    {{ field }}
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
                v-if="selectedSkillIds(run).length || selectedSkillNames(recruitingPlan(run)).length || selectedMemoryIds(run).length || selectedMemoryNames(recruitingPlan(run)).length"
                class="selection-grid"
              >
                <div class="selection-box">
                  <span class="selection-box__label">Skill</span>
                  <div class="chip-list">
                    <el-tag
                      v-for="id in selectedSkillIds(run)"
                      :key="`skill-id-${id}`"
                      size="small"
                      type="warning"
                      effect="plain"
                    >
                      #{{ id }}
                    </el-tag>
                    <el-tag
                      v-for="skill in selectedSkillNames(recruitingPlan(run))"
                      :key="`skill-${skill}`"
                      size="small"
                      type="warning"
                      effect="plain"
                    >
                      {{ skill }}
                    </el-tag>
                    <span v-if="!selectedSkillIds(run).length && !selectedSkillNames(recruitingPlan(run)).length" class="muted">未选择</span>
                  </div>
                </div>
                <div class="selection-box">
                  <span class="selection-box__label">Memory</span>
                  <div class="chip-list">
                    <el-tag
                      v-for="id in selectedMemoryIds(run)"
                      :key="`memory-id-${id}`"
                      size="small"
                      type="primary"
                      effect="plain"
                    >
                      #{{ id }}
                    </el-tag>
                    <el-tag
                      v-for="memory in selectedMemoryNames(recruitingPlan(run))"
                      :key="`memory-${memory}`"
                      size="small"
                      type="primary"
                      effect="plain"
                    >
                      {{ memory }}
                    </el-tag>
                    <span v-if="!selectedMemoryIds(run).length && !selectedMemoryNames(recruitingPlan(run)).length" class="muted">未选择</span>
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
                    {{ risk }}
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
            <div class="final-answer__content">{{ run.final_answer }}</div>
          </div>

          <el-timeline class="run-steps">
            <el-timeline-item
              v-for="step in run.steps"
              :key="step.id"
              :color="step.status === 'failed' ? 'var(--el-color-danger)' : isEvidenceStep(step) ? 'var(--el-color-warning)' : 'var(--el-color-primary)'"
              :timestamp="formatTime(step.started_at || step.created_at)"
            >
              <div class="trace-item" :class="{ 'trace-item--evidence': isEvidenceStep(step) }">
                <div class="trace-item__header">
                  <span class="trace-item__name" :class="{ 'trace-item__name--error': step.status === 'failed' }">
                    {{ stepTitle(step) }}
                  </span>
                  <el-tag :type="stepTagType(step)" size="small" effect="plain">
                    {{ stepTypeLabel(step) }} · {{ step.status }}
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

      <div v-if="traces.length > 0" class="legacy-traces">
        <div class="legacy-traces__title">兼容工具调用明细</div>
        <el-timeline>
          <el-timeline-item
            v-for="item in traces"
            :key="item.id"
            :color="item.error_msg ? 'var(--el-color-danger)' : 'var(--el-color-primary)'"
            :timestamp="formatTime(item.created_at)"
          >
            <div class="trace-item">
              <!-- Tool name -->
              <div class="trace-item__header">
                <span class="trace-item__name" :class="{ 'trace-item__name--error': !!item.error_msg }">
                  {{ item.tool_name }}
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
                <div class="trace-item__label">入参：</div>
                <div
                  class="trace-item__code"
                  :class="{ 'trace-item__code--collapsed': item.args_json.length > 200 && !expandedArgs.has(item.id) }"
                >
                  <pre>{{ expandedArgs.has(item.id) ? formatJson(item.args_json) : formatJson(item.args_json).slice(0, 200) }}</pre>
                </div>
                <button
                  v-if="item.args_json.length > 200"
                  class="trace-item__toggle"
                  @click="toggleArgs(item.id)"
                >
                  {{ expandedArgs.has(item.id) ? '收起' : '展开全部' }}
                </button>
              </div>

              <!-- Result -->
              <div v-if="item.result_content" class="trace-item__section">
                <div class="trace-item__label">结果：</div>
                <div
                  class="trace-item__result"
                  :class="{ 'trace-item__result--collapsed': item.result_content.length > 200 && !expandedResult.has(item.id) }"
                >
                  {{ expandedResult.has(item.id) ? item.result_content : item.result_content.slice(0, 200) }}
                </div>
                <button
                  v-if="item.result_content.length > 200"
                  class="trace-item__toggle"
                  @click="toggleResult(item.id)"
                >
                  {{ expandedResult.has(item.id) ? '收起' : '展开全部' }}
                </button>
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
  padding: 8px 10px;
  margin-bottom: 12px;
  background: var(--el-color-success-light-9);
}

.final-answer__content {
  font-size: 13px;
  line-height: 1.6;
  color: var(--el-text-color-primary);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.run-steps {
  margin-top: 8px;
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

.trace-item--evidence {
  border-left: 3px solid var(--el-color-warning);
  padding-left: 8px;
  background: linear-gradient(90deg, var(--el-color-warning-light-9), transparent 70%);
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
</style>
