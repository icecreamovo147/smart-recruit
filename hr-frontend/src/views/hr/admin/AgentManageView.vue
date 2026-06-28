<script setup lang="ts">
import { onMounted, reactive, ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Edit, Plus, Refresh } from '@element-plus/icons-vue'
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
  prompt_template_id: 0,
  instruction: '',
  max_iterations: 5,
  temperature_override: 0,
  temperature_override_enabled: false,
  is_default: false,
  is_enabled: true,
  capability_ids: [] as string[],
})

const currentToolOptions = computed(() => {
  if (capabilityList.value.length > 0) {
    return capabilityList.value.map((cap) => ({
      id: capabilitySelectID(cap.source, cap.key),
      label: capabilityDisplayLabel(cap),
      group: cap.source === 'mcp' ? 'MCP' : cap.source === 'skill' ? 'SKILL' : '内置',
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

const handleAgentTypeChange = async () => {
  dialogForm.capability_ids = []
  await loadCapabilities(dialogForm.agent_type)
}

const resetDialogForm = () => {
  dialogForm.name = ''
  dialogForm.display_name = ''
  dialogForm.description = ''
  dialogForm.agent_type = 'hr_recruiting_agent'
  dialogForm.prompt_template_id = 0
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
  dialogForm.prompt_template_id = row.prompt_template_id || 0
  dialogForm.instruction = row.instruction || ''
  dialogForm.max_iterations = row.max_iterations || 5
  dialogForm.temperature_override = row.temperature_override || 0
  dialogForm.temperature_override_enabled = row.temperature_override > 0
  dialogForm.is_default = row.is_default
  dialogForm.is_enabled = row.is_enabled
  await loadCapabilities(row.agent_type)
  const capabilityBindings = row.capability_bindings || []
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
      payload.prompt_template_id = dialogForm.prompt_template_id
      payload.prompt_template_id_set = dialogForm.prompt_template_id > 0
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
      if (dialogForm.prompt_template_id > 0) payload.prompt_template_id = dialogForm.prompt_template_id
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
    <h2 class="page-title">Agent 管理</h2>

    <div class="toolbar">
      <div class="toolbar-left">
        <el-button type="primary" :icon="Plus" @click="openCreate">新增 Agent</el-button>
        <el-select
          v-model="agentTypeFilter"
          placeholder="全部类型"
          clearable
          style="width: 180px"
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
      </div>
      <el-button :icon="Refresh" @click="loadList">刷新</el-button>
    </div>

    <!-- ── List Table ──────────────────────────────────────────────── -->
    <el-table
      v-loading="loading"
      :data="list"
      stripe
      border
      style="width: 100%"
      :empty-text="error || '暂无数据'"
    >
      <el-table-column prop="display_name" label="显示名称" min-width="140" />
      <el-table-column prop="name" label="标识" width="180" />
      <el-table-column label="类型" width="140">
        <template #default="{ row }: { row: AgentConfigInfo }">
          {{ AGENT_TYPE_LABEL[row.agent_type] || row.agent_type }}
        </template>
      </el-table-column>
      <el-table-column label="绑定 Prompt" width="160" show-overflow-tooltip>
        <template #default="{ row }: { row: AgentConfigInfo }">
          {{ row.prompt_template_name || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="工具数" width="80">
        <template #default="{ row }: { row: AgentConfigInfo }">
          {{ (row.capability_bindings || row.tool_bindings || []).length }}
        </template>
      </el-table-column>
      <el-table-column label="默认" width="70">
        <template #default="{ row }: { row: AgentConfigInfo }">
          <el-tag v-if="row.is_default" type="warning" size="small">默认</el-tag>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="80">
        <template #default="{ row }: { row: AgentConfigInfo }">
          <el-tag :type="row.is_enabled ? 'success' : 'info'" size="small">
            {{ row.is_enabled ? '启用' : '禁用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="更新时间" width="170">
        <template #default="{ row }: { row: AgentConfigInfo }">
          {{ formatTime(row.updated_at) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="170" fixed="right">
        <template #default="{ row }: { row: AgentConfigInfo }">
          <el-button size="small" :icon="Edit" @click="openEdit(row)">
            编辑
          </el-button>
          <el-button size="small" type="danger" :icon="Delete" @click="handleDelete(row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-wrap">
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

    <!-- ── Edit / Create Dialog ────────────────────────────────────── -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="640px"
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
    </el-dialog>
  </div>
</template>

<style scoped>
.agent-manage-view {
  padding-bottom: 24px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  margin: 0 0 16px;
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
}

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
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
</style>
