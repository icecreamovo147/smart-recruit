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
} from '@element-plus/icons-vue'
import {
  listMcpServers,
  createMcpServer,
  updateMcpServer,
  deleteMcpServer,
  testMcpServerConnection,
  listMcpServerTools,
  listMcpServerLogs,
} from '@/api/mcp'
import type {
  McpServerInfo,
  McpToolInfo,
  McpCallLog,
  CreateMcpServerPayload,
  UpdateMcpServerPayload,
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

// ====== View state ======

type ActiveView = 'list' | 'tools' | 'logs'

const activeView = ref<ActiveView>('list')
const selectedServer = ref<McpServerInfo | null>(null)

const viewTitle = computed(() => {
  if (activeView.value === 'tools' && selectedServer.value) {
    return `工具列表 - ${selectedServer.value.name}`
  }
  if (activeView.value === 'logs' && selectedServer.value) {
    return `调用日志 - ${selectedServer.value.name}`
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
    <div class="console-header">
      <div class="console-header__copy">
        <p class="console-eyebrow">TOOL CENTER</p>
        <h2 class="console-title">{{ viewTitle }}</h2>
        <p class="console-description">接入和管理 MCP Server，统一查看工具能力、连接状态和调用审计，作为 Agent 工具层的控制台。</p>
      </div>
      <div class="console-header__actions">
      <el-button
        v-if="activeView !== 'list'"
        :icon="Sort"
        @click="goBackToList"
      >
        返回列表
      </el-button>
      <el-button v-if="activeView === 'list'" :icon="Refresh" @click="loadList">刷新</el-button>
      <el-button v-if="activeView === 'list'" type="primary" :icon="Plus" @click="openCreate">新增 Server</el-button>
      </div>
    </div>

    <!-- ════════════════════════════════════════════════════════════════════════
         View 1: MCP Server List (Main)
         ════════════════════════════════════════════════════════════════════════ -->
    <template v-if="activeView === 'list'">
      <div class="console-card console-card--fill">
        <div class="console-card__head">
          <div>
            <h3 class="console-card__title">MCP Server 列表</h3>
            <p class="console-card__desc">共 {{ filteredServers.length }} 个匹配服务</p>
          </div>
        </div>
        <div class="console-toolbar">
          <div class="console-toolbar__filters">
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
      <div class="console-table-wrap">
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
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }: { row: McpServerInfo }">
            <el-button size="small" :icon="Connection" @click="handleTestConnection(row)" :loading="testingId === row.id">
              测试
            </el-button>
            <el-button size="small" :icon="Document" @click="navigateToTools(row)">
              工具
            </el-button>
            <el-button size="small" :icon="Monitor" @click="navigateToLogs(row)">
              日志
            </el-button>
            <el-button size="small" :icon="Edit" @click="openEdit(row)">
              编辑
            </el-button>
            <el-dropdown trigger="click">
              <el-button size="small" :icon="MoreFilled" circle />
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="handleToggleEnabled(row)">{{ row.is_enabled ? '禁用' : '启用' }}</el-dropdown-item>
                  <el-dropdown-item divided style="color: var(--el-color-danger)" @click="handleDelete(row)">
                    <el-icon><Delete /></el-icon>删除
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
      </div>

      <div class="console-pagination">
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
      </div>
    </template>

    <!-- ════════════════════════════════════════════════════════════════════════
         View 2: Tool List (sub-view)
         ════════════════════════════════════════════════════════════════════════ -->
    <template v-if="activeView === 'tools'">
      <div class="console-card console-card--fill">
        <div class="console-card__head">
          <div>
            <h3 class="console-card__title">工具能力</h3>
            <p class="console-card__desc">{{ selectedServer?.name }} 暴露的 Tool 与输入 Schema</p>
          </div>
          <el-button :icon="Refresh" @click="loadTools">刷新</el-button>
        </div>
        <div class="console-table-wrap">
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
      </el-table>
        </div>
      </div>
    </template>

    <!-- ════════════════════════════════════════════════════════════════════════
         View 3: Tool Call Logs (sub-view)
         ════════════════════════════════════════════════════════════════════════ -->
    <template v-if="activeView === 'logs'">
      <div class="console-card console-card--fill">
      <div class="console-toolbar">
        <div class="console-toolbar__filters">
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
        <div class="console-toolbar__actions">
        <el-button :icon="Refresh" @click="logPage = 1; loadLogs()">刷新</el-button>
        </div>
      </div>

      <div class="console-table-wrap">
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
        <el-table-column label="操作" width="90" fixed="right">
          <template #default="{ row }: { row: McpCallLog }">
            <el-button size="small" text :icon="View" @click="openLogDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
      </div>

      <div class="console-pagination">
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
</style>
