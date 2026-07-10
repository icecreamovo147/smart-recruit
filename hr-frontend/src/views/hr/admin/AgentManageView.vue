<script setup lang="ts">
import { onMounted, reactive, ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown, Delete, Edit, MoreFilled, Plus, Refresh, Search, View } from '@element-plus/icons-vue'
import {
  listAgentConfigs,
  listAgentCapabilities,
  createAgentConfig,
  updateAgentConfig,
  deleteAgentConfig,
} from '@/api/agent'
import { listPromptTemplates } from '@/api/prompt'
import type {
  AgentConfigInfo,
  AgentCapabilityBindingInfo,
  CapabilityInfo,
  AgentToolBindingInfo,
  CreateAgentPayload,
  UpdateAgentPayload,
} from '@/types/agent'
import type { PromptTemplate } from '@/types/prompt'

// ====== Agent types dropdown options ======

const AGENT_TYPE_OPTIONS = [
  { value: 'hr_recruiting_agent', label: 'HR 招聘助手' },
  { value: 'candidate_assistant', label: '候选人 AI 助手' },
  { value: 'custom', label: '自定义' },
]

const AGENT_TYPE_LABEL: Record<string, string> = {
  hr_recruiting_agent: 'HR 招聘助手',
  candidate_assistant: '候选人 AI 助手',
  custom: '自定义',
}

// ====== Available tool names (from backend hardcoded tools) ======

const AVAILABLE_TOOLS_BY_TYPE: Record<string, { name: string; label: string }[]> = {
  hr_recruiting_agent: [
    { name: 'query_total_applications', label: '查询总投递量' },
    { name: 'query_today_applications', label: '查询今日投递' },
    { name: 'get_job_heat_ranking', label: '岗位热度排行' },
    { name: 'search_candidates', label: '搜索候选人' },
    { name: 'get_job_detail', label: '获取岗位详情' },
    { name: 'search_jobs', label: '搜索岗位' },
    { name: 'get_candidate_detail', label: '获取候选人详情' },
    { name: 'propose_application_status_update', label: '更新投递状态' },
    { name: 'list_all_applications', label: '列出所有投递' },
    { name: 'list_applications_by_job', label: '按岗位列出投递' },
    { name: 'list_applications_by_status', label: '按状态列出投递' },
    { name: 'get_application_status_summary', label: '投递状态汇总' },
    { name: 'get_application_trend', label: '投递趋势' },
    { name: 'get_job_list', label: '岗位列表' },
  ],
  candidate_assistant: [
    { name: 'list_my_applications', label: '我的投递列表' },
    { name: 'get_my_application_detail', label: '投递详情' },
    { name: 'get_my_resume_text', label: '我的简历' },
    { name: 'list_jobs_for_recommendation', label: '推荐岗位' },
    { name: 'get_job_detail_for_candidate', label: '岗位详情' },
    { name: 'recommend_jobs_by_resume', label: '按简历推荐岗位' },
  ],
  custom: [],
}

// ====== List State ======

const list = ref<AgentConfigInfo[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const error = ref('')
const agentTypeFilter = ref('')
const keywordFilter = ref('')
const statusFilter = ref('')
const promptFilter = ref('')

const loadList = async () => {
  loading.value = true
  error.value = ''
  try {
    const data = await listAgentConfigs(
      page.value,
      pageSize.value,
      agentTypeFilter.value || undefined,
    )
    list.value = data.list || []
    total.value = data.total || 0
  } catch (e: unknown) {
    error.value = (e as { message?: string }).message || '加载 Agent 配置列表失败'
  } finally {
    loading.value = false
  }
}

// ====== Reference data for selectors ======

const promptList = ref<PromptTemplate[]>([])
const capabilityList = ref<CapabilityInfo[]>([])

const loadReferenceData = async () => {
  try {
    const promptData = await listPromptTemplates(1, 200)
    promptList.value = promptData.list || []
  } catch {
    // Non-fatal: selectors will be empty but user can still type
  }
}

const loadCapabilities = async (agentType = dialogForm.agent_type) => {
  try {
    const data = await listAgentCapabilities(agentType)
    capabilityList.value = data.list || []
  } catch {
    capabilityList.value = []
  }
}

// ====== Helpers ======

const formatTime = (s?: string): string => {
  if (!s) return '-'
  return new Date(s).toLocaleString('zh-CN')
}

const agentTypeLabel = (type: string) => AGENT_TYPE_LABEL[type] || type || '-'

const getAgentCapabilityCount = (row: AgentConfigInfo) =>
  configurableBindings(row.capability_bindings || []).length || (row.tool_bindings || []).length

const filteredList = computed(() => {
  const keyword = keywordFilter.value.trim().toLowerCase()
  return list.value.filter((item) => {
    const matchesKeyword = !keyword
      || item.name.toLowerCase().includes(keyword)
      || item.display_name.toLowerCase().includes(keyword)
    const matchesStatus = !statusFilter.value
      || (statusFilter.value === 'enabled' ? item.is_enabled : !item.is_enabled)
    const hasPrompt = Boolean(item.prompt_template_id || item.prompt_template_name)
    const matchesPrompt = !promptFilter.value
      || (promptFilter.value === 'bound' ? hasPrompt : !hasPrompt)
    return matchesKeyword && matchesStatus && matchesPrompt
  })
})

const resetFilters = () => {
  keywordFilter.value = ''
  agentTypeFilter.value = ''
  statusFilter.value = ''
  promptFilter.value = ''
  page.value = 1
  loadList()
}

const bindingLabel = (binding: AgentCapabilityBindingInfo) => {
  const sourceLabel: Record<string, string> = {
    builtin: '内置',
    mcp: 'MCP',
    skill: 'SKILL',
  }
  return `${sourceLabel[binding.capability_source] || binding.capability_source} / ${binding.capability_key}`
}

// ====== Edit / Create Dialog ======

const dialogVisible = ref(false)
const dialogTitle = ref('')
const saving = ref(false)
const isEditing = ref(false)
const editingId = ref(0)
const dialogForm = reactive({
  name: '',
  display_name: '',
  description: '',
  agent_type: 'hr_recruiting_agent',
  prompt_template_id: null as number | null,
  instruction: '',
  max_iterations: 5,
  temperature_override: 0,
  temperature_override_enabled: false,
  is_default: false,
  is_enabled: true,
  capability_ids: [] as string[],
})

const detailVisible = ref(false)
const detailAgent = ref<AgentConfigInfo | null>(null)

const openDetail = (row: AgentConfigInfo) => {
  detailAgent.value = row
  detailVisible.value = true
}

const currentToolOptions = computed(() => {
  const configurableCapabilities = capabilityList.value.filter((cap) => cap.source !== 'skill')
  if (configurableCapabilities.length > 0) {
    return configurableCapabilities.map((cap) => ({
      id: capabilitySelectID(cap.source, cap.key),
      label: capabilityDisplayLabel(cap),
      group: cap.source === 'mcp' ? 'MCP' : '内置',
      description: cap.description,
      disabled: !cap.is_available,
    }))
  }
  return (AVAILABLE_TOOLS_BY_TYPE[dialogForm.agent_type] || []).map((tool) => ({
    id: capabilitySelectID('builtin', tool.name),
    label: tool.label,
    group: '内置',
    description: '',
    disabled: false,
  }))
})

const capabilityDisplayLabel = (cap: CapabilityInfo) => {
  if (cap.source === 'builtin') {
    const builtin = (AVAILABLE_TOOLS_BY_TYPE[dialogForm.agent_type] || []).find((tool) => tool.name === cap.key)
    return builtin?.label || cap.display_name || cap.name || cap.key
  }
  return cap.display_name || cap.name || cap.key
}

const selectedCapabilities = computed(() =>
  dialogForm.capability_ids.map(capabilityFromSelectID).filter((cap): cap is AgentCapabilityBindingInfo => Boolean(cap)),
)

const capabilitySelectID = (source: string, key: string) => `${source}:${key}`

const capabilityFromSelectID = (id: string): AgentCapabilityBindingInfo | null => {
  const index = id.indexOf(':')
  if (index <= 0) return null
  const source = id.slice(0, index) as AgentCapabilityBindingInfo['capability_source']
  const key = id.slice(index + 1)
  if (!source || !key) return null
  return {
    capability_source: source,
    capability_key: key,
    is_enabled: true,
    priority: 0,
  }
}

const capabilityIDFromBinding = (binding: AgentCapabilityBindingInfo) =>
  capabilitySelectID(binding.capability_source, binding.capability_key)

const configurableBindings = (bindings: AgentCapabilityBindingInfo[] = []) =>
  bindings.filter((binding) => binding.capability_source !== 'skill')

const legacySkillBindings = (bindings: AgentCapabilityBindingInfo[] = []) =>
  bindings.filter((binding) => binding.capability_source === 'skill')

const handleAgentTypeChange = async () => {
  dialogForm.capability_ids = []
  await loadCapabilities(dialogForm.agent_type)
}

const resetDialogForm = () => {
  dialogForm.name = ''
  dialogForm.display_name = ''
  dialogForm.description = ''
  dialogForm.agent_type = 'hr_recruiting_agent'
  dialogForm.prompt_template_id = null
  dialogForm.instruction = ''
  dialogForm.max_iterations = 5
  dialogForm.temperature_override = 0
  dialogForm.temperature_override_enabled = false
  dialogForm.is_default = false
  dialogForm.is_enabled = true
  dialogForm.capability_ids = []
}

const openCreate = async () => {
  isEditing.value = false
  editingId.value = 0
  dialogTitle.value = '新增 Agent 配置'
  resetDialogForm()
  await loadCapabilities(dialogForm.agent_type)
  dialogVisible.value = true
}

const openEdit = async (row: AgentConfigInfo) => {
  isEditing.value = true
  editingId.value = row.id
  dialogTitle.value = '编辑 Agent 配置'
  dialogForm.name = row.name
  dialogForm.display_name = row.display_name
  dialogForm.description = row.description || ''
  dialogForm.agent_type = row.agent_type
  dialogForm.prompt_template_id = row.prompt_template_id || null
  dialogForm.instruction = row.instruction || ''
  dialogForm.max_iterations = row.max_iterations || 5
  dialogForm.temperature_override = row.temperature_override || 0
  dialogForm.temperature_override_enabled = row.temperature_override > 0
  dialogForm.is_default = row.is_default
  dialogForm.is_enabled = row.is_enabled
  await loadCapabilities(row.agent_type)
  const capabilityBindings = configurableBindings(row.capability_bindings || [])
  dialogForm.capability_ids = capabilityBindings.length > 0
    ? capabilityBindings
      .filter((binding) => binding.is_enabled !== false)
      .map(capabilityIDFromBinding)
    : (row.tool_bindings || [])
      .filter((tb: AgentToolBindingInfo) => tb.is_enabled)
      .map((tb: AgentToolBindingInfo) => capabilitySelectID('builtin', tb.tool_name))
  dialogVisible.value = true
}

const save = async () => {
  if (!dialogForm.name) {
    ElMessage.warning('请输入 Agent 名称')
    return
  }
  if (!dialogForm.display_name) {
    ElMessage.warning('请输入显示名称')
    return
  }
  if (!dialogForm.agent_type) {
    ElMessage.warning('请选择 Agent 类型')
    return
  }
  saving.value = true
  try {
    if (isEditing.value) {
      const payload: UpdateAgentPayload = {
        name: dialogForm.name,
        display_name: dialogForm.display_name,
      }
      if (dialogForm.description) payload.description = dialogForm.description
      payload.prompt_template_id = dialogForm.prompt_template_id ?? undefined
      payload.prompt_template_id_set = (dialogForm.prompt_template_id ?? 0) > 0
      if (dialogForm.instruction) payload.instruction = dialogForm.instruction
      payload.max_iterations = dialogForm.max_iterations
      payload.max_iterations_set = true
      payload.temperature_override = dialogForm.temperature_override
      payload.temperature_override_set = dialogForm.temperature_override_enabled
      payload.is_default = dialogForm.is_default
      payload.is_default_set = true
      payload.is_enabled = dialogForm.is_enabled
      payload.is_enabled_set = true
      payload.capability_bindings = selectedCapabilities.value
      payload.capability_bindings_set = true
      await updateAgentConfig(editingId.value, payload)
      ElMessage.success('Agent 配置已更新')
    } else {
      const payload: CreateAgentPayload = {
        name: dialogForm.name,
        display_name: dialogForm.display_name,
        agent_type: dialogForm.agent_type,
      }
      if (dialogForm.description) payload.description = dialogForm.description
      if ((dialogForm.prompt_template_id ?? 0) > 0) payload.prompt_template_id = dialogForm.prompt_template_id ?? undefined
      if (dialogForm.instruction) payload.instruction = dialogForm.instruction
      payload.max_iterations = dialogForm.max_iterations
      payload.temperature_override = dialogForm.temperature_override
      payload.temperature_override_set = dialogForm.temperature_override_enabled
      payload.is_default = dialogForm.is_default
      if (selectedCapabilities.value.length > 0) payload.capability_bindings = selectedCapabilities.value
      await createAgentConfig(payload)
      ElMessage.success('Agent 配置已创建')
    }
    dialogVisible.value = false
    await loadList()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '保存失败')
  } finally {
    saving.value = false
  }
}

// ====== Delete ======

const handleDelete = async (row: AgentConfigInfo) => {
  try {
    await ElMessageBox.confirm(
      `确认删除 Agent 配置「${row.display_name}」？此操作不可撤销。`,
      '删除确认',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return
  }
  try {
    await deleteAgentConfig(row.id)
    ElMessage.success('Agent 配置已删除')
    await loadList()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '删除失败')
  }
}

// ====== Init ======

onMounted(() => {
  loadList()
  loadReferenceData()
})
</script>

<template>
  <div class="agent-manage-view">
    <div class="workspace-surface">
      <div class="workspace-surface__header">
        <div class="workspace-surface__header-copy">
          <p class="page-kicker">AI Agent Console</p>
          <h2 class="page-title">Agent 管理</h2>
          <p class="page-desc">配置 HR 后台可调用的 AI Agent、Prompt 绑定、工具能力和运行参数。</p>
        </div>
        <div class="workspace-surface__header-actions">
          <el-button :icon="Refresh" @click="loadList">刷新</el-button>
          <el-button type="primary" :icon="Plus" @click="openCreate">新增 Agent</el-button>
        </div>
      </div>

      <div class="workspace-surface__divider"></div>

      <div class="workspace-surface__toolbar">
        <div class="workspace-surface__filters">
          <el-input
            v-model="keywordFilter"
            style="width: 260px"
            :prefix-icon="Search"
            clearable
            placeholder="搜索名称 / 标识"
          />
          <el-select
            v-model="agentTypeFilter"
            placeholder="全部类型"
            clearable
            style="width: 168px"
            @change="() => { page = 1; loadList() }"
          >
            <el-option value="" label="全部类型" />
            <el-option
              v-for="opt in AGENT_TYPE_OPTIONS"
              :key="opt.value"
              :value="opt.value"
              :label="opt.label"
            />
          </el-select>
          <el-select v-model="statusFilter" placeholder="全部状态" clearable style="width: 140px">
            <el-option value="enabled" label="已启用" />
            <el-option value="disabled" label="已禁用" />
          </el-select>
          <el-select v-model="promptFilter" placeholder="Prompt 绑定" clearable style="width: 150px">
            <el-option value="bound" label="已绑定 Prompt" />
            <el-option value="unbound" label="未绑定 Prompt" />
          </el-select>
        </div>
        <div class="workspace-surface__actions">
          <el-button @click="resetFilters">重置</el-button>
          <el-button :icon="Refresh" @click="loadList">刷新</el-button>
        </div>
      </div>

      <div class="workspace-surface__body">
        <el-table
          v-loading="loading"
          :data="filteredList"
          stripe
          style="width: 100%"
          :empty-text="error || '暂无 Agent 配置'"
          @row-click="openDetail"
        >
          <el-table-column label="Agent 信息" min-width="240">
            <template #default="{ row }: { row: AgentConfigInfo }">
              <div class="entity-cell">
                <div class="entity-title">{{ row.display_name || row.name }}</div>
                <div class="entity-sub">{{ row.name }}</div>
                <div v-if="row.description" class="entity-desc">{{ row.description }}</div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="类型" min-width="140">
            <template #default="{ row }: { row: AgentConfigInfo }">
              <el-tag effect="plain">{{ agentTypeLabel(row.agent_type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="Prompt" min-width="180" show-overflow-tooltip>
            <template #default="{ row }: { row: AgentConfigInfo }">
              <span v-if="row.prompt_template_name">{{ row.prompt_template_name }}</span>
              <el-tag v-else size="small" type="info">未绑定</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="能力数" width="92">
            <template #default="{ row }: { row: AgentConfigInfo }">
              <el-tag size="small" type="primary">{{ getAgentCapabilityCount(row) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="运行参数" min-width="150">
            <template #default="{ row }: { row: AgentConfigInfo }">
              <div class="runtime-cell">
                <span>迭代 {{ row.max_iterations || '-' }}</span>
                <span>温度 {{ row.temperature_override > 0 ? row.temperature_override.toFixed(2) : '默认' }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="默认 / 状态" width="128">
            <template #default="{ row }: { row: AgentConfigInfo }">
              <div class="tag-stack">
                <el-tag v-if="row.is_default" type="warning" size="small">默认</el-tag>
                <el-tag :type="row.is_enabled ? 'success' : 'info'" size="small">
                  {{ row.is_enabled ? '启用' : '禁用' }}
                </el-tag>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="更新时间" width="170">
            <template #default="{ row }: { row: AgentConfigInfo }">
              {{ formatTime(row.updated_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="168" fixed="right">
            <template #default="{ row }: { row: AgentConfigInfo }">
              <el-button size="small" :icon="Edit" @click.stop="openEdit(row)">编辑</el-button>
              <el-button size="small" :icon="View" @click.stop="openDetail(row)">详情</el-button>
              <el-dropdown trigger="click" @click.stop>
                <el-button size="small">更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item :icon="Delete" @click="handleDelete(row)">删除</el-dropdown-item>
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
    </div>

    <el-drawer
      v-model="dialogVisible"
      :title="dialogTitle"
      size="680px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <el-form :model="dialogForm" label-width="130px">
        <el-form-item label="Agent 名称" required>
          <el-input v-model="dialogForm.name" placeholder="英文标识，例如：hr_recruiting_agent" :disabled="isEditing" />
        </el-form-item>
        <el-form-item label="显示名称" required>
          <el-input v-model="dialogForm.display_name" placeholder="例如：HR 招聘助手" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input
            v-model="dialogForm.description"
            type="textarea"
            :rows="2"
            placeholder="Agent 功能描述（选填）"
          />
        </el-form-item>
        <el-form-item label="Agent 类型" required>
          <el-select v-model="dialogForm.agent_type" style="width: 100%" @change="handleAgentTypeChange">
            <el-option
              v-for="opt in AGENT_TYPE_OPTIONS"
              :key="opt.value"
              :value="opt.value"
              :label="opt.label"
            />
          </el-select>
        </el-form-item>

        <el-divider content-position="left">Prompt 配置</el-divider>

        <el-form-item label="绑定 Prompt">
          <el-select v-model="dialogForm.prompt_template_id" style="width: 100%" placeholder="选择 Prompt 模板" clearable>
            <el-option
              v-for="p in promptList"
              :key="p.id"
              :value="p.id"
              :label="p.name"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="额外指令">
          <el-input
            v-model="dialogForm.instruction"
            type="textarea"
            :rows="3"
            placeholder="附加在 Prompt 之后的额外指令（选填）"
          />
        </el-form-item>

        <el-divider content-position="left">运行参数</el-divider>

        <el-form-item label="最大迭代次数">
          <el-input-number v-model="dialogForm.max_iterations" :min="1" :max="50" :step="1" style="width: 100%" />
        </el-form-item>
        <el-form-item label="温度覆盖">
          <div class="slider-with-toggle">
            <el-switch v-model="dialogForm.temperature_override_enabled" style="margin-right: 8px; flex-shrink: 0" />
            <el-slider
              v-model="dialogForm.temperature_override"
              :disabled="!dialogForm.temperature_override_enabled"
              :min="0"
              :max="2"
              :step="0.01"
              style="flex: 1"
            />
            <span class="slider-value">{{ dialogForm.temperature_override.toFixed(2) }}</span>
          </div>
          <div class="form-help-text">开启后覆盖模型默认温度，关闭则使用模型默认值</div>
        </el-form-item>

        <el-divider content-position="left">工具绑定</el-divider>

        <el-form-item label="绑定工具">
          <el-select
            v-model="dialogForm.capability_ids"
            multiple
            style="width: 100%"
            placeholder="选择要绑定的工具（多选）"
          >
            <el-option
              v-for="tool in currentToolOptions"
              :key="tool.id"
              :value="tool.id"
              :label="tool.label"
              :disabled="tool.disabled"
            />
          </el-select>
          <div v-if="currentToolOptions.length === 0" class="form-help-text">当前 Agent 类型没有预定义工具</div>
        </el-form-item>

        <el-divider content-position="left">其他</el-divider>

        <el-form-item label="设为默认">
          <el-switch v-model="dialogForm.is_default" />
          <div class="form-help-text">同一类型下最多一个默认配置</div>
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

    <el-drawer v-model="detailVisible" title="Agent 详情" size="560px" destroy-on-close>
      <template v-if="detailAgent">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="显示名称">{{ detailAgent.display_name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="标识">{{ detailAgent.name }}</el-descriptions-item>
          <el-descriptions-item label="类型">{{ agentTypeLabel(detailAgent.agent_type) }}</el-descriptions-item>
          <el-descriptions-item label="描述">{{ detailAgent.description || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Prompt">{{ detailAgent.prompt_template_name || '未绑定' }}</el-descriptions-item>
          <el-descriptions-item label="运行参数">
            最大迭代 {{ detailAgent.max_iterations || '-' }}；
            温度 {{ detailAgent.temperature_override > 0 ? detailAgent.temperature_override.toFixed(2) : '默认' }}
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <div class="tag-stack inline">
              <el-tag v-if="detailAgent.is_default" type="warning" size="small">默认</el-tag>
              <el-tag :type="detailAgent.is_enabled ? 'success' : 'info'" size="small">
                {{ detailAgent.is_enabled ? '启用' : '禁用' }}
              </el-tag>
            </div>
          </el-descriptions-item>
          <el-descriptions-item label="更新时间">{{ formatTime(detailAgent.updated_at) }}</el-descriptions-item>
        </el-descriptions>

        <h3 class="detail-title">工具能力</h3>
        <div v-if="configurableBindings(detailAgent.capability_bindings || []).length" class="binding-list">
          <el-tag
            v-for="binding in configurableBindings(detailAgent.capability_bindings || [])"
            :key="bindingLabel(binding)"
            effect="plain"
          >
            {{ bindingLabel(binding) }}
          </el-tag>
        </div>
        <div v-else-if="(detailAgent.tool_bindings || []).length" class="binding-list">
          <el-tag v-for="tool in detailAgent.tool_bindings" :key="tool.id" effect="plain">
            {{ tool.tool_name }}
          </el-tag>
        </div>
        <el-empty v-else description="暂无绑定能力" :image-size="72" />

        <template v-if="legacySkillBindings(detailAgent.capability_bindings || []).length">
          <h3 class="detail-title">历史工具能力（只读）</h3>
          <div class="binding-list">
            <el-tag
              v-for="binding in legacySkillBindings(detailAgent.capability_bindings || [])"
              :key="bindingLabel(binding)"
              effect="plain"
              type="info"
            >
              {{ binding.capability_key }}
            </el-tag>
          </div>
          <div class="form-help-text">这些旧系统级 Skill 绑定仅用于历史展示，编辑保存时不会混入可配置工具能力。</div>
        </template>

        <h3 class="detail-title">额外指令</h3>
        <pre class="instruction-preview">{{ detailAgent.instruction || '暂无额外指令' }}</pre>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped>
.agent-manage-view {
  height: 100%;
  min-height: 0;
  padding-bottom: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.page-header,
.filter-toolbar,
.filter-actions,
.page-actions,
.toolbar-left {
  display: flex;
  align-items: center;
}

.page-header {
  justify-content: space-between;
  align-items: flex-start;
  gap: 20px;
  margin-bottom: 18px;
  padding: 22px 24px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--admin-console-header-bg);
  flex-shrink: 0;
}

.page-title {
  font-size: 24px;
  font-weight: 700;
  margin: 0;
  line-height: 1.25;
}

.page-kicker {
  margin: 0 0 6px;
  font-size: 12px;
  font-weight: 700;
  line-height: 1;
  color: var(--el-color-primary);
  text-transform: uppercase;
  letter-spacing: 0;
}

.page-desc {
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.table-card {
  border-radius: 8px;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.table-card :deep(.el-card__body) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.filter-toolbar {
  gap: 10px;
  flex-wrap: wrap;
  margin-bottom: 12px;
  flex-shrink: 0;
}

.table-card :deep(.el-table) {
  flex: 1;
  min-height: 320px;
}

.table-card :deep(.el-table__body-wrapper) {
  overflow-y: auto;
}

.filter-search {
  width: 260px;
}

.filter-select {
  width: 168px;
}

.filter-actions {
  gap: 8px;
  margin-left: auto;
}

.entity-cell {
  min-width: 0;
}

.entity-title {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.entity-sub,
.entity-desc,
.runtime-cell {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.entity-desc {
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.runtime-cell,
.tag-stack {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
}

.tag-stack.inline {
  flex-direction: row;
  align-items: center;
}

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
  flex-shrink: 0;
}

.slider-with-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.slider-value {
  min-width: 40px;
  text-align: right;
  font-family: monospace;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  flex-shrink: 0;
}

.form-help-text {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  margin-top: 4px;
  line-height: 1.4;
}

.detail-title {
  margin: 18px 0 10px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.binding-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.instruction-preview {
  max-height: 220px;
  overflow: auto;
  margin: 0;
  padding: 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-regular);
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
}

@media (max-width: 900px) {
  .page-header,
  .filter-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .filter-search,
  .filter-select {
    width: 100%;
  }

  .filter-actions {
    width: 100%;
    margin-left: 0;
  }
}
</style>
