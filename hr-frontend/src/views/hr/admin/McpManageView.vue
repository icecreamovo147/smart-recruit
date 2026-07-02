<script setup lang="ts">
import { onMounted, reactive, ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Delete,
  Edit,
  Plus,
  Refresh,
  Connection,
  MoreFilled,
  Search,
  Monitor,
  Document,
  Sort,
  View,
  ArrowDown,
  Lock,
} from '@element-plus/icons-vue'
import {
  listMcpServers,
  createMcpServer,
  updateMcpServer,
  deleteMcpServer,
  testMcpServerConnection,
  listMcpServerTools,
  listMcpServerLogs,
  listMcpToolPolicies,
  createMcpToolPolicy,
  updateMcpToolPolicy,
  deleteMcpToolPolicy,
} from '@/api/mcp'
import type {
  McpServerInfo,
  McpToolInfo,
  McpToolPolicy,
  McpCallLog,
  CreateMcpServerPayload,
  UpdateMcpServerPayload,
  CreateMcpToolPolicyPayload,
  McpPolicyEffect,
  McpPolicyRiskLevel,
  McpTransportType,
} from '@/types/mcp'

// ====== Transport type options ======

const TRANSPORT_OPTIONS = [
  { value: 'stdio', label: 'STDIO' },
  { value: 'sse', label: 'SSE' },
  { value: 'http', label: 'HTTP' },
]

const TRANSPORT_LABEL: Record<string, string> = {
  stdio: 'STDIO',
  sse: 'SSE',
  http: 'HTTP',
}

const STATUS_LABEL: Record<string, string> = {
  connected: '已连接',
  disconnected: '未连接',
  error: '错误',
}

const STATUS_TYPE: Record<string, string> = {
  connected: 'success',
  disconnected: 'info',
  error: 'danger',
}

const POLICY_EFFECT_LABEL: Record<string, string> = {
  allow: '允许',
  deny: '拒绝',
}

const POLICY_RISK_LABEL: Record<string, string> = {
  low: '低',
  medium: '中',
  high: '高',
  critical: '严重',
}

const POLICY_DECISION_LABEL: Record<string, string> = {
  allow: '允许',
  deny: '拒绝',
  confirmation_required: '需确认',
  rate_limited: '限流',
  invalid_args: '参数不合规',
  no_policy: '无策略',
}

// ====== View state ======

type ActiveView = 'list' | 'tools' | 'logs' | 'policies'

const activeView = ref<ActiveView>('list')
const selectedServer = ref<McpServerInfo | null>(null)

const viewTitle = computed(() => {
  if (activeView.value === 'tools' && selectedServer.value) {
    return `工具列表 - ${selectedServer.value.name}`
  }
  if (activeView.value === 'logs' && selectedServer.value) {
    return `调用日志 - ${selectedServer.value.name}`
  }
  if (activeView.value === 'policies') {
    return selectedServer.value ? `工具策略 - ${selectedServer.value.name}` : 'MCP 工具策略'
  }
  return 'MCP Server 管理'
})

const navigateToTools = (row: McpServerInfo) => {
  selectedServer.value = row
  activeView.value = 'tools'
  loadTools()
}

const navigateToLogs = (row: McpServerInfo) => {
  selectedServer.value = row
  activeView.value = 'logs'
  logPage.value = 1
  loadLogs()
}

const navigateToPolicies = (row?: McpServerInfo) => {
  selectedServer.value = row || null
  activeView.value = 'policies'
  policyPage.value = 1
  loadPolicies()
}

const goBackToList = () => {
  activeView.value = 'list'
  selectedServer.value = null
}

// ====== Server List ======

const list = ref<McpServerInfo[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const error = ref('')
const keywordFilter = ref('')
const statusFilter = ref('')
const transportFilter = ref('')

const loadList = async () => {
  loading.value = true
  error.value = ''
  try {
    const data = await listMcpServers(page.value, pageSize.value)
    list.value = data.list || []
    total.value = data.total || 0
  } catch (e: unknown) {
    error.value = (e as { message?: string }).message || '加载 MCP Server 列表失败'
  } finally {
    loading.value = false
  }
}

const filteredServers = computed(() => {
  const keyword = keywordFilter.value.trim().toLowerCase()
  return list.value.filter((item) => {
    const matchesKeyword = !keyword
      || item.name.toLowerCase().includes(keyword)
      || (item.description || '').toLowerCase().includes(keyword)
      || (item.command || '').toLowerCase().includes(keyword)
      || (item.url || '').toLowerCase().includes(keyword)
    const matchesStatus = !statusFilter.value || item.status === statusFilter.value
    const matchesTransport = !transportFilter.value || item.transport_type === transportFilter.value
    return matchesKeyword && matchesStatus && matchesTransport
  })
})

// ====== Edit / Create Dialog ======

const dialogVisible = ref(false)
const dialogTitle = ref('')
const saving = ref(false)
const isEditing = ref(false)
const editingId = ref(0)
const dialogForm = reactive({
  name: '',
  description: '',
  transport_type: 'stdio' as McpTransportType,
  command: '',
  url: '',
  args: '',
  env: '',
  timeout_seconds: 30,
  is_enabled: true,
})

const showCommandFields = computed(() => dialogForm.transport_type === 'stdio')
const showUrlFields = computed(() => dialogForm.transport_type === 'sse' || dialogForm.transport_type === 'http')

const resetDialogForm = () => {
  dialogForm.name = ''
  dialogForm.description = ''
  dialogForm.transport_type = 'stdio'
  dialogForm.command = ''
  dialogForm.url = ''
  dialogForm.args = ''
  dialogForm.env = ''
  dialogForm.timeout_seconds = 30
  dialogForm.is_enabled = true
}

const openCreate = () => {
  isEditing.value = false
  editingId.value = 0
  dialogTitle.value = '新增 MCP Server'
  resetDialogForm()
  dialogVisible.value = true
}

const openEdit = (row: McpServerInfo) => {
  isEditing.value = true
  editingId.value = row.id
  dialogTitle.value = '编辑 MCP Server'
  dialogForm.name = row.name
  dialogForm.description = row.description || ''
  dialogForm.transport_type = row.transport_type
  dialogForm.command = row.command || ''
  dialogForm.url = row.url || ''
  dialogForm.args = (row.args || []).join('\n')
  dialogForm.env = Object.entries(row.env || {})
    .map(([k, v]) => `${k}=${v}`)
    .join('\n')
  dialogForm.timeout_seconds = row.timeout_seconds || 30
  dialogForm.is_enabled = row.is_enabled
  dialogVisible.value = true
}

const save = async () => {
  if (!dialogForm.name) {
    ElMessage.warning('请输入 Server 名称')
    return
  }
  if (showCommandFields.value && !dialogForm.command) {
    ElMessage.warning('请输入命令')
    return
  }
  if (showUrlFields.value && !dialogForm.url) {
    ElMessage.warning('请输入 URL')
    return
  }
  if (dialogForm.timeout_seconds < 1 || dialogForm.timeout_seconds > 300) {
    ElMessage.warning('超时时间请在 1-300 秒之间')
    return
  }

  saving.value = true
  try {
    // Parse args and env from textarea
    const args = dialogForm.args
      ? dialogForm.args.split('\n').map((s) => s.trim()).filter(Boolean)
      : undefined
    const envLines = dialogForm.env
      ? dialogForm.env.split('\n').map((s) => s.trim()).filter(Boolean)
      : []
    const env: Record<string, string> = {}
    for (const line of envLines) {
      const eqIdx = line.indexOf('=')
      if (eqIdx > 0) {
        env[line.slice(0, eqIdx)] = line.slice(eqIdx + 1)
      }
    }

    if (isEditing.value) {
      const payload: UpdateMcpServerPayload = {
        name: dialogForm.name,
        transport_type: dialogForm.transport_type,
        timeout_seconds: dialogForm.timeout_seconds,
        is_enabled: dialogForm.is_enabled,
      }
      if (dialogForm.description) payload.description = dialogForm.description
      if (showCommandFields.value) {
        payload.command = dialogForm.command
        payload.args = args
        payload.env = Object.keys(env).length > 0 ? env : undefined
      } else {
        payload.url = dialogForm.url
      }
      await updateMcpServer(editingId.value, payload)
      ElMessage.success('MCP Server 已更新')
    } else {
      const payload: CreateMcpServerPayload = {
        name: dialogForm.name,
        transport_type: dialogForm.transport_type,
        timeout_seconds: dialogForm.timeout_seconds,
      }
      if (dialogForm.description) payload.description = dialogForm.description
      if (showCommandFields.value) {
        payload.command = dialogForm.command
        payload.args = args
        payload.env = Object.keys(env).length > 0 ? env : undefined
      } else {
        payload.url = dialogForm.url
      }
      await createMcpServer(payload)
      ElMessage.success('MCP Server 已创建')
    }
    dialogVisible.value = false
    await loadList()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '保存失败')
  } finally {
    saving.value = false
  }
}

// ====== Connection Test ======

const testingId = ref<number | null>(null)

const handleTestConnection = async (row: McpServerInfo) => {
  testingId.value = row.id
  try {
    const result = await testMcpServerConnection(row.id)
    if (result.success) {
      ElMessage.success(`连接测试成功（发现 ${result.tools_found} 个工具，耗时 ${result.duration_ms}ms）`)
    } else {
      ElMessage.error(`连接测试失败：${result.message}`)
    }
    await loadList()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '连接测试失败')
  } finally {
    testingId.value = null
  }
}

// ====== Toggle enabled ======

const handleToggleEnabled = async (row: McpServerInfo) => {
  try {
    await updateMcpServer(row.id, { is_enabled: !row.is_enabled })
    ElMessage.success(row.is_enabled ? '已禁用' : '已启用')
    await loadList()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '操作失败')
  }
}

// ====== Delete ======

const handleDelete = async (row: McpServerInfo) => {
  try {
    await ElMessageBox.confirm(
      `确认删除 MCP Server「${row.name}」？此操作不可撤销。`,
      '删除确认',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return
  }
  try {
    await deleteMcpServer(row.id)
    ElMessage.success('MCP Server 已删除')
    await loadList()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '删除失败')
  }
}

// ====== Tool List (sub-view) ======

const tools = ref<McpToolInfo[]>([])
const toolsLoading = ref(false)
const toolsError = ref('')

const loadTools = async () => {
  if (!selectedServer.value) return
  toolsLoading.value = true
  toolsError.value = ''
  try {
    const data = await listMcpServerTools(selectedServer.value.id)
    tools.value = data.tools || []
  } catch (e: unknown) {
    toolsError.value = (e as { message?: string }).message || '加载工具列表失败'
  } finally {
    toolsLoading.value = false
  }
}

const expandedToolRows = ref<string[]>([])

const formatSchema = (schema: Record<string, unknown>): string => {
  try {
    return JSON.stringify(schema, null, 2)
  } catch {
    return String(schema)
  }
}

// ====== Tool Policies (sub-view) ======

const policies = ref<McpToolPolicy[]>([])
const policyTotal = ref(0)
const policyPage = ref(1)
const policyPageSize = ref(20)
const policiesLoading = ref(false)
const policiesError = ref('')
const policyFilterServerId = ref<number | null>(null)
const policyFilterKeyword = ref('')
const policyDialogVisible = ref(false)
const policySaving = ref(false)
const policyEditingId = ref(0)
const policyIsEditing = ref(false)
const policyDialogTitle = ref('新增工具策略')

const policyForm = reactive({
  server_id: 0,
  tool_name: '',
  effect: 'allow' as McpPolicyEffect,
  risk_level: 'low' as McpPolicyRiskLevel,
  require_confirmation: false,
  allowed_roles: '',
  allowed_scopes: '',
  required_args: '',
  denied_args: '',
  arg_rules: '{}',
  redact_fields: '',
  rate_limit_window_seconds: 0,
  rate_limit_max_calls: 0,
  is_enabled: true,
})

const policyServerOptions = computed(() => list.value.map((server) => ({
  label: server.name,
  value: server.id,
})))

const filteredPolicies = computed(() => {
  const keyword = policyFilterKeyword.value.trim().toLowerCase()
  return policies.value.filter((policy) => {
    const matchesKeyword = !keyword
      || policy.tool_name.toLowerCase().includes(keyword)
      || (policy.server_name || '').toLowerCase().includes(keyword)
      || String(policy.server_id).includes(keyword)
    const matchesServer = !policyFilterServerId.value || policy.server_id === policyFilterServerId.value
    return matchesKeyword && matchesServer
  })
})

const splitLines = (value: string): string[] =>
  value.split('\n').map((item) => item.trim()).filter(Boolean)

const joinLines = (value: string[]): string =>
  (value || []).join('\n')

const parsePolicyRules = (): Record<string, unknown> | null => {
  if (!policyForm.arg_rules.trim()) return {}
  try {
    const parsed: unknown = JSON.parse(policyForm.arg_rules)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>
    }
  } catch { /* handled below */ }
  ElMessage.warning('参数规则必须是 JSON 对象')
  return null
}

const policyPayload = (): CreateMcpToolPolicyPayload | null => {
  if (!policyForm.server_id) {
    ElMessage.warning('请选择 MCP Server')
    return null
  }
  if (!policyForm.tool_name.trim()) {
    ElMessage.warning('请输入工具名称')
    return null
  }
  const argRules = parsePolicyRules()
  if (!argRules) return null
  return {
    server_id: policyForm.server_id,
    tool_name: policyForm.tool_name.trim(),
    effect: policyForm.effect,
    risk_level: policyForm.risk_level,
    require_confirmation: policyForm.require_confirmation,
    allowed_roles: splitLines(policyForm.allowed_roles),
    allowed_scopes: splitLines(policyForm.allowed_scopes),
    required_args: splitLines(policyForm.required_args),
    denied_args: splitLines(policyForm.denied_args),
    arg_rules: argRules,
    redact_fields: splitLines(policyForm.redact_fields),
    rate_limit_window_seconds: policyForm.rate_limit_window_seconds,
    rate_limit_max_calls: policyForm.rate_limit_max_calls,
    is_enabled: policyForm.is_enabled,
  }
}

const loadPolicies = async () => {
  policiesLoading.value = true
  policiesError.value = ''
  try {
    const serverId = selectedServer.value?.id || policyFilterServerId.value || undefined
    const data = await listMcpToolPolicies({
      page: policyPage.value,
      page_size: policyPageSize.value,
      server_id: serverId,
    })
    policies.value = data.list || []
    policyTotal.value = data.total || 0
  } catch (e: unknown) {
    policiesError.value = (e as { message?: string }).message || '加载工具策略失败'
  } finally {
    policiesLoading.value = false
  }
}

const resetPolicyForm = (server?: McpServerInfo) => {
  policyForm.server_id = server?.id || selectedServer.value?.id || 0
  policyForm.tool_name = ''
  policyForm.effect = 'allow'
  policyForm.risk_level = 'low'
  policyForm.require_confirmation = false
  policyForm.allowed_roles = ''
  policyForm.allowed_scopes = ''
  policyForm.required_args = ''
  policyForm.denied_args = ''
  policyForm.arg_rules = '{}'
  policyForm.redact_fields = ''
  policyForm.rate_limit_window_seconds = 0
  policyForm.rate_limit_max_calls = 0
  policyForm.is_enabled = true
}

const openCreatePolicy = (server?: McpServerInfo, toolName = '') => {
  policyIsEditing.value = false
  policyEditingId.value = 0
  policyDialogTitle.value = '新增工具策略'
  resetPolicyForm(server)
  policyForm.tool_name = toolName
  policyDialogVisible.value = true
}

const openEditPolicy = (policy: McpToolPolicy) => {
  policyIsEditing.value = true
  policyEditingId.value = policy.id
  policyDialogTitle.value = '编辑工具策略'
  policyForm.server_id = policy.server_id
  policyForm.tool_name = policy.tool_name
  policyForm.effect = policy.effect
  policyForm.risk_level = policy.risk_level
  policyForm.require_confirmation = policy.require_confirmation
  policyForm.allowed_roles = joinLines(policy.allowed_roles)
  policyForm.allowed_scopes = joinLines(policy.allowed_scopes)
  policyForm.required_args = joinLines(policy.required_args)
  policyForm.denied_args = joinLines(policy.denied_args)
  policyForm.arg_rules = JSON.stringify(policy.arg_rules || {}, null, 2)
  policyForm.redact_fields = joinLines(policy.redact_fields)
  policyForm.rate_limit_window_seconds = policy.rate_limit_window_seconds || 0
  policyForm.rate_limit_max_calls = policy.rate_limit_max_calls || 0
  policyForm.is_enabled = policy.is_enabled
  policyDialogVisible.value = true
}

const savePolicy = async () => {
  const payload = policyPayload()
  if (!payload) return
  policySaving.value = true
  try {
    if (policyIsEditing.value) {
      await updateMcpToolPolicy(policyEditingId.value, payload)
      ElMessage.success('工具策略已更新')
    } else {
      await createMcpToolPolicy(payload)
      ElMessage.success('工具策略已创建')
    }
    policyDialogVisible.value = false
    await loadPolicies()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '保存工具策略失败')
  } finally {
    policySaving.value = false
  }
}

const togglePolicyEnabled = async (policy: McpToolPolicy) => {
  try {
    await updateMcpToolPolicy(policy.id, { is_enabled: !policy.is_enabled })
    ElMessage.success(policy.is_enabled ? '策略已禁用' : '策略已启用')
    await loadPolicies()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '切换策略失败')
  }
}

const deletePolicy = async (policy: McpToolPolicy) => {
  try {
    await ElMessageBox.confirm(
      `确认删除「${policy.tool_name}」的 MCP 工具策略？`,
      '删除确认',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return
  }
  try {
    await deleteMcpToolPolicy(policy.id)
    ElMessage.success('工具策略已删除')
    await loadPolicies()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '删除工具策略失败')
  }
}

const policyServerName = (policy: McpToolPolicy): string =>
  policy.server_name || list.value.find((server) => server.id === policy.server_id)?.name || `#${policy.server_id}`

const policyDecisionTagType = (decision: string): 'success' | 'warning' | 'danger' | 'info' | 'primary' => {
  if (decision === 'allow') return 'success'
  if (decision === 'deny' || decision === 'invalid_args') return 'danger'
  if (decision === 'confirmation_required' || decision === 'rate_limited') return 'warning'
  return 'info'
}

// ====== Call Logs (sub-view) ======

const logs = ref<McpCallLog[]>([])
const logsTotal = ref(0)
const logPage = ref(1)
const logPageSize = ref(20)
const logsLoading = ref(false)
const logsError = ref('')
const logFilterToolName = ref('')
const logFilterStartTime = ref('')
const logFilterEndTime = ref('')

// Log detail dialog
const logDetailVisible = ref(false)
const logDetail = ref<McpCallLog | null>(null)

const loadLogs = async () => {
  if (!selectedServer.value) return
  logsLoading.value = true
  logsError.value = ''
  try {
    const data = await listMcpServerLogs(selectedServer.value.id, {
      page: logPage.value,
      page_size: logPageSize.value,
      tool_name: logFilterToolName.value || undefined,
      start_time: logFilterStartTime.value || undefined,
      end_time: logFilterEndTime.value || undefined,
    })
    logs.value = data.list || []
    logsTotal.value = data.total || 0
  } catch (e: unknown) {
    logsError.value = (e as { message?: string }).message || '加载调用日志失败'
  } finally {
    logsLoading.value = false
  }
}

const openLogDetail = (row: McpCallLog) => {
  logDetail.value = row
  logDetailVisible.value = true
}

const formatLogParam = (jsonStr: string, maxLen = 200): string => {
  if (!jsonStr) return '-'
  try {
    const parsed = JSON.parse(jsonStr)
    const formatted = JSON.stringify(parsed, null, 2)
    if (formatted.length > maxLen) {
      return formatted.slice(0, maxLen) + '...'
    }
    return formatted
  } catch {
    return jsonStr.length > maxLen ? jsonStr.slice(0, maxLen) + '...' : jsonStr
  }
}

const formatLogParamFull = (jsonStr: string): string => {
  if (!jsonStr) return '-'
  try {
    const parsed = JSON.parse(jsonStr)
    return JSON.stringify(parsed, null, 2)
  } catch {
    return jsonStr
  }
}

// Format duration for display
const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`
}

// ====== Helpers ======

const formatTime = (s?: string): string => {
  if (!s) return '-'
  return new Date(s).toLocaleString('zh-CN')
}

// ====== Init ======

onMounted(() => {
  loadList()
})
</script>

<template>
  <div class="console-page console-page--fill mcp-manage-view">
    <div class="workspace-surface">
      <div class="workspace-surface__header">
        <div class="workspace-surface__header-copy">
          <p class="console-eyebrow">TOOL CENTER</p>
          <h2 class="console-title">{{ viewTitle }}</h2>
          <p class="console-description">接入和管理 MCP Server，统一查看工具能力、连接状态和调用审计，作为 Agent 工具层的控制台。</p>
        </div>
        <div class="workspace-surface__header-actions">
          <el-button
            v-if="activeView !== 'list'"
            :icon="Sort"
            @click="goBackToList"
          >
            返回列表
          </el-button>
          <el-button v-if="activeView === 'list'" :icon="Refresh" @click="loadList">刷新</el-button>
          <el-button v-if="activeView === 'list'" :icon="Lock" @click="navigateToPolicies()">策略</el-button>
          <el-button v-if="activeView === 'list'" type="primary" :icon="Plus" @click="openCreate">新增 Server</el-button>
          <el-button v-if="activeView === 'policies'" :icon="Refresh" @click="loadPolicies">刷新</el-button>
          <el-button v-if="activeView === 'policies'" type="primary" :icon="Plus" @click="openCreatePolicy(selectedServer || undefined)">新增策略</el-button>
        </div>
      </div>

      <!-- ════════════════════════════════════════════════════════════════════════
           View 1: MCP Server List (Main)
           ════════════════════════════════════════════════════════════════════════ -->
      <template v-if="activeView === 'list'">
        <div class="workspace-surface__divider"></div>
        <div class="workspace-surface__toolbar">
          <div class="workspace-surface__filters">
            <el-input v-model="keywordFilter" :prefix-icon="Search" clearable placeholder="搜索名称 / 命令 / URL" style="width: 260px" />
            <el-select v-model="transportFilter" clearable placeholder="全部传输" style="width: 140px">
              <el-option v-for="opt in TRANSPORT_OPTIONS" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>
            <el-select v-model="statusFilter" clearable placeholder="全部状态" style="width: 140px">
              <el-option label="已连接" value="connected" />
              <el-option label="未连接" value="disconnected" />
              <el-option label="错误" value="error" />
            </el-select>
          </div>
        </div>
      <div class="workspace-surface__body">
      <el-table
        v-loading="loading"
        :data="filteredServers"
        class="console-table"
        stripe
        style="width: 100%"
        :empty-text="error || '暂无数据'"
      >
        <el-table-column label="Server 信息" min-width="240">
          <template #default="{ row }: { row: McpServerInfo }">
            <div class="console-entity">
              <div class="console-entity__name">{{ row.name }}</div>
              <div class="console-entity__meta">{{ row.description || '暂无描述' }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="传输类型" width="100">
          <template #default="{ row }: { row: McpServerInfo }">
            <el-tag size="small">{{ TRANSPORT_LABEL[row.transport_type] || row.transport_type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="command" label="命令/URL" min-width="200" show-overflow-tooltip>
          <template #default="{ row }: { row: McpServerInfo }">
            {{ row.command || row.url || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }: { row: McpServerInfo }">
            <el-tag :type="(STATUS_TYPE[row.status] as 'success' | 'info' | 'danger') || 'info'" size="small">
              {{ STATUS_LABEL[row.status] || row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="tool_count" label="工具数" width="80" />
        <el-table-column prop="timeout_seconds" label="超时(s)" width="80" />
        <el-table-column label="启用" width="80">
          <template #default="{ row }: { row: McpServerInfo }">
            <el-switch
              :model-value="row.is_enabled"
              size="small"
              @click="handleToggleEnabled(row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="170">
          <template #default="{ row }: { row: McpServerInfo }">
            {{ formatTime(row.updated_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }: { row: McpServerInfo }">
            <el-button size="small" :icon="Connection" @click="handleTestConnection(row)" :loading="testingId === row.id">
              测试
            </el-button>
            <el-button size="small" :icon="Edit" @click="openEdit(row)">
              编辑
            </el-button>
            <el-dropdown trigger="click" @command="(cmd: string) => { if (cmd === 'tools') navigateToTools(row); if (cmd === 'logs') navigateToLogs(row); if (cmd === 'policies') navigateToPolicies(row); if (cmd === 'toggle') handleToggleEnabled(row); if (cmd === 'delete') handleDelete(row) }">
              <el-button size="small">更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="tools" :icon="Document">工具</el-dropdown-item>
                  <el-dropdown-item command="policies" :icon="Lock">策略</el-dropdown-item>
                  <el-dropdown-item command="logs" :icon="Monitor">日志</el-dropdown-item>
                  <el-dropdown-item command="toggle">{{ row.is_enabled ? '禁用' : '启用' }}</el-dropdown-item>
                  <el-dropdown-item command="delete" divided style="color: var(--el-color-danger)">删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
      </div>

      <div class="workspace-surface__pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @current-change="loadList"
          @size-change="(s: number) => { pageSize = s; page = 1; loadList() }"
        />
      </div>
    </template>

    <!-- ════════════════════════════════════════════════════════════════════════
         View 2: Tool List (sub-view)
         ════════════════════════════════════════════════════════════════════════ -->
    <template v-if="activeView === 'tools'">
      <div class="workspace-surface__body">
      <el-table
        v-loading="toolsLoading"
        :data="tools"
        class="console-table"
        stripe
        style="width: 100%"
        :empty-text="toolsError || '暂无工具'"
      >
        <el-table-column prop="name" label="工具名称" min-width="180" />
        <el-table-column prop="description" label="描述" min-width="250" show-overflow-tooltip />
        <el-table-column label="输入 Schema" min-width="300">
          <template #default="{ row }: { row: McpToolInfo }">
            <el-button
              size="small"
              text
              :icon="View"
              @click="
                expandedToolRows.includes(row.name)
                  ? expandedToolRows.splice(expandedToolRows.indexOf(row.name), 1)
                  : expandedToolRows.push(row.name)
              "
            >
              {{ expandedToolRows.includes(row.name) ? '收起' : '查看 Schema' }}
            </el-button>
            <pre v-if="expandedToolRows.includes(row.name)" class="schema-json">{{ formatSchema(row.input_schema) }}</pre>
          </template>
        </el-table-column>
        <el-table-column label="治理" width="110" fixed="right">
          <template #default="{ row }: { row: McpToolInfo }">
            <el-button size="small" text :icon="Lock" @click="openCreatePolicy(selectedServer || undefined, row.name)">
              配策略
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      </div>
    </template>

    <!-- ════════════════════════════════════════════════════════════════════════
         View 3: Tool Policies (sub-view)
         ════════════════════════════════════════════════════════════════════════ -->
    <template v-if="activeView === 'policies'">
      <div class="workspace-surface__toolbar">
        <div class="workspace-surface__filters">
          <el-input
            v-model="policyFilterKeyword"
            :prefix-icon="Search"
            clearable
            placeholder="搜索 Server / 工具"
            style="width: 220px"
          />
          <el-select
            v-if="!selectedServer"
            v-model="policyFilterServerId"
            clearable
            placeholder="全部 Server"
            style="width: 220px"
            @change="policyPage = 1; loadPolicies()"
          >
            <el-option
              v-for="server in policyServerOptions"
              :key="server.value"
              :label="server.label"
              :value="server.value"
            />
          </el-select>
          <el-button type="primary" @click="policyPage = 1; loadPolicies()">查询</el-button>
        </div>
      </div>

      <div class="workspace-surface__body">
        <el-table
          v-loading="policiesLoading"
          :data="filteredPolicies"
          class="console-table"
          stripe
          style="width: 100%"
          :empty-text="policiesError || '暂无工具策略'"
        >
          <el-table-column label="策略对象" min-width="220">
            <template #default="{ row }: { row: McpToolPolicy }">
              <div class="console-entity">
                <div class="console-entity__name">{{ row.tool_name }}</div>
                <div class="console-entity__meta">{{ policyServerName(row) }}</div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="效果" width="90">
            <template #default="{ row }: { row: McpToolPolicy }">
              <el-tag :type="row.effect === 'allow' ? 'success' : 'danger'" size="small">
                {{ POLICY_EFFECT_LABEL[row.effect] || row.effect }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="风险" width="90">
            <template #default="{ row }: { row: McpToolPolicy }">
              <el-tag :type="row.risk_level === 'high' || row.risk_level === 'critical' ? 'danger' : row.risk_level === 'medium' ? 'warning' : 'info'" size="small">
                {{ POLICY_RISK_LABEL[row.risk_level] || row.risk_level }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="确认" width="80">
            <template #default="{ row }: { row: McpToolPolicy }">
              <el-tag :type="row.require_confirmation ? 'warning' : 'info'" size="small" effect="plain">
                {{ row.require_confirmation ? '需要' : '不需要' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="角色 / Scope" min-width="180">
            <template #default="{ row }: { row: McpToolPolicy }">
              <div class="policy-chip-row">
                <el-tag v-for="role in row.allowed_roles" :key="`role-${role}`" size="small" effect="plain">{{ role }}</el-tag>
                <el-tag v-for="scope in row.allowed_scopes" :key="`scope-${scope}`" size="small" type="info" effect="plain">{{ scope }}</el-tag>
                <span v-if="!row.allowed_roles.length && !row.allowed_scopes.length" class="policy-muted">不限</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="参数策略" min-width="220">
            <template #default="{ row }: { row: McpToolPolicy }">
              <div class="policy-chip-row">
                <el-tag v-for="arg in row.required_args" :key="`required-${arg}`" size="small" type="success" effect="plain">必填 {{ arg }}</el-tag>
                <el-tag v-for="arg in row.denied_args" :key="`denied-${arg}`" size="small" type="danger" effect="plain">禁用 {{ arg }}</el-tag>
                <el-tag v-for="field in row.redact_fields" :key="`redact-${field}`" size="small" type="warning" effect="plain">脱敏 {{ field }}</el-tag>
                <span v-if="!row.required_args.length && !row.denied_args.length && !row.redact_fields.length" class="policy-muted">未配置</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="限流" width="130">
            <template #default="{ row }: { row: McpToolPolicy }">
              <span v-if="row.rate_limit_window_seconds > 0 && row.rate_limit_max_calls > 0">
                {{ row.rate_limit_window_seconds }}s / {{ row.rate_limit_max_calls }} 次
              </span>
              <span v-else class="policy-muted">未启用</span>
            </template>
          </el-table-column>
          <el-table-column label="启用" width="80">
            <template #default="{ row }: { row: McpToolPolicy }">
              <el-switch :model-value="row.is_enabled" size="small" @click="togglePolicyEnabled(row)" />
            </template>
          </el-table-column>
          <el-table-column label="操作" width="140" fixed="right">
            <template #default="{ row }: { row: McpToolPolicy }">
              <el-button size="small" text :icon="Edit" @click="openEditPolicy(row)">编辑</el-button>
              <el-button size="small" text type="danger" :icon="Delete" @click="deletePolicy(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="workspace-surface__pagination">
        <el-pagination
          v-model:current-page="policyPage"
          v-model:page-size="policyPageSize"
          :total="policyTotal"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @current-change="loadPolicies"
          @size-change="(s: number) => { policyPageSize = s; policyPage = 1; loadPolicies() }"
        />
      </div>
    </template>

    <!-- ════════════════════════════════════════════════════════════════════════
         View 3: Tool Call Logs (sub-view)
         ════════════════════════════════════════════════════════════════════════ -->
    <template v-if="activeView === 'logs'">
      <div class="workspace-surface__toolbar">
        <div class="workspace-surface__filters">
          <el-input
            v-model="logFilterToolName"
            placeholder="工具名"
            clearable
            style="width: 160px"
            @keyup.enter="logPage = 1; loadLogs()"
          />
          <el-date-picker
            v-model="logFilterStartTime"
            type="datetime"
            placeholder="开始时间"
            format="YYYY-MM-DD HH:mm"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 180px"
          />
          <el-date-picker
            v-model="logFilterEndTime"
            type="datetime"
            placeholder="结束时间"
            format="YYYY-MM-DD HH:mm"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 180px"
          />
          <el-button type="primary" @click="logPage = 1; loadLogs()">查询</el-button>
        </div>
        <div class="workspace-surface__actions">
        <el-button :icon="Refresh" @click="logPage = 1; loadLogs()">刷新</el-button>
        </div>
      </div>

      <div class="workspace-surface__body">
      <el-table
        v-loading="logsLoading"
        :data="logs"
        class="console-table"
        stripe
        style="width: 100%"
        :empty-text="logsError || '暂无日志'"
      >
        <el-table-column label="时间" width="170">
          <template #default="{ row }: { row: McpCallLog }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="tool_name" label="工具名" width="160" />
        <el-table-column label="入参摘要" min-width="200">
          <template #default="{ row }: { row: McpCallLog }">
            <code class="log-preview">{{ formatLogParam(row.input_args) }}</code>
          </template>
        </el-table-column>
        <el-table-column label="结果摘要" min-width="200">
          <template #default="{ row }: { row: McpCallLog }">
            <code class="log-preview">{{ formatLogParam(row.result) }}</code>
          </template>
        </el-table-column>
        <el-table-column label="耗时" width="90">
          <template #default="{ row }: { row: McpCallLog }">
            {{ formatDuration(row.duration_ms) }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }: { row: McpCallLog }">
            <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">
              {{ row.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="策略决策" width="120">
          <template #default="{ row }: { row: McpCallLog }">
            <el-tag
              v-if="row.policy_decision"
              :type="policyDecisionTagType(row.policy_decision)"
              size="small"
              effect="plain"
            >
              {{ POLICY_DECISION_LABEL[row.policy_decision] || row.policy_decision }}
            </el-tag>
            <span v-else class="policy-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="90" fixed="right">
          <template #default="{ row }: { row: McpCallLog }">
            <el-button size="small" text :icon="View" @click="openLogDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
      </div>

      <div class="workspace-surface__pagination">
        <el-pagination
          v-model:current-page="logPage"
          v-model:page-size="logPageSize"
          :total="logsTotal"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @current-change="loadLogs"
          @size-change="(s: number) => { logPageSize = s; logPage = 1; loadLogs() }"
        />
      </div>
    </template>

    <!-- ════════════════════════════════════════════════════════════════════════
         Create / Edit Dialog
         ════════════════════════════════════════════════════════════════════════ -->
    <el-drawer
      v-model="dialogVisible"
      :title="dialogTitle"
      size="640px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <el-form :model="dialogForm" label-width="130px">
        <el-form-item label="Server 名称" required>
          <el-input v-model="dialogForm.name" placeholder="英文标识，例如：my-mcp-server" :disabled="isEditing" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input
            v-model="dialogForm.description"
            type="textarea"
            :rows="2"
            placeholder="MCP Server 功能描述（选填）"
          />
        </el-form-item>
        <el-form-item label="传输类型" required>
          <el-select v-model="dialogForm.transport_type" style="width: 100%">
            <el-option
              v-for="opt in TRANSPORT_OPTIONS"
              :key="opt.value"
              :value="opt.value"
              :label="opt.label"
            />
          </el-select>
        </el-form-item>

        <!-- STDIO fields -->
        <template v-if="showCommandFields">
          <el-form-item label="命令" required>
            <el-input v-model="dialogForm.command" placeholder="例如：npx /path/to/mcp-server" />
          </el-form-item>
          <el-form-item label="参数">
            <el-input
              v-model="dialogForm.args"
              type="textarea"
              :rows="3"
              placeholder="每行一个参数"
            />
          </el-form-item>
          <el-form-item label="环境变量">
            <el-input
              v-model="dialogForm.env"
              type="textarea"
              :rows="3"
              placeholder="每行一个，格式：KEY=VALUE"
            />
          </el-form-item>
        </template>

        <!-- SSE/HTTP fields -->
        <template v-if="showUrlFields">
          <el-form-item label="URL" required>
            <el-input v-model="dialogForm.url" placeholder="例如：http://localhost:3000/mcp" />
          </el-form-item>
        </template>

        <el-form-item label="超时时间(秒)">
          <el-input-number v-model="dialogForm.timeout_seconds" :min="1" :max="300" :step="1" style="width: 100%" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="dialogForm.is_enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">
          {{ isEditing ? '保存' : '创建' }}
        </el-button>
      </template>
    </el-drawer>

    <!-- ════════════════════════════════════════════════════════════════════════
         Policy Dialog
         ════════════════════════════════════════════════════════════════════════ -->
    <el-drawer
      v-model="policyDialogVisible"
      :title="policyDialogTitle"
      size="720px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <el-form :model="policyForm" label-width="150px">
        <el-form-item label="MCP Server" required>
          <el-select v-model="policyForm.server_id" filterable style="width: 100%" :disabled="policyIsEditing">
            <el-option
              v-for="server in policyServerOptions"
              :key="server.value"
              :label="server.label"
              :value="server.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="工具名称" required>
          <el-input v-model="policyForm.tool_name" placeholder="MCP tool name" :disabled="policyIsEditing" />
        </el-form-item>
        <el-form-item label="策略效果">
          <el-radio-group v-model="policyForm.effect">
            <el-radio-button label="allow">允许</el-radio-button>
            <el-radio-button label="deny">拒绝</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="风险等级">
          <el-select v-model="policyForm.risk_level" style="width: 100%">
            <el-option label="低" value="low" />
            <el-option label="中" value="medium" />
            <el-option label="高" value="high" />
            <el-option label="严重" value="critical" />
          </el-select>
        </el-form-item>
        <el-form-item label="需要人工确认">
          <el-switch v-model="policyForm.require_confirmation" />
        </el-form-item>
        <el-form-item label="允许角色">
          <el-input
            v-model="policyForm.allowed_roles"
            type="textarea"
            :rows="3"
            placeholder="每行一个角色；留空表示不限"
          />
        </el-form-item>
        <el-form-item label="允许 Scope">
          <el-input
            v-model="policyForm.allowed_scopes"
            type="textarea"
            :rows="3"
            placeholder="每行一个 scope；留空表示不限"
          />
        </el-form-item>
        <el-form-item label="必填参数">
          <el-input
            v-model="policyForm.required_args"
            type="textarea"
            :rows="3"
            placeholder="每行一个参数路径，例如 candidate_id"
          />
        </el-form-item>
        <el-form-item label="禁用参数">
          <el-input
            v-model="policyForm.denied_args"
            type="textarea"
            :rows="3"
            placeholder="每行一个参数路径，例如 raw_password"
          />
        </el-form-item>
        <el-form-item label="参数规则 JSON">
          <el-input
            v-model="policyForm.arg_rules"
            type="textarea"
            :rows="5"
            placeholder='例如 {"salary.max": {"lte": 100000}}'
          />
        </el-form-item>
        <el-form-item label="脱敏字段">
          <el-input
            v-model="policyForm.redact_fields"
            type="textarea"
            :rows="3"
            placeholder="每行一个字段或路径，例如 token / password / authorization"
          />
        </el-form-item>
        <el-form-item label="限流窗口(秒)">
          <el-input-number v-model="policyForm.rate_limit_window_seconds" :min="0" :max="86400" :step="60" style="width: 100%" />
        </el-form-item>
        <el-form-item label="窗口最大调用">
          <el-input-number v-model="policyForm.rate_limit_max_calls" :min="0" :max="100000" :step="1" style="width: 100%" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="policyForm.is_enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="policyDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="policySaving" @click="savePolicy">
          {{ policyIsEditing ? '保存' : '创建' }}
        </el-button>
      </template>
    </el-drawer>

    <!-- ════════════════════════════════════════════════════════════════════════
         Log Detail Dialog
         ════════════════════════════════════════════════════════════════════════ -->
    <el-dialog
      v-model="logDetailVisible"
      title="调用日志详情"
      width="800px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <template v-if="logDetail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="时间" :span="2">{{ formatTime(logDetail.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="Server">{{ logDetail.server_name }}</el-descriptions-item>
          <el-descriptions-item label="工具名">{{ logDetail.tool_name }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="logDetail.status === 'success' ? 'success' : 'danger'" size="small">
              {{ logDetail.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="耗时">{{ formatDuration(logDetail.duration_ms) }}</el-descriptions-item>
          <el-descriptions-item label="调用者">{{ logDetail.called_by || '-' }}</el-descriptions-item>
          <el-descriptions-item label="策略决策">
            <el-tag
              v-if="logDetail.policy_decision"
              :type="policyDecisionTagType(logDetail.policy_decision)"
              size="small"
              effect="plain"
            >
              {{ POLICY_DECISION_LABEL[logDetail.policy_decision] || logDetail.policy_decision }}
            </el-tag>
            <span v-else>-</span>
          </el-descriptions-item>
          <el-descriptions-item label="策略 ID">{{ logDetail.policy_id || '-' }}</el-descriptions-item>
          <el-descriptions-item v-if="logDetail.policy_reason" label="策略原因" :span="2">
            {{ logDetail.policy_reason }}
          </el-descriptions-item>
          <el-descriptions-item v-if="logDetail.status === 'error'" label="错误信息" :span="2">
            <el-alert :title="logDetail.error_message" type="error" :closable="false" show-icon />
          </el-descriptions-item>
        </el-descriptions>

        <el-divider content-position="left">完整入参</el-divider>
        <pre class="log-detail-json">{{ formatLogParamFull(logDetail.input_args) }}</pre>

        <el-divider content-position="left">完整结果</el-divider>
        <pre class="log-detail-json">{{ formatLogParamFull(logDetail.result) }}</pre>
      </template>
      <template #footer>
        <el-button @click="logDetailVisible = false">关闭</el-button>
      </template>
    </el-dialog>
    </div>
  </div>
</template>

<style scoped>
.mcp-manage-view {
  padding-bottom: 24px;
}

.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  margin: 0;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  flex-wrap: wrap;
  gap: 8px;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.schema-json {
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  padding: 8px;
  font-size: 12px;
  line-height: 1.5;
  max-height: 300px;
  overflow: auto;
  margin: 8px 0 0;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-preview {
  font-size: 12px;
  line-height: 1.4;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--el-text-color-secondary);
}

.log-detail-json {
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  padding: 12px;
  font-size: 12px;
  line-height: 1.5;
  max-height: 300px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
}

.policy-chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  min-width: 0;
}

.policy-muted {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}
</style>
