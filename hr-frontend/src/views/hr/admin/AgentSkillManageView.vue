<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown, CircleCheck, Document, Edit, MoreFilled, Plus, Refresh, Search, Tickets, TurnOff, WarningFilled, View } from '@element-plus/icons-vue'
import * as agentSkillApi from '@/api/agentSkill'
import { listAgentCapabilities } from '@/api/agent'
import { DataTableCard, EmptyGuide, FilterToolbar, PageHeader } from '@/components/admin-console'
import AgentSkillCanvasEditor from '@/components/agent-skill/AgentSkillCanvasEditor.vue'
import type { CapabilityInfo } from '@/types/agent'
import type {
  AgentSkillCanvasEdge,
  AgentSkillCanvasFlow,
  AgentSkillCanvasNode,
  AgentSkillInfo,
  AgentSkillNode,
  AgentSkillNodeType,
  AgentSkillValidation,
  AgentSkillVersionInfo,
  CreateAgentSkillVersionPayload,
  CreateAgentSkillPayload,
  UpdateAgentSkillPayload,
} from '@/types/agentSkill'
import { debugLog } from '@/utils/debugLog'

const api = agentSkillApi

const NODE_TYPES: { type: AgentSkillNodeType; label: string; description: string; placeholder: string }[] = [
  { type: 'trigger', label: '触发场景', description: '定义何时适合使用这个 Agent Skill', placeholder: '例如：当 HR 要求比较候选人与岗位 JD 的匹配度时使用。' },
  { type: 'context', label: '上下文', description: '列出助手应读取或引用的业务信息', placeholder: '例如：候选人简历、岗位职责、任职要求、当前投递状态。' },
  { type: 'instruction', label: '执行指令', description: '说明助手应该如何推理和组织回答', placeholder: '例如：先提取硬性条件，再比较项目经历，最后给出风险提示。' },
  { type: 'condition', label: '条件分支', description: '描述不同输入情况下的处理规则', placeholder: '例如：如果简历缺少关键字段，需要明确说明无法判断。' },
  { type: 'output', label: '输出格式', description: '规定回答结构、字段和语气', placeholder: '例如：输出匹配结论、证据、疑点、下一步建议四段。' },
  { type: 'constraint', label: '约束边界', description: '声明安全、合规、不可做事项', placeholder: '例如：不得编造候选人经历，不得输出歧视性判断。' },
]

const NODE_TYPE_LABEL = NODE_TYPES.reduce<Record<AgentSkillNodeType, string>>((acc, item) => {
  acc[item.type] = item.label
  return acc
}, {} as Record<AgentSkillNodeType, string>)

const AGENT_TYPE_OPTIONS = [
  { value: 'hr_recruiting_agent', label: 'HR 招聘助手' },
  { value: 'candidate_assistant', label: '候选人 AI 助手' },
  { value: 'custom', label: '自定义 Agent' },
]

const SCENARIO_OPTIONS = [
  {
    value: 'candidate_job_match',
    label: '候选人与岗位匹配',
    category: 'candidate_match',
    riskLevel: 'medium',
    tags: ['招聘', '简历筛选', '岗位匹配', '候选人评估'],
    criteria: ['是否逐项对齐 JD 要求', '是否引用简历证据', '是否说明不确定性', '是否避免敏感因素判断'],
  },
  {
    value: 'resume_risk_review',
    label: '简历风险识别',
    category: 'resume_screening',
    riskLevel: 'high',
    tags: ['招聘', '简历筛选', '风险识别'],
    criteria: ['是否标明证据来源', '是否区分事实和推断', '是否避免敏感因素判断'],
  },
  {
    value: 'interview_question_generation',
    label: '面试题生成',
    category: 'interview',
    riskLevel: 'medium',
    tags: ['招聘', '面试准备', '问题生成'],
    criteria: ['问题是否围绕岗位要求', '是否覆盖关键能力', '是否可用于面试验证'],
  },
  {
    value: 'interview_feedback_summary',
    label: '面评总结',
    category: 'interview',
    riskLevel: 'high',
    tags: ['招聘', '面试评估', '面评总结'],
    criteria: ['是否基于面试记录', '是否列出优势和风险', '是否避免绝对化录用结论'],
  },
  {
    value: 'job_requirement_extraction',
    label: 'JD 要求提取',
    category: 'job_analysis',
    riskLevel: 'low',
    tags: ['招聘', '岗位分析', 'JD 解析'],
    criteria: ['是否区分硬性要求和加分项', '是否提取岗位职责', '是否保留不确定信息'],
  },
  {
    value: 'recruiting_analytics',
    label: '招聘数据分析',
    category: 'recruiting_analytics',
    riskLevel: 'low',
    tags: ['招聘', '数据分析', '投递分析'],
    criteria: ['是否引用指标口径', '是否说明时间范围', '是否给出可执行建议'],
  },
]

const CATEGORY_OPTIONS = [
  { value: 'general', label: '通用' },
  { value: 'resume_screening', label: '简历筛选' },
  { value: 'candidate_match', label: '候选人匹配' },
  { value: 'interview', label: '面试' },
  { value: 'job_analysis', label: '岗位分析' },
  { value: 'recruiting_analytics', label: '招聘数据分析' },
  { value: 'candidate_communication', label: '候选人沟通' },
]

const OUTPUT_SCHEMA_OPTIONS = [
  { value: '', label: '普通文本', schema: '' },
  {
    value: 'analysis_report',
    label: '结构化分析报告',
    schema: '{"type":"object","properties":{"conclusion":{"type":"string"},"evidence":{"type":"array","items":{"type":"string"}},"risks":{"type":"array","items":{"type":"string"}},"next_steps":{"type":"array","items":{"type":"string"}}}}',
  },
  {
    value: 'match_result',
    label: '匹配评分结果',
    schema: '{"type":"object","properties":{"match_level":{"type":"string"},"score":{"type":"number"},"matched_requirements":{"type":"array","items":{"type":"string"}},"gaps":{"type":"array","items":{"type":"string"}}}}',
  },
  {
    value: 'risk_list',
    label: '风险清单',
    schema: '{"type":"object","properties":{"risks":{"type":"array","items":{"type":"object","properties":{"level":{"type":"string"},"description":{"type":"string"},"evidence":{"type":"string"}}}}}}',
  },
  { value: 'custom', label: '自定义 JSON Schema', schema: '' },
]

const list = ref<AgentSkillInfo[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const saving = ref(false)
const editLoading = ref(false)
const previewLoading = ref(false)
const savedPreviewLoading = ref(false)
const versionsLoading = ref(false)
const statusChangingId = ref<number | null>(null)
const activatingVersionId = ref<number | null>(null)
const regeneratingEmbeddingId = ref<number | null>(null)
const keyword = ref('')
const statusFilter = ref('')
const previewMarkdown = ref('')
const validation = ref<AgentSkillValidation>({ valid: false, errors: [], warnings: [] })
const selectedNodeId = ref('')
const builderDialogVisible = ref(false)
const previewDrawerVisible = ref(false)
const savedPreviewDrawerVisible = ref(false)
const versionsDrawerVisible = ref(false)
const editingSkill = ref<AgentSkillInfo | null>(null)
const savedPreviewSkill = ref<AgentSkillInfo | null>(null)
const savedPreviewVersion = ref<AgentSkillVersionInfo | null>(null)
const versionSkill = ref<AgentSkillInfo | null>(null)
const versions = ref<AgentSkillVersionInfo[]>([])
const selectedVersion = ref<AgentSkillVersionInfo | null>(null)
const changeNote = ref('')
const flowIntegrityWarning = ref('')
const advancedConfigVisible = ref(false)
const capabilitiesLoading = ref(false)
const capabilityList = ref<CapabilityInfo[]>([])
const outputSchemaMode = ref('')
const form = reactive({
  name: '',
  display_name: '',
  description: '',
  agent_type: 'hr_recruiting_agent',
  category: 'general',
  scenario: '',
  priority: 0,
  risk_level: 'medium',
  required_capabilities: [] as string[],
  output_schema: '',
  evaluation_criteria: [] as string[],
  semantic_tags: [] as string[],
  version: '1.0.0',
  is_enabled: true,
})

function createCanvasNode(type: AgentSkillNodeType, index: number): AgentSkillCanvasNode {
  const config = NODE_TYPES.find((item) => item.type === type)!
  return {
    id: `${type}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    type,
    position: {
      x: 80 + (index % 2) * 300,
      y: 80 + Math.floor(index / 2) * 150,
    },
    data: {
      title: config.label,
      content: '',
    },
  }
}

const NODE_WIDTH = 184
const NODE_HEIGHT = 104

const nodeCenter = (node: AgentSkillCanvasNode) => ({
  x: node.position.x + NODE_WIDTH / 2,
  y: node.position.y + NODE_HEIGHT / 2,
})

const inferSequentialEdgeHandles = (
  source: AgentSkillCanvasNode,
  target: AgentSkillCanvasNode,
): Pick<AgentSkillCanvasEdge, 'sourceHandle' | 'targetHandle'> => {
  const sourceCenter = nodeCenter(source)
  const targetCenter = nodeCenter(target)
  const dx = targetCenter.x - sourceCenter.x
  const dy = targetCenter.y - sourceCenter.y
  if (Math.abs(dx) >= Math.abs(dy)) {
    return dx >= 0
      ? { sourceHandle: 'right', targetHandle: 'left' }
      : { sourceHandle: 'left', targetHandle: 'right' }
  }
  return dy >= 0
    ? { sourceHandle: 'bottom', targetHandle: 'top' }
    : { sourceHandle: 'top', targetHandle: 'bottom' }
}

const createSequentialEdge = (
  source: AgentSkillCanvasNode,
  target: AgentSkillCanvasNode,
  index = 0,
): AgentSkillCanvasEdge => ({
  id: `edge-${source.id}-${target.id}-${index}`,
  source: source.id,
  target: target.id,
  ...inferSequentialEdgeHandles(source, target),
})

function createDefaultFlow(): AgentSkillCanvasFlow {
  const nodes = (['trigger', 'context', 'instruction', 'output', 'constraint'] as AgentSkillNodeType[])
    .map((type, index) => createCanvasNode(type, index))
  return {
    format: 'canvas.v1',
    version: '1.0.0',
    type: 'agent-skill',
    nodes,
    edges: nodes.slice(0, -1).map((node, index) => createSequentialEdge(node, nodes[index + 1], index)),
    viewport: { x: 0, y: 0, zoom: 1 },
  }
}

const flow = ref<AgentSkillCanvasFlow>(createDefaultFlow())

const isEditing = computed(() => Boolean(editingSkill.value))
const saveButtonText = computed(() => (isEditing.value ? '保存' : '创建 Agent Skill'))
const currentVersion = computed(() => (
  versions.value.find((item) => item.id === versionSkill.value?.current_version_id)
  || versions.value.find((item) => item.is_current)
  || null
))

const loadList = async () => {
  debugLog.skill.info('loadList_started', { page: page.value, keyword: keyword.value, status_filter: statusFilter.value })
  loading.value = true
  try {
    const data = await api.listAgentSkills({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
      enabled_only: statusFilter.value === 'enabled' ? true : undefined,
    })
    const rawList = data.list || []
    list.value = statusFilter.value === 'disabled'
      ? rawList.filter((item) => !item.is_enabled)
      : rawList
    total.value = data.total || list.value.length
    debugLog.skill.info('loadList_finished', { total: total.value, returned: list.value.length })
  } catch (e: unknown) {
    debugLog.skill.error('loadList_failed', { error: getErrorMessage(e, '') })
    ElMessage.error(getErrorMessage(e, 'Agent Skill 列表加载失败'))
  } finally {
    loading.value = false
  }
}

const resetBuilder = () => {
  editingSkill.value = null
  form.name = ''
  form.display_name = ''
  form.description = ''
  form.agent_type = 'hr_recruiting_agent'
  form.category = 'general'
  form.scenario = 'candidate_job_match'
  form.priority = 0
  form.risk_level = 'medium'
  form.required_capabilities = []
  form.output_schema = ''
  form.evaluation_criteria = []
  form.semantic_tags = []
  form.version = '1.0.0'
  form.is_enabled = true
  outputSchemaMode.value = ''
  advancedConfigVisible.value = false
  changeNote.value = ''
  flowIntegrityWarning.value = ''
  flow.value = createDefaultFlow()
  selectedNodeId.value = flow.value.nodes[0]?.id || ''
  previewMarkdown.value = ''
  applyScenarioPreset(form.scenario)
  validation.value = { valid: false, errors: [], warnings: [] }
}

const openCreate = () => {
  resetBuilder()
  loadCapabilities()
  builderDialogVisible.value = true
}

const closeBuilderDialog = () => {
  builderDialogVisible.value = false
}

const resetOrCloseBuilder = () => {
  if (isEditing.value) {
    closeBuilderDialog()
    return
  }
  resetBuilder()
}

const canvasNodes = computed<AgentSkillNode[]>(() => flow.value.nodes.map((node, index) => ({
  id: node.id,
  type: node.type,
  title: node.data.title,
  content: node.data.content,
  order: index + 1,
})))

const selectedNode = computed(() => (
  flow.value.nodes.find((node) => node.id === selectedNodeId.value)
  || null
))

const buildLocalValidation = (): AgentSkillValidation => {
  const errors: string[] = []
  const warnings: string[] = []
  if (!form.name.trim()) errors.push('请填写唯一标识')
  if (!/^[a-z][a-z0-9_-]{1,127}$/.test(form.name.trim())) errors.push('唯一标识只能使用小写字母开头，包含小写字母、数字、下划线或短横线，长度 2-128')
  if (!form.display_name.trim()) errors.push('请填写显示名称')
  if (!form.description.trim()) errors.push('请填写技能描述，描述会写入 SKILL.md frontmatter')
  if (!form.agent_type.trim()) errors.push('请填写适用 Agent 类型')
  if (!form.category.trim()) errors.push('请填写治理分类')
  if (!['low', 'medium', 'high', 'critical'].includes(form.risk_level)) errors.push('风险等级只能是 low / medium / high / critical')
  if (form.priority < -1000 || form.priority > 1000) errors.push('优先级范围为 -1000 到 1000')
  if (form.output_schema.trim()) {
    try {
      JSON.parse(form.output_schema)
    } catch {
      errors.push('输出 Schema 必须是合法 JSON')
    }
  }
  warnings.push(...skillGovernanceWarnings(editingSkill.value))
  if (!canvasNodes.value.some((node) => node.type === 'trigger' && node.content.trim())) errors.push('至少需要一个有内容的触发场景节点')
  if (!canvasNodes.value.some((node) => node.type === 'instruction' && node.content.trim())) errors.push('至少需要一个有内容的执行指令节点')
  if (!canvasNodes.value.some((node) => node.type === 'output' && node.content.trim())) errors.push('至少需要一个有内容的输出格式节点')
  if (!canvasNodes.value.some((node) => node.type === 'constraint' && node.content.trim())) errors.push('至少需要一个有内容的约束边界节点')
  if (!flow.value.nodes.length) errors.push('画布至少需要一个节点')
  return { valid: errors.length === 0, errors, warnings }
}

const workflowNodeText = (node: AgentSkillCanvasNode) => {
  const content = node.data.content.trim()
  const title = node.data.title.trim() || NODE_TYPE_LABEL[node.type]
  return `${NODE_TYPE_LABEL[node.type]}：${content || title}`
}

const localWorkflowLines = () => {
  const nodes = flow.value.nodes
  const nodeMap = new Map(nodes.map((node) => [node.id, node]))
  const validEdges = flow.value.edges.filter((edge) => nodeMap.has(edge.source) && nodeMap.has(edge.target))
  if (!validEdges.length) return []
  const indegree = new Map(nodes.map((node) => [node.id, 0]))
  const outgoing = new Map<string, AgentSkillCanvasEdge[]>()
  validEdges.forEach((edge) => {
    indegree.set(edge.target, (indegree.get(edge.target) || 0) + 1)
    outgoing.set(edge.source, [...(outgoing.get(edge.source) || []), edge])
  })
  outgoing.forEach((edges) => edges.sort((a, b) => {
    const left = nodes.findIndex((node) => node.id === a.target)
    const right = nodes.findIndex((node) => node.id === b.target)
    return left - right
  }))
  const starts = nodes.filter((node) => node.type === 'trigger' && (indegree.get(node.id) || 0) === 0)
  const fallbackStarts = starts.length ? starts : nodes.filter((node) => (indegree.get(node.id) || 0) === 0)
  const entryNodes = fallbackStarts.length ? fallbackStarts : nodes
  const visited = new Set<string>()
  const visiting = new Set<string>()
  const lines: string[] = ['## Workflow', '']
  const visit = (node: AgentSkillCanvasNode) => {
    if (visited.has(node.id) || visiting.has(node.id)) return
    visiting.add(node.id)
    lines.push(`1. ${workflowNodeText(node)}`)
    const nextEdges = outgoing.get(node.id) || []
    if (nextEdges.length > 1) {
      nextEdges.forEach((edge) => {
        const target = nodeMap.get(edge.target)
        if (!target) return
        lines.push(`   - 如果 ${edge.label?.trim() || '选择该路径'}，继续到「${target.data.title || NODE_TYPE_LABEL[target.type]}」。`)
      })
    }
    nextEdges.forEach((edge) => {
      const target = nodeMap.get(edge.target)
      if (target) visit(target)
    })
    visiting.delete(node.id)
    visited.add(node.id)
  }
  entryNodes.forEach(visit)
  nodes.forEach((node) => {
    if (!visited.has(node.id)) visit(node)
  })
  lines.push('')
  return lines
}

const localMarkdown = computed(() => {
  const lines = [
    `# ${form.display_name.trim() || '未命名 Agent Skill'}`,
    '',
    '## Metadata',
    `- name: ${form.name.trim() || 'agent.skill.name'}`,
    `- version: ${form.version.trim() || '1.0.0'}`,
    `- category: ${form.category.trim() || 'recruiting'}`,
  ]
  if (form.description.trim()) {
    lines.push(`- description: ${form.description.trim()}`)
  }
  const workflowLines = localWorkflowLines()
  if (workflowLines.length) {
    lines.push('', ...workflowLines)
  }
  canvasNodes.value.forEach((node, index) => {
    lines.push('', `## ${index + 1}. ${node.title || NODE_TYPE_LABEL[node.type]}`, '', `> node_type: ${node.type}`, '')
    lines.push(node.content.trim() || `（待补充${NODE_TYPE_LABEL[node.type]}）`)
  })
  return lines.join('\n')
})

const flowJson = computed(() => JSON.stringify({
  ...flow.value,
  format: 'canvas.v1',
  version: flow.value.version || '1.0.0',
  type: 'agent-skill',
  nodes: flow.value.nodes.map((node) => ({
    id: node.id,
    type: node.type,
    position: node.position,
    data: {
      title: node.data.title?.trim() || NODE_TYPE_LABEL[node.type],
      content: node.data.content?.trim() || '',
    },
  })),
  edges: flow.value.edges,
  viewport: flow.value.viewport || { x: 0, y: 0, zoom: 1 },
}))

const uniqueList = (values: string[]) => Array.from(new Set(values.map((item) => item.trim()).filter(Boolean)))

const slugifySkillName = (value: string) => value
  .trim()
  .toLowerCase()
  .replace(/[^a-z0-9_-]+/g, '_')
  .replace(/^_+|_+$/g, '')
  .replace(/_{2,}/g, '_')


const riskLevelLabel = (value?: string) => {
  const labels: Record<string, string> = {
    low: '低',
    medium: '中',
    high: '高',
    critical: '关键',
  }
  return labels[value || ''] || value || '-'
}

const riskLevelTagType = (value?: string) => {
  if (value === 'critical' || value === 'high') return 'danger'
  if (value === 'medium') return 'warning'
  return 'info'
}

const skillGovernanceWarnings = (skill?: AgentSkillInfo | null) => [
  ...(skill?.unavailable_capabilities || []).map((item) => `不可用能力：${item}`),
  ...(skill?.validation_warnings || []),
]

const selectedScenario = computed(() => SCENARIO_OPTIONS.find((item) => item.value === form.scenario) || null)

const capabilitySelectOptions = computed(() => capabilityList.value
  .filter((item) => item.source !== 'skill')
  .map((item) => ({
    value: `${item.source}:${item.key}`,
    label: capabilityDisplayLabel(item),
    disabled: !item.is_available,
  })))

const capabilityDisplayLabel = (capability: CapabilityInfo) => {
  const sourceLabel: Record<string, string> = { builtin: '内置', mcp: 'MCP', skill: 'Skill' }
  const title = capability.display_name || capability.name || capability.key
  return `${sourceLabel[capability.source] || capability.source} / ${title}`
}

const applyScenarioPreset = (scenarioValue: string) => {
  const preset = SCENARIO_OPTIONS.find((item) => item.value === scenarioValue)
  if (!preset) return
  form.category = preset.category
  form.risk_level = preset.riskLevel
  form.semantic_tags = uniqueList([...form.semantic_tags, ...preset.tags])
  form.evaluation_criteria = uniqueList([...form.evaluation_criteria, ...preset.criteria])
  if (!form.name.trim()) {
    const slug = slugifySkillName(preset.value)
    form.name = slug || `agent_skill_${Date.now().toString(36)}`
  }
}

const handleScenarioChange = (value: string) => {
  applyScenarioPreset(value)
}

const ensureGeneratedSkillName = () => {
  if (form.name.trim() || isEditing.value) return
  const scenarioSlug = slugifySkillName(form.scenario)
  const displaySlug = slugifySkillName(form.display_name)
  form.name = scenarioSlug || displaySlug || `agent_skill_${Date.now().toString(36)}`
}

const handleOutputSchemaModeChange = (value: string) => {
  const option = OUTPUT_SCHEMA_OPTIONS.find((item) => item.value === value)
  if (!option) return
  if (value !== 'custom') {
    form.output_schema = option.schema
  }
}

const loadCapabilities = async (agentType = form.agent_type) => {
  capabilitiesLoading.value = true
  try {
    const data = await listAgentCapabilities(agentType)
    capabilityList.value = data.list || []
  } catch {
    capabilityList.value = []
  } finally {
    capabilitiesLoading.value = false
  }
}

const handleAgentTypeChange = async () => {
  form.required_capabilities = []
  await loadCapabilities(form.agent_type)
}

const payload = (): CreateAgentSkillPayload => ({
  name: form.name.trim(),
  display_name: form.display_name.trim(),
  description: form.description.trim(),
  agent_type: form.agent_type.trim(),
  category: form.category.trim(),
  scenario: form.scenario.trim(),
  priority: Number(form.priority) || 0,
  risk_level: form.risk_level,
  required_capabilities: form.required_capabilities,
  output_schema: form.output_schema.trim(),
  evaluation_criteria: form.evaluation_criteria,
  semantic_tags: form.semantic_tags,
  version: form.version.trim(),
  is_enabled: form.is_enabled,
  is_enabled_set: true,
  is_manual_invocable: true,
  is_manual_invocable_set: true,
  activate: true,
  flow_json: flowJson.value,
  nodes: canvasNodes.value.map((node, index) => ({
    id: node.id,
    type: node.type,
    title: node.title.trim() || NODE_TYPE_LABEL[node.type],
    content: node.content.trim(),
    order: index + 1,
  })),
})

const updatePayload = (): UpdateAgentSkillPayload => ({
  display_name: form.display_name.trim(),
  display_name_set: true,
  description: form.description.trim(),
  description_set: true,
  agent_type: form.agent_type.trim(),
  agent_type_set: true,
  category: form.category.trim(),
  category_set: true,
  scenario: form.scenario.trim(),
  scenario_set: true,
  priority: Number(form.priority) || 0,
  priority_set: true,
  risk_level: form.risk_level,
  risk_level_set: true,
  required_capabilities: form.required_capabilities,
  required_capabilities_set: true,
  output_schema: form.output_schema.trim(),
  output_schema_set: true,
  evaluation_criteria: form.evaluation_criteria,
  evaluation_criteria_set: true,
  semantic_tags: form.semantic_tags,
  semantic_tags_set: true,
  is_enabled: form.is_enabled,
  is_enabled_set: true,
  is_manual_invocable: true,
  is_manual_invocable_set: true,
})

const versionPayload = (): CreateAgentSkillVersionPayload => ({
  version: form.version.trim(),
  skill_md: previewMarkdown.value || localMarkdown.value,
  flow_json: flowJson.value,
  nodes: canvasNodes.value.map((node, index) => ({
    id: node.id,
    type: node.type,
    title: node.title.trim() || NODE_TYPE_LABEL[node.type],
    content: node.content.trim(),
    order: index + 1,
  })),
  change_note: changeNote.value.trim() || undefined,
  activate: true,
})

const getErrorMessage = (e: unknown, fallback: string) => (
  (e as { response?: { data?: { message?: string } }; message?: string })?.response?.data?.message
  || (e as { message?: string })?.message
  || fallback
)

const normalizeVersionList = (data: AgentSkillVersionInfo[] | { list: AgentSkillVersionInfo[] }) => (
  Array.isArray(data) ? data : data.list || []
)

const normalizeSkill = (data: AgentSkillInfo | { skill: AgentSkillInfo }) => (
  'skill' in data ? data.skill : data
)

const nextVersionText = (version?: string) => {
  const value = version?.trim()
  if (!value) return '1.0.0'
  const semver = value.match(/^(\d+)\.(\d+)\.(\d+)$/)
  if (semver) {
    return `${semver[1]}.${semver[2]}.${Number(semver[3]) + 1}`
  }
  const integer = value.match(/^(\d+)$/)
  if (integer) {
    return String(Number(integer[1]) + 1)
  }
  return `${value}.${new Date().toISOString().replace(/[-:TZ.]/g, '').slice(0, 14)}`
}

const buildSequentialEdges = (nodes: AgentSkillCanvasNode[]): AgentSkillCanvasEdge[] => (
  nodes.slice(0, -1).map((node, index) => createSequentialEdge(node, nodes[index + 1], index))
)

const normalizeCanvasEdges = (
  edges: AgentSkillCanvasFlow['edges'] | undefined,
  nodes: AgentSkillCanvasNode[],
  fallbackToSequential = false,
): AgentSkillCanvasEdge[] => {
  const nodeIds = new Set(nodes.map((node) => node.id))
  const validEdges = Array.isArray(edges)
    ? edges
      .filter((edge) => edge.source && edge.target && nodeIds.has(edge.source) && nodeIds.has(edge.target))
      .map((edge, index) => ({
        ...edge,
        id: edge.id || `edge-${edge.source}-${edge.target}-${index}`,
      }))
    : []
  if (validEdges.length > 0) return validEdges
  return fallbackToSequential ? buildSequentialEdges(nodes) : []
}

const flowEdgeIntegrityWarning = (value?: string) => {
  if (!value) return ''
  try {
    const parsed = JSON.parse(value) as Partial<AgentSkillCanvasFlow>
    if (!Array.isArray(parsed.nodes)) return ''
    const isCanvasFlow = parsed.format === 'canvas.v1' || parsed.type === 'agent-skill'
    if (!isCanvasFlow || parsed.nodes.length <= 1) return ''
    const nodeIds = new Set(parsed.nodes.map((node, index) => node.id || `node-${index}`))
    const validEdgeCount = Array.isArray(parsed.edges)
      ? parsed.edges.filter((edge) => edge.source && edge.target && nodeIds.has(edge.source) && nodeIds.has(edge.target)).length
      : 0
    if (validEdgeCount === 0) {
      return '当前历史版本没有保存有效的节点连接关系。请在画布中重新连线后发布新版本，系统不会自动猜测错误连线。'
    }
    const missingHandleCount = parsed.edges?.filter((edge) => !edge.sourceHandle || !edge.targetHandle).length || 0
    if (missingHandleCount > 0) {
      return '当前历史版本的部分连接缺少锚点信息，回显时会使用默认左右连接点。重新发布新版本后会完整保存锚点。'
    }
    return ''
  } catch {
    return ''
  }
}

const parseFlowJson = (value?: string): AgentSkillCanvasFlow | null => {
  if (!value) return null
  try {
    const parsed = JSON.parse(value) as Partial<AgentSkillCanvasFlow>
    if (!Array.isArray(parsed.nodes)) return null
    const isCanvasFlow = parsed.format === 'canvas.v1' || parsed.type === 'agent-skill'
    const nodes = parsed.nodes.map((node, index) => ({
      id: node.id || `node-${index}`,
      type: node.type,
      position: {
        x: Number.isFinite(node.position?.x) ? node.position!.x : 80 + (index % 2) * 300,
        y: Number.isFinite(node.position?.y) ? node.position!.y : 80 + Math.floor(index / 2) * 150,
      },
      data: {
        title: node.data?.title || ('title' in node ? String(node.title || '') : '') || NODE_TYPE_LABEL[node.type] || '节点',
        content: node.data?.content || ('content' in node ? String(node.content || '') : ''),
      },
    }))
    return {
      format: 'canvas.v1',
      version: parsed.version || '1.0.0',
      type: 'agent-skill',
      nodes,
      edges: normalizeCanvasEdges(parsed.edges, nodes, !isCanvasFlow),
      viewport: parsed.viewport || { x: 0, y: 0, zoom: 1 },
    }
  } catch {
    return null
  }
}

const flowFromNodes = (nodes?: AgentSkillNode[]): AgentSkillCanvasFlow => {
  if (!nodes?.length) return createDefaultFlow()
  const sorted = [...nodes].sort((a, b) => (a.order || 0) - (b.order || 0))
  const canvasNodes = sorted.map((node, index) => ({
    id: node.id || `${node.type}-${index}`,
    type: node.type,
    position: {
      x: 80 + (index % 2) * 300,
      y: 80 + Math.floor(index / 2) * 150,
    },
    data: {
      title: node.title || NODE_TYPE_LABEL[node.type],
      content: node.content || '',
    },
  }))
  return {
    format: 'canvas.v1',
    version: '1.0.0',
    type: 'agent-skill',
    nodes: canvasNodes,
    edges: buildSequentialEdges(canvasNodes),
    viewport: { x: 0, y: 0, zoom: 1 },
  }
}

const loadVersions = async (skillId: number) => normalizeVersionList(await api.listAgentSkillVersions(skillId))

const findCurrentVersion = (skill: AgentSkillInfo, versionList: AgentSkillVersionInfo[]) => (
  versionList.find((item) => item.id === skill.current_version_id)
  || versionList.find((item) => item.is_current)
  || ('current_version' in skill ? (skill.current_version as AgentSkillVersionInfo | undefined) : undefined)
  || null
)

const refreshPreview = async () => {
  debugLog.skill.info('refreshPreview_started', {})
  ensureGeneratedSkillName()
  const localValidation = buildLocalValidation()
  validation.value = localValidation
  previewMarkdown.value = localMarkdown.value
  if (!localValidation.valid) {
    debugLog.skill.warn('refreshPreview_skipped', { reason: 'local_validation_failed', errors: localValidation.errors.length })
    return
  }
  previewLoading.value = true
  try {
    const data = await api.previewAgentSkill(payload())
    const serverValidation = (data as { validation?: AgentSkillValidation }).validation
    previewMarkdown.value = data.skill_md || localMarkdown.value
    validation.value = serverValidation || { ...localValidation, valid: localValidation.errors.length === 0 }
    debugLog.skill.info('refreshPreview_finished', { valid: validation.value.valid })
  } catch (e: unknown) {
    debugLog.skill.error('refreshPreview_failed', { error: getErrorMessage(e, '') })
    ElMessage.error(getErrorMessage(e, 'SKILL.md 预览生成失败'))
  } finally {
    previewLoading.value = false
  }
}

const openPreviewDrawer = async () => {
  previewDrawerVisible.value = true
  await refreshPreview()
}

const saveSkill = async () => {
  debugLog.skill.info('saveSkill_started', { is_edit: !!editingSkill.value, skill_id: editingSkill.value?.id, name: form.name })
  if (saving.value) return
  if (editingSkill.value) {
    try {
      await ElMessageBox.confirm(
        '保存将覆盖当前版本的 Agent Skill，确定继续？',
        '确认保存',
        { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' },
      )
    } catch {
      debugLog.skill.info('saveSkill_cancelled', { skill_id: editingSkill.value.id })
      return
    }
  }
  await refreshPreview()
  if (!validation.value.valid) {
    debugLog.skill.warn('saveSkill_skipped', { reason: 'validation_failed' })
    ElMessage.warning('请先修复校验错误')
    return
  }
  saving.value = true
  try {
    if (editingSkill.value) {
      const skillId = editingSkill.value.id
      await api.updateAgentSkill(skillId, updatePayload())
      await api.createAgentSkillVersion(skillId, versionPayload())
      ElMessage.success('Agent Skill 已保存并激活新版本')
      debugLog.skill.info('saveSkill_succeeded', { skill_id: skillId, action: 'update' })
    } else {
      await api.createAgentSkill(payload())
      ElMessage.success('Agent Skill 已创建')
      debugLog.skill.info('saveSkill_succeeded', { action: 'create', name: form.name })
    }
    builderDialogVisible.value = false
    await loadList()
  } catch (e: unknown) {
    debugLog.skill.error('saveSkill_failed', { error: getErrorMessage(e, '') })
    ElMessage.error(getErrorMessage(e, 'Agent Skill 保存失败'))
  } finally {
    saving.value = false
  }
}

const toggleStatus = async (row: AgentSkillInfo) => {
  debugLog.skill.info('toggleStatus_started', { skill_id: row.id, new_enabled: !row.is_enabled })
  if (statusChangingId.value) return
  statusChangingId.value = row.id
  try {
    await api.updateAgentSkillStatus(row.id, { is_enabled: !row.is_enabled })
    ElMessage.success(row.is_enabled ? 'Agent Skill 已停用' : 'Agent Skill 已启用')
    debugLog.skill.info('toggleStatus_finished', { skill_id: row.id, is_enabled: !row.is_enabled })
    await loadList()
  } catch (e: unknown) {
    debugLog.skill.error('toggleStatus_failed', { skill_id: row.id, error: getErrorMessage(e, '') })
    ElMessage.error(getErrorMessage(e, '状态更新失败'))
  } finally {
    statusChangingId.value = null
  }
}

const regenerateEmbedding = async (row: AgentSkillInfo) => {
  debugLog.skill.info('regenerateEmbedding_started', { skill_id: row.id })
  if (regeneratingEmbeddingId.value) return
  try {
    await ElMessageBox.confirm(
      `确认重新生成「${row.display_name || row.name}」的 Embedding？该操作会覆盖当前默认模型下用于语义召回的向量。`,
      '重新生成 Embedding',
      {
        confirmButtonText: '重新生成',
        cancelButtonText: '取消',
        type: 'warning',
      },
    )
  } catch {
    debugLog.skill.info('regenerateEmbedding_cancelled', { skill_id: row.id })
    return
  }
  regeneratingEmbeddingId.value = row.id
  try {
    const result = await api.regenerateAgentSkillEmbedding(row.id)
    if (result.failed_count > 0) {
      ElMessage.error('Embedding 重新生成失败，请检查模型配置或服务日志')
      return
    }
    if (result.skipped_count > 0) {
      ElMessage.warning('当前 Agent Skill 未发布、未启用或无可生成内容，已跳过')
      return
    }
    ElMessage.success('Embedding 已重新生成')
    debugLog.skill.info('regenerateEmbedding_succeeded', { skill_id: row.id, success_count: result.success_count })
  } catch (e: unknown) {
    debugLog.skill.error('regenerateEmbedding_failed', { skill_id: row.id, error: getErrorMessage(e, '') })
    ElMessage.error(getErrorMessage(e, 'Embedding 重新生成失败'))
  } finally {
    regeneratingEmbeddingId.value = null
  }
}

const openEdit = async (row: AgentSkillInfo) => {
  debugLog.skill.info('openEdit_started', { skill_id: row.id, name: row.name })
  if (editLoading.value) return
  editLoading.value = true
  try {
    const detail = normalizeSkill(await api.getAgentSkill(row.id))
    const versionList = await loadVersions(row.id)
    const current = findCurrentVersion(detail, versionList)
    editingSkill.value = detail
    form.name = detail.name
    form.display_name = detail.display_name || detail.name
    form.description = detail.description || ''
    form.agent_type = detail.agent_type || 'hr_recruiting_agent'
    form.category = detail.category || 'general'
    form.scenario = detail.scenario || ''
    form.priority = detail.priority || 0
    form.risk_level = detail.risk_level || 'medium'
    form.required_capabilities = [...(detail.required_capabilities || [])]
    form.output_schema = detail.output_schema || ''
    outputSchemaMode.value = detail.output_schema ? 'custom' : ''
    form.evaluation_criteria = [...(detail.evaluation_criteria || [])]
    form.semantic_tags = [...(detail.semantic_tags || [])]
    form.version = nextVersionText(current?.version)
    form.is_enabled = detail.is_enabled
    advancedConfigVisible.value = true
    await loadCapabilities(form.agent_type)
    changeNote.value = ''
    const rawFlowJson = current?.flow_json || detail.flow_json
    flow.value = parseFlowJson(rawFlowJson) || flowFromNodes(detail.node_schema)
    flowIntegrityWarning.value = flowEdgeIntegrityWarning(rawFlowJson)
    selectedNodeId.value = flow.value.nodes[0]?.id || ''
    previewMarkdown.value = current?.skill_md || detail.skill_md || localMarkdown.value
    validation.value = buildLocalValidation()
    debugLog.skill.info('openEdit_finished', { skill_id: row.id, version: form.version, node_count: flow.value.nodes.length, edge_count: flow.value.edges.length })
    builderDialogVisible.value = true
  } catch (e: unknown) {
    debugLog.skill.error('openEdit_failed', { skill_id: row.id, error: getErrorMessage(e, '') })
    ElMessage.error(getErrorMessage(e, '加载 Agent Skill 详情失败'))
  } finally {
    editLoading.value = false
  }
}

const openSavedPreview = async (row: AgentSkillInfo) => {
  debugLog.skill.info('openSavedPreview_started', { skill_id: row.id })
  savedPreviewDrawerVisible.value = true
  savedPreviewLoading.value = true
  savedPreviewSkill.value = row
  savedPreviewVersion.value = null
  try {
    const detail = normalizeSkill(await api.getAgentSkill(row.id))
    const versionList = await loadVersions(row.id)
    savedPreviewSkill.value = detail
    savedPreviewVersion.value = findCurrentVersion(detail, versionList)
    debugLog.skill.info('openSavedPreview_finished', { skill_id: row.id, has_current_version: !!savedPreviewVersion.value })
  } catch (e: unknown) {
    debugLog.skill.error('openSavedPreview_failed', { skill_id: row.id, error: getErrorMessage(e, '') })
    ElMessage.error(getErrorMessage(e, '加载当前版本预览失败'))
  } finally {
    savedPreviewLoading.value = false
  }
}

const openVersions = async (row: AgentSkillInfo) => {
  debugLog.skill.info('openVersions_started', { skill_id: row.id })
  versionsDrawerVisible.value = true
  versionsLoading.value = true
  versionSkill.value = row
  versions.value = []
  selectedVersion.value = null
  try {
    const detail = normalizeSkill(await api.getAgentSkill(row.id))
    const versionList = await loadVersions(row.id)
    versionSkill.value = detail
    versions.value = versionList
    selectedVersion.value = findCurrentVersion(detail, versionList) || versionList[0] || null
    debugLog.skill.info('openVersions_finished', { skill_id: row.id, version_count: versionList.length })
  } catch (e: unknown) {
    debugLog.skill.error('openVersions_failed', { skill_id: row.id, error: getErrorMessage(e, '') })
    ElMessage.error(getErrorMessage(e, '加载版本列表失败'))
  } finally {
    versionsLoading.value = false
  }
}

const selectVersion = (version: AgentSkillVersionInfo) => {
  selectedVersion.value = version
}

const activateVersion = async (version: AgentSkillVersionInfo) => {
  debugLog.skill.info('activateVersion_started', { skill_id: versionSkill.value?.id, version_id: version.id, version_label: version.version })
  if (!versionSkill.value || activatingVersionId.value) return
  try {
    await ElMessageBox.confirm(
      `确认将版本 #${version.id} ${version.version || ''} 设为当前版本？该操作会影响后续使用该 Agent Skill 的请求。`,
      '设为当前版本',
      {
        confirmButtonText: '设为当前',
        cancelButtonText: '取消',
        type: 'warning',
      },
    )
  } catch {
    debugLog.skill.info('activateVersion_cancelled', { version_id: version.id })
    return
  }
  activatingVersionId.value = version.id
  try {
    await api.activateAgentSkillVersion(versionSkill.value.id, version.id)
    ElMessage.success('已设为当前版本')
    debugLog.skill.info('activateVersion_succeeded', { skill_id: versionSkill.value.id, version_id: version.id })
    await loadList()
    if (versionSkill.value) {
      const nextSkill = { ...versionSkill.value, current_version_id: version.id }
      versionSkill.value = nextSkill
      versions.value = versions.value.map((item) => ({
        ...item,
        is_current: item.id === version.id,
      }))
      selectedVersion.value = versions.value.find((item) => item.id === version.id) || version
    }
  } catch (e: unknown) {
    debugLog.skill.error('activateVersion_failed', { version_id: version.id, error: getErrorMessage(e, '') })
    ElMessage.error(getErrorMessage(e, '设为当前版本失败'))
  } finally {
    activatingVersionId.value = null
  }
}

const formatTime = (value?: string) => {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-CN')
}

watch([
  () => form.name,
  () => form.display_name,
  () => form.description,
  () => form.agent_type,
  () => form.category,
  () => form.scenario,
  () => form.priority,
  () => form.risk_level,
  () => form.required_capabilities,
  () => form.output_schema,
  () => form.evaluation_criteria,
  () => form.semantic_tags,
  () => form.version,
  flow,
], () => {
  validation.value = buildLocalValidation()
  previewMarkdown.value = localMarkdown.value
}, { deep: true, immediate: true })

onMounted(() => {
  selectedNodeId.value = flow.value.nodes[0]?.id || ''
  loadList()
  loadCapabilities()
})
</script>

<template>
  <section class="agent-skill-page">
    <div class="workspace-surface">
      <div class="workspace-surface__header">
        <div class="workspace-surface__header-copy">
          <p class="console-eyebrow">Agent Skill</p>
          <h1 class="console-title">Agent Skill 管理</h1>
          <p class="console-description">用流程节点编排生成数据库版 SKILL.md，供 AI 助手手动选择使用。</p>
        </div>
        <div class="workspace-surface__header-actions">
          <el-button :icon="Refresh" @click="loadList">刷新列表</el-button>
          <el-button type="primary" :icon="Plus" @click="openCreate">新建 Skill</el-button>
        </div>
      </div>

      <div class="workspace-surface__divider"></div>

      <div class="workspace-surface__toolbar">
        <div class="workspace-surface__filters">
          <el-input
            v-model="keyword"
            style="width: 260px"
            placeholder="搜索名称/标识"
            clearable
            :prefix-icon="Search"
            @keyup.enter="loadList"
          />
          <el-select v-model="statusFilter" placeholder="状态" clearable style="width: 140px" @change="loadList">
            <el-option label="已启用" value="enabled" />
            <el-option label="已停用" value="disabled" />
          </el-select>
        </div>
        <div class="workspace-surface__actions">
          <el-button :icon="Search" type="primary" @click="loadList">查询</el-button>
        </div>
      </div>

      <div class="workspace-surface__body">
        <div class="agent-skill-table-wrap">
          <el-table v-loading="loading" class="agent-skill-table" :data="list" row-key="id" height="100%">
            <el-table-column prop="display_name" label="名称" min-width="180">
              <template #default="{ row }">
                <div class="skill-name">{{ row.display_name || row.name }}</div>
                <div class="skill-key">{{ row.name }}</div>
              </template>
            </el-table-column>
            <el-table-column prop="description" label="描述" min-width="240" show-overflow-tooltip />
            <el-table-column label="治理" min-width="220">
              <template #default="{ row }">
                <div class="governance-cell">
                  <div>
                    <el-tag size="small" type="info">{{ row.agent_type || 'hr_recruiting_agent' }}</el-tag>
                    <el-tag size="small" :type="riskLevelTagType(row.risk_level)">{{ riskLevelLabel(row.risk_level) }}</el-tag>
                    <el-tag size="small">P{{ row.priority ?? 0 }}</el-tag>
                  </div>
                  <div class="skill-key">{{ row.category || 'general' }}<span v-if="row.scenario"> · {{ row.scenario }}</span></div>
                  <div v-if="skillGovernanceWarnings(row).length" class="capability-warning">
                    {{ skillGovernanceWarnings(row).join('；') }}
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="当前版本" width="110">
              <template #default="{ row }">
                <el-tag v-if="row.current_version_id" size="small" type="success">#{{ row.current_version_id }}</el-tag>
                <el-tag v-else size="small" type="info">未发布</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <el-tag size="small" :type="row.is_enabled ? 'success' : 'info'">
                  {{ row.is_enabled ? '已启用' : '已停用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="更新时间" width="180">
              <template #default="{ row }">{{ formatTime(row.updated_at || row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="200" fixed="right">
              <template #default="{ row }">
                <el-button size="small" :icon="Edit" @click="openEdit(row)">编辑</el-button>
                <el-dropdown trigger="click" @command="(cmd: string) => { if (cmd === 'preview') openSavedPreview(row); if (cmd === 'versions') openVersions(row); if (cmd === 'embedding') regenerateEmbedding(row); if (cmd === 'toggle') toggleStatus(row) }">
                  <el-button size="small" :loading="regeneratingEmbeddingId === row.id">更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="preview" :icon="View">预览</el-dropdown-item>
                      <el-dropdown-item command="versions" :icon="Tickets">版本</el-dropdown-item>
                      <el-dropdown-item command="embedding" divided :disabled="!row.current_version_id || regeneratingEmbeddingId === row.id">
                        重新生成 Embedding
                      </el-dropdown-item>
                      <el-dropdown-item v-if="row.is_enabled" command="toggle" divided style="color: var(--el-color-danger)">停用</el-dropdown-item>
                      <el-dropdown-item v-else command="toggle" divided>启用</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </template>
            </el-table-column>
          </el-table>
        </div>
        <EmptyGuide v-if="!loading && list.length === 0" title="暂无 Agent Skill" description="从画布编排中创建第一个数据库版 SKILL.md。" />
      </div>

      <div class="workspace-surface__pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @current-change="loadList"
          @size-change="(size: number) => { pageSize = size; page = 1; loadList() }"
        />
      </div>
    </div>

    <el-dialog
      v-model="builderDialogVisible"
      :title="isEditing ? '编辑 Agent Skill' : '新建 Agent Skill'"
      width="min(1180px, 96vw)"
      top="4vh"
      class="agent-skill-builder-dialog"
      :close-on-click-modal="false"
      destroy-on-close
      @closed="resetBuilder"
    >
      <main class="canvas-workbench">
        <div class="builder-title-row">
          <div>
            <h2>{{ isEditing ? '编辑 Agent Skill' : '画布式流程编排' }}</h2>
            <p>
              {{ isEditing
                ? `正在编辑 ${editingSkill?.display_name || editingSkill?.name}，保存会发布并激活一个新版本。`
                : '按 canvas.v1 flow 生成 SKILL.md，本页面只编排提示规则，不执行工作流。' }}
            </p>
          </div>
          <el-button :icon="Document" :loading="previewLoading" @click="openPreviewDrawer">预览 SKILL.md</el-button>
        </div>

        <div class="meta-grid meta-grid--primary">
          <label class="form-field">
            <span class="form-label">显示名称</span>
            <el-input v-model="form.display_name" placeholder="例如 简历匹配分析" clearable />
          </label>
          <label class="form-field">
            <span class="form-label">适用 Agent</span>
            <el-select v-model="form.agent_type" placeholder="选择适用 Agent" filterable @change="handleAgentTypeChange">
              <el-option v-for="item in AGENT_TYPE_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </label>
          <label class="form-field">
            <span class="form-label">业务场景</span>
            <el-select v-model="form.scenario" placeholder="选择业务场景" filterable @change="handleScenarioChange">
              <el-option v-for="item in SCENARIO_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </label>
          <label class="form-field">
            <span class="form-label">风险等级</span>
            <el-select v-model="form.risk_level" placeholder="风险等级">
              <el-option label="低风险" value="low" />
              <el-option label="中风险" value="medium" />
              <el-option label="高风险" value="high" />
              <el-option label="关键风险" value="critical" />
            </el-select>
          </label>
        </div>
        <p v-if="selectedScenario" class="form-hint">
          已按“{{ selectedScenario.label }}”预设分类、风险等级、语义标签和评估标准，可在高级配置中调整。
        </p>
        <label class="form-field">
          <span class="form-label">业务用途描述</span>
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="2"
            placeholder="简要描述这个 Agent Skill 的招聘业务用途"
          />
        </label>
        <div class="advanced-config-toggle">
          <el-button text type="primary" @click="advancedConfigVisible = !advancedConfigVisible">
            {{ advancedConfigVisible ? '收起高级配置' : '展开高级配置' }}
          </el-button>
        </div>
        <div v-if="advancedConfigVisible" class="advanced-config-panel">
          <div class="meta-grid">
            <label class="form-field">
              <span class="form-label">唯一标识</span>
              <el-input v-model="form.name" placeholder="例如 candidate_job_match" clearable :disabled="isEditing" />
            </label>
            <label class="form-field">
              <span class="form-label">版本号</span>
              <el-input v-model="form.version" placeholder="例如 1.0.0" clearable />
            </label>
            <label class="form-field">
              <span class="form-label">治理分类</span>
              <el-select v-model="form.category" placeholder="治理分类" filterable>
                <el-option v-for="item in CATEGORY_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </label>
            <label class="form-field">
              <span class="form-label">自动选择优先级</span>
              <el-input-number v-model="form.priority" :min="-1000" :max="1000" controls-position="right" />
            </label>
          </div>
          <div class="governance-text-grid">
            <label class="form-field">
              <span class="form-label">依赖能力</span>
              <el-select
                v-model="form.required_capabilities"
                multiple
                filterable
                clearable
                collapse-tags
                collapse-tags-tooltip
                :loading="capabilitiesLoading"
                placeholder="选择依赖能力"
              >
                <el-option
                  v-for="item in capabilitySelectOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                  :disabled="item.disabled"
                />
              </el-select>
            </label>
            <label class="form-field">
              <span class="form-label">语义标签</span>
              <el-select
                v-model="form.semantic_tags"
                multiple
                filterable
                allow-create
                default-first-option
                clearable
                placeholder="选择或输入语义标签"
              >
                <el-option v-for="item in selectedScenario?.tags || []" :key="item" :label="item" :value="item" />
              </el-select>
            </label>
            <label class="form-field">
              <span class="form-label">评估标准</span>
              <el-select
                v-model="form.evaluation_criteria"
                multiple
                filterable
                allow-create
                default-first-option
                clearable
                placeholder="选择或输入评估标准"
              >
                <el-option v-for="item in selectedScenario?.criteria || []" :key="item" :label="item" :value="item" />
              </el-select>
            </label>
            <label class="form-field">
              <span class="form-label">输出模板</span>
              <el-select v-model="outputSchemaMode" placeholder="输出模板" @change="handleOutputSchemaModeChange">
                <el-option v-for="item in OUTPUT_SCHEMA_OPTIONS" :key="item.value || 'text'" :label="item.label" :value="item.value" />
              </el-select>
            </label>
          </div>
          <label v-if="outputSchemaMode === 'custom' || form.output_schema" class="form-field">
            <span class="form-label">输出 Schema JSON</span>
            <el-input
              v-model="form.output_schema"
              type="textarea"
              :rows="3"
              placeholder='例如 {"type":"object"}'
            />
          </label>
        </div>
        <label v-if="isEditing" class="form-field">
          <span class="form-label">版本变更说明</span>
          <el-input
            v-model="changeNote"
            type="textarea"
            :rows="2"
            placeholder="例如：补充候选人风险提示输出要求"
          />
        </label>
        <el-alert
          v-if="flowIntegrityWarning"
          class="flow-integrity-alert"
          type="warning"
          :closable="false"
          show-icon
          :title="flowIntegrityWarning"
        />

        <div class="canvas-frame">
          <AgentSkillCanvasEditor
            v-model="flow"
            v-model:selected-node-id="selectedNodeId"
            class="canvas-editor"
            :show-preview="false"
          />
        </div>

        <div class="builder-action-bar">
          <el-button plain @click="resetOrCloseBuilder">
            {{ isEditing ? '退出编辑' : '清空重置' }}
          </el-button>
          <el-button
            type="primary"
            :icon="CircleCheck"
            :loading="saving"
            @click="saveSkill"
          >
            {{ saveButtonText }}
          </el-button>
        </div>
      </main>
    </el-dialog>

    <el-drawer
      v-model="previewDrawerVisible"
      title="SKILL.md 只读预览"
      size="min(560px, 92vw)"
      direction="rtl"
      class="agent-skill-preview-drawer"
      destroy-on-close
    >
      <div class="preview-drawer-body" v-loading="previewLoading">
        <div class="preview-panel__head">
          <div>
            <div class="section-title">生成结果</div>
            <p>由画布 flow 生成，不提供原生 Markdown 编辑。</p>
          </div>
          <el-button :icon="Document" :loading="previewLoading" @click="refreshPreview">重新预览</el-button>
        </div>

        <div v-if="selectedNode" class="selected-node-summary">
          <span>{{ NODE_TYPE_LABEL[selectedNode.type] }}</span>
          <strong>{{ selectedNode.data.title || NODE_TYPE_LABEL[selectedNode.type] }}</strong>
        </div>

        <div class="validation-box" :class="{ 'validation-box--ok': validation.valid }">
          <el-icon><CircleCheck v-if="validation.valid" /><WarningFilled v-else /></el-icon>
          <span>{{ validation.valid ? '校验通过' : '需要补充' }}</span>
        </div>
        <ul v-if="validation.errors.length || validation.warnings.length" class="validation-list">
          <li v-for="item in validation.errors" :key="`e-${item}`" class="validation-list__error">{{ item }}</li>
          <li v-for="item in validation.warnings" :key="`w-${item}`">{{ item }}</li>
        </ul>

        <pre class="markdown-preview">{{ previewMarkdown }}</pre>
      </div>
    </el-drawer>

    <el-drawer
      v-model="savedPreviewDrawerVisible"
      title="当前版本预览"
      size="min(640px, 94vw)"
      direction="rtl"
      destroy-on-close
    >
      <div class="preview-drawer-body" v-loading="savedPreviewLoading">
        <div class="preview-panel__head">
          <div>
            <div class="section-title">{{ savedPreviewSkill?.display_name || savedPreviewSkill?.name || 'Agent Skill' }}</div>
            <p>
              当前版本：
              <span v-if="savedPreviewVersion">#{{ savedPreviewVersion.id }} {{ savedPreviewVersion.version || '' }}</span>
              <span v-else>未发布</span>
            </p>
          </div>
        </div>

        <EmptyGuide
          v-if="!savedPreviewLoading && !savedPreviewVersion"
          title="暂无当前版本"
          description="该 Agent Skill 尚未激活版本，无法展示 SKILL.md。"
        />
        <pre v-else class="markdown-preview">{{ savedPreviewVersion?.skill_md || '当前版本暂无 SKILL.md 内容。' }}</pre>
      </div>
    </el-drawer>

    <el-drawer
      v-model="versionsDrawerVisible"
      title="版本管理"
      size="min(1180px, 96vw)"
      direction="rtl"
      class="agent-skill-versions-drawer"
      destroy-on-close
    >
      <div class="version-drawer-body" v-loading="versionsLoading">
        <div class="version-hero">
          <div class="version-hero__main">
            <div class="version-hero__title-row">
              <div class="version-hero__title">{{ versionSkill?.display_name || versionSkill?.name || 'Agent Skill' }}</div>
              <el-tag v-if="versionSkill?.current_version_id" type="success" size="small">
                当前 #{{ versionSkill.current_version_id }}
              </el-tag>
            </div>
            <p>查看历史版本、预览 SKILL.md，并可将历史版本重新设为当前版本。</p>
            <div class="version-meta-line">
              <span>Skill Key：{{ versionSkill?.name || '-' }}</span>
              <span>当前版本 #{{ versionSkill?.current_version_id || '-' }}</span>
              <span>版本号：{{ currentVersion?.version || '-' }}</span>
            </div>
          </div>
        </div>

        <EmptyGuide
          v-if="!versionsLoading && versions.length === 0"
          title="暂无版本"
          description="保存编辑后会生成可设为当前的 Agent Skill 版本。"
        />

        <div v-else class="version-layout">
          <aside class="version-history" aria-label="版本历史">
            <div class="version-history__head">
              <span>版本历史</span>
              <el-tag size="small" type="info">{{ versions.length }} 个版本</el-tag>
            </div>
            <div class="version-history__list">
              <button
                v-for="item in versions"
                :key="item.id"
                type="button"
                class="version-history-item"
                :class="{ 'version-history-item--active': selectedVersion?.id === item.id }"
                @click="selectVersion(item)"
              >
                <div class="version-history-item__top">
                  <span class="version-history-item__id">#{{ item.id }}</span>
                  <span class="version-history-item__version">{{ item.version || '-' }}</span>
                  <el-tag v-if="item.id === versionSkill?.current_version_id || item.is_current" size="small" type="success">当前</el-tag>
                </div>
                <div class="version-history-item__note">{{ item.change_note || '暂无变更说明' }}</div>
                <div class="version-history-item__time">{{ formatTime(item.created_at) }}</div>
              </button>
            </div>
          </aside>

          <section class="version-detail">
            <template v-if="selectedVersion">
              <div class="version-detail__head">
                <div class="version-detail__title-block">
                  <div class="version-title">
                    <span>#{{ selectedVersion.id }}</span>
                    <strong>{{ selectedVersion.version || '-' }}</strong>
                    <el-tag
                      v-if="selectedVersion.id === versionSkill?.current_version_id || selectedVersion.is_current"
                      size="small"
                      type="success"
                    >
                      当前
                    </el-tag>
                  </div>
                  <div class="version-detail__meta">
                    <span>创建时间：{{ formatTime(selectedVersion.created_at) }}</span>
                    <span>状态：{{ selectedVersion.id === versionSkill?.current_version_id || selectedVersion.is_current ? '当前版本' : '历史版本' }}</span>
                  </div>
                </div>
                <el-button
                  v-if="selectedVersion.id !== versionSkill?.current_version_id && !selectedVersion.is_current"
                  type="primary"
                  :loading="activatingVersionId === selectedVersion.id"
                  @click="activateVersion(selectedVersion)"
                >
                  设为当前
                </el-button>
              </div>

              <div class="version-detail__note">
                <span>变更说明</span>
                <p>{{ selectedVersion.change_note || '暂无变更说明' }}</p>
              </div>

              <div class="version-doc">
                <div class="version-doc__head">
                  <div class="section-title">SKILL.md 预览</div>
                </div>
                <pre class="version-doc__content">{{ selectedVersion.skill_md || '当前版本暂无 SKILL.md 内容。' }}</pre>
              </div>
            </template>

            <div v-else class="version-empty-detail">
              <div class="section-title">版本详情</div>
              <p>请选择左侧版本查看详情和 SKILL.md 内容。</p>
            </div>
          </section>
        </div>
      </div>
    </el-drawer>
  </section>
</template>

<style scoped>
.agent-skill-page {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 18px;
  overflow-x: hidden;
  overflow-y: auto;
  padding-bottom: 24px;
}

.list-shell {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: 14px;
  min-height: 0;
}

:global(.agent-skill-builder-dialog) {
  max-height: 92vh;
  display: flex;
  flex-direction: column;
}

:global(.agent-skill-builder-dialog .el-dialog__body) {
  flex: 1 1 auto;
  min-height: 0;
  max-height: calc(92vh - 72px);
  overflow: auto;
  padding-top: 8px;
}

.canvas-workbench {
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  box-shadow: var(--admin-console-card-shadow);
}

.canvas-workbench {
  min-width: 0;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.section-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--text-primary);
}

.skill-name {
  display: block;
  font-weight: 700;
}

.skill-key,
.preview-drawer-body p,
.builder-title-row p {
  margin: 4px 0 0;
  color: var(--text-faint);
  font-size: 12px;
  line-height: 1.5;
}

.builder-title-row,
.preview-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.builder-title-row h2 {
  margin: 0;
  font-size: 18px;
}

.meta-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.meta-grid--primary {
  grid-template-columns: 1.2fr 1fr 1fr 150px;
}

.form-field {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 700;
  line-height: 1.3;
}

.form-field :deep(.el-select),
.form-field :deep(.el-input-number) {
  width: 100%;
}

.form-hint {
  margin: -2px 0 2px;
  color: var(--text-faint);
  font-size: 12px;
}

.advanced-config-toggle {
  display: flex;
  justify-content: flex-start;
}

.advanced-config-panel {
  display: grid;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  background: color-mix(in srgb, var(--surface-muted) 62%, transparent);
}

.governance-grid {
  display: grid;
  grid-template-columns: 1.3fr 1fr 1fr 150px 150px;
  gap: 10px;
}

.governance-text-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.governance-cell {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}

.governance-cell > div:first-child {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.capability-warning {
  color: var(--el-color-danger);
  font-size: 12px;
  line-height: 1.4;
}

.canvas-frame {
  min-height: 560px;
  height: clamp(560px, 52vh, 720px);
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-muted);
}

.canvas-editor {
  width: 100%;
  height: 100%;
  min-height: 560px;
}

.builder-action-bar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  padding-top: 4px;
}

.preview-drawer-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 100%;
}

.selected-node-summary {
  display: grid;
  gap: 4px;
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 10px;
  background: var(--surface-muted);
}

.selected-node-summary span {
  color: var(--text-faint);
  font-size: 12px;
  font-weight: 700;
}

.selected-node-summary strong {
  overflow: hidden;
  color: var(--text-primary);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.validation-box {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px;
  border-radius: 8px;
  background: color-mix(in srgb, #d97706 12%, var(--surface));
  color: #b45309;
  font-weight: 700;
}

.validation-box--ok {
  background: color-mix(in srgb, #16a34a 12%, var(--surface));
  color: #15803d;
}

.validation-list {
  margin: 0;
  padding-left: 18px;
  color: var(--text-secondary);
  font-size: 13px;
}

.validation-list__error {
  color: #dc2626;
}

.markdown-preview {
  min-height: 420px;
  max-height: calc(100vh - 300px);
  margin: 0;
  padding: 14px;
  overflow: auto;
  border-radius: 8px;
  background: var(--surface-muted);
  color: var(--text-primary);
  white-space: pre-wrap;
  line-height: 1.55;
  font-size: 12px;
}

.pagination-wrap {
  display: flex;
  flex: 0 0 auto;
  justify-content: flex-end;
  padding-top: 14px;
  padding-bottom: 14px;
}

.list-table-card {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.list-table-card :deep(.admin-table-card__header) {
  flex: 0 0 auto;
}

.list-table-card :deep(.admin-table-card__body) {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
  padding: 0;
}

.agent-skill-table-wrap {
  flex: 1 1 auto;
  min-height: 0;
}

.agent-skill-table {
  height: 100%;
}

:global(.agent-skill-versions-drawer .el-drawer__body) {
  min-height: 0;
  overflow: hidden;
}

.version-drawer-body {
  display: flex;
  height: calc(100vh - 96px);
  max-height: 86vh;
  min-height: 0;
  flex-direction: column;
  gap: 16px;
  overflow: hidden;
}

.version-hero {
  flex: 0 0 auto;
  padding: 2px 0 0;
}

.version-hero__main {
  min-width: 0;
}

.version-hero__title-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.version-hero__title {
  color: var(--text-primary);
  font-size: 18px;
  font-weight: 700;
}

.version-hero p {
  margin: 6px 0 0;
  color: var(--text-faint);
  font-size: 13px;
  line-height: 1.6;
}

.version-meta-line {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 16px;
  margin-top: 10px;
  color: var(--text-secondary);
  font-size: 12px;
}

.version-layout {
  display: grid;
  flex: 1 1 auto;
  grid-template-columns: clamp(320px, 31vw, 360px) minmax(0, 1fr);
  gap: 16px;
  min-height: 0;
  overflow: hidden;
}

.version-history,
.version-detail {
  min-height: 0;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
}

.version-history {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.version-history__head {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--border);
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 700;
}

.version-history__list {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
  gap: 8px;
  overflow: auto;
  padding: 10px;
}

.version-history-item {
  appearance: none;
  width: 100%;
  display: grid;
  gap: 7px;
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 11px 12px;
  background: var(--surface-muted);
  color: var(--text-secondary);
  cursor: pointer;
  font: inherit;
  text-align: left;
  transition: border-color 0.18s ease, background-color 0.18s ease, box-shadow 0.18s ease;
}

.version-history-item:hover,
.version-history-item--active {
  border-color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 8%, var(--surface));
}

.version-history-item--active {
  box-shadow: inset 3px 0 0 var(--el-color-primary);
}

.version-history-item__top {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.version-history-item__id {
  color: var(--text-primary);
  font-weight: 700;
}

.version-history-item__version {
  overflow: hidden;
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.version-history-item__note {
  display: -webkit-box;
  overflow: hidden;
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.5;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.version-history-item__time {
  color: var(--text-faint);
  font-size: 12px;
}

.version-detail {
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.version-detail__head {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--border);
}

.version-detail__title-block {
  display: grid;
  gap: 6px;
  min-width: 0;
}

.version-title {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  color: var(--text-primary);
  font-size: 15px;
  font-weight: 700;
}

.version-detail__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 14px;
  color: var(--text-faint);
  font-size: 12px;
}

.version-detail__note {
  flex: 0 0 auto;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
  background: color-mix(in srgb, var(--surface-muted) 72%, transparent);
}

.version-detail__note span {
  color: var(--text-faint);
  font-size: 12px;
  font-weight: 700;
}

.version-detail__note p {
  margin: 5px 0 0;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.version-doc {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
}

.version-doc__head {
  flex: 0 0 auto;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
}

.version-doc__content {
  flex: 1 1 auto;
  min-height: 0;
  margin: 0;
  padding: 22px 24px;
  overflow: auto;
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--surface-muted) 92%, transparent), var(--surface-muted));
  color: var(--text-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-size: 13px;
  line-height: 1.72;
  white-space: pre-wrap;
  word-break: break-word;
}

.version-empty-detail {
  display: grid;
  flex: 1 1 auto;
  place-content: center;
  gap: 6px;
  padding: 24px;
  color: var(--text-faint);
  text-align: center;
}

.version-empty-detail p {
  margin: 0;
  font-size: 13px;
}

.filter-input {
  width: 260px;
}

.filter-select {
  width: 140px;
}

@media (max-width: 860px) {
  .meta-grid,
  .meta-grid--primary,
  .governance-grid,
  .governance-text-grid,
  .version-layout {
    grid-template-columns: 1fr;
  }


  .version-drawer-body {
    height: 84vh;
    max-height: 84vh;
  }

  .version-history {
    min-height: 240px;
  }

  .canvas-frame {
    height: 560px;
  }

  .builder-action-bar {
    flex-wrap: wrap;
  }
}
</style>
