<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Check, Document, Edit, Plus, Refresh, Search, Sort, Tools, View } from '@element-plus/icons-vue'
import {
  activateSkillVersion,
  createSkill,
  createSkillVersion,
  listSkills,
  listSkillTools,
  listSkillVersions,
  updateSkill,
  updateSkillTool,
} from '@/api/skill'
import { useAuthStore } from '@/stores/auth'
import { PLATFORM_PERMISSIONS } from '@/permissions'
import type {
  CreateSkillPayload,
  SkillInfo,
  SkillToolInfo,
  SkillVersionInfo,
  UpdateSkillPayload,
  UpdateSkillToolPayload,
} from '@shared/types/skill'

const canManage = computed(() => useAuthStore().can(PLATFORM_PERMISSIONS.AI_CONFIG_MANAGE))

type ActiveView = 'list' | 'versions' | 'tools'
type TagType = 'success' | 'info' | 'warning' | 'danger' | 'primary'
type ToolVersionValue = number | 'current' | 'all'
type ToolVersionOption = { value: ToolVersionValue; label: string }

const SOURCE_OPTIONS = [
  { value: 'local', label: 'Local' },
  { value: 'git', label: 'Git' },
  { value: 'http', label: 'HTTP' },
  { value: 'mcp', label: 'MCP' },
  { value: 'builtin', label: 'Builtin' },
]

const activeView = ref<ActiveView>('list')
const selectedSkill = ref<SkillInfo | null>(null)

const viewTitle = computed(() => {
  if (activeView.value === 'versions' && selectedSkill.value) {
    return `Manifest 版本管理 - ${selectedSkill.value.display_name || selectedSkill.value.name}`
  }
  if (activeView.value === 'tools' && selectedSkill.value) {
    return `运行时 Tool 管理 - ${selectedSkill.value.display_name || selectedSkill.value.name}`
  }
  return '高级 SKILL 配置'
})

const list = ref<SkillInfo[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const error = ref('')
const keywordFilter = ref('')
const sourceFilter = ref('')
const statusFilter = ref('')

const loadList = async () => {
  loading.value = true
  error.value = ''
  try {
    const data = await listSkills(page.value, pageSize.value)
    list.value = data.list || []
    total.value = data.total || 0
  } catch (e: unknown) {
    error.value = (e as { message?: string }).message || '加载 SKILL 列表失败'
  } finally {
    loading.value = false
  }
}

const formatTime = (s?: string): string => {
  if (!s) return '-'
  return new Date(s).toLocaleString('zh-CN')
}

const formatJson = (json?: string): string => {
  if (!json) return ''
  try {
    return JSON.stringify(JSON.parse(json), null, 2)
  } catch {
    return json
  }
}

const shortText = (value?: string, max = 80): string => {
  if (!value) return '-'
  return value.length > max ? `${value.slice(0, max)}...` : value
}

const filteredSkills = computed(() => {
  const keyword = keywordFilter.value.trim().toLowerCase()
  return list.value.filter((item) => {
    const matchesKeyword = !keyword
      || item.name.toLowerCase().includes(keyword)
      || (item.display_name || '').toLowerCase().includes(keyword)
      || (item.description || '').toLowerCase().includes(keyword)
    const matchesSource = !sourceFilter.value || item.source_type === sourceFilter.value
    const matchesStatus = !statusFilter.value
      || (statusFilter.value === 'enabled' ? item.is_enabled : !item.is_enabled)
    return matchesKeyword && matchesSource && matchesStatus
  })
})

const sourceLabel = (source?: string) =>
  SOURCE_OPTIONS.find((item) => item.value === source)?.label || source || '-'

const resetListFilters = () => {
  keywordFilter.value = ''
  sourceFilter.value = ''
  statusFilter.value = ''
}

const validateJsonText = (value: string, label: string, allowEmpty = false): boolean => {
  if (!value.trim()) {
    if (allowEmpty) return true
    ElMessage.warning(`请输入 ${label}`)
    return false
  }
  try {
    JSON.parse(value)
    return true
  } catch (e: unknown) {
    ElMessage.error(`${label} 不是合法 JSON：${(e as { message?: string }).message || '解析失败'}`)
    return false
  }
}

const validateJsonField = (value: string, label: string, allowEmpty = false) => {
  if (validateJsonText(value, label, allowEmpty)) {
    ElMessage.success(`${label} 校验通过`)
  }
}

const formatJsonField = (value: string, label: string, allowEmpty = false): string => {
  if (!validateJsonText(value, label, allowEmpty)) return value
  if (!value.trim()) {
    ElMessage.success(`${label} 校验通过`)
    return value
  }
  ElMessage.success(`${label} 已格式化`)
  return JSON.stringify(JSON.parse(value), null, 2)
}

// ====== Skill create / edit ======

const dialogVisible = ref(false)
const dialogTitle = ref('')
const saving = ref(false)
const isEditing = ref(false)
const editingId = ref(0)
const dialogForm = reactive({
  name: '',
  display_name: '',
  description: '',
  source_type: 'local',
  source_uri: '',
  is_enabled: true,
})

const resetDialogForm = () => {
  dialogForm.name = ''
  dialogForm.display_name = ''
  dialogForm.description = ''
  dialogForm.source_type = 'local'
  dialogForm.source_uri = ''
  dialogForm.is_enabled = true
}

const openCreate = () => {
  isEditing.value = false
  editingId.value = 0
  dialogTitle.value = '新增底层 SKILL'
  resetDialogForm()
  dialogVisible.value = true
}

const openCreateWithSource = (sourceType: string) => {
  openCreate()
  dialogForm.source_type = sourceType
}

const openEdit = (row: SkillInfo) => {
  isEditing.value = true
  editingId.value = row.id
  dialogTitle.value = '编辑底层 SKILL 配置'
  dialogForm.name = row.name
  dialogForm.display_name = row.display_name || ''
  dialogForm.description = row.description || ''
  dialogForm.source_type = row.source_type || 'local'
  dialogForm.source_uri = row.source_uri || ''
  dialogForm.is_enabled = row.is_enabled
  dialogVisible.value = true
}

const saveSkill = async () => {
  if (!dialogForm.name.trim()) {
    ElMessage.warning('请输入 SKILL 名称')
    return
  }
  saving.value = true
  try {
    if (isEditing.value) {
      const payload: UpdateSkillPayload = {
        display_name: dialogForm.display_name,
        description: dialogForm.description,
        source_type: dialogForm.source_type,
        source_uri: dialogForm.source_uri,
        is_enabled: dialogForm.is_enabled,
        is_enabled_set: true,
      }
      await updateSkill(editingId.value, payload)
      ElMessage.success('底层 SKILL 配置已更新')
    } else {
      const payload: CreateSkillPayload = {
        name: dialogForm.name,
        display_name: dialogForm.display_name,
        description: dialogForm.description,
        source_type: dialogForm.source_type,
        source_uri: dialogForm.source_uri,
        is_enabled: dialogForm.is_enabled,
        is_enabled_set: true,
      }
      await createSkill(payload)
      ElMessage.success('底层 SKILL 已创建')
    }
    dialogVisible.value = false
    await loadList()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '保存失败')
  } finally {
    saving.value = false
  }
}

const handleToggleEnabled = async (row: SkillInfo) => {
  try {
    await updateSkill(row.id, { is_enabled: !row.is_enabled, is_enabled_set: true })
    ElMessage.success(row.is_enabled ? '已禁用' : '已启用')
    await loadList()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '操作失败')
  }
}

const detailVisible = ref(false)
const detailSkill = ref<SkillInfo | null>(null)
const openDetail = (row: SkillInfo) => {
  detailSkill.value = row
  detailVisible.value = true
}

const exampleVisible = ref(false)
const exampleManifest = `{
  "name": "resume_parser",
  "version": "1.0.0",
  "runtime_type": "tool",
  "instruction": "Parse resume text and return structured candidate profile.",
  "tools": [
    {
      "name": "parse_resume",
      "capability_key": "resume.parse",
      "description": "Extract candidate basics, skills and work history."
    }
  ]
}`

const openCreateFromExample = () => {
  exampleVisible.value = false
  openCreate()
}

// ====== Version management ======

const versions = ref<SkillVersionInfo[]>([])
const versionsLoading = ref(false)
const versionsError = ref('')

const loadVersions = async () => {
  if (!selectedSkill.value) return
  versionsLoading.value = true
  versionsError.value = ''
  try {
    const data = await listSkillVersions(selectedSkill.value.id)
    versions.value = data.list || []
  } catch (e: unknown) {
    versionsError.value = (e as { message?: string }).message || '加载版本列表失败'
  } finally {
    versionsLoading.value = false
  }
}

const navigateToVersions = async (row: SkillInfo) => {
  selectedSkill.value = row
  activeView.value = 'versions'
  await loadVersions()
}

const versionDialogVisible = ref(false)
const versionSaving = ref(false)
const versionForm = reactive({
  manifest_json: '{\n  \n}',
  activate: true,
})

const openCreateVersion = () => {
  versionForm.manifest_json = '{\n  \n}'
  versionForm.activate = true
  versionDialogVisible.value = true
}

const saveVersion = async () => {
  if (!selectedSkill.value) return
  if (!validateJsonText(versionForm.manifest_json, 'manifest_json')) return
  versionSaving.value = true
  try {
    await createSkillVersion(selectedSkill.value.id, {
      manifest_json: versionForm.manifest_json,
      activate: versionForm.activate,
    })
    ElMessage.success(versionForm.activate ? '版本已创建并激活' : '版本已创建')
    versionDialogVisible.value = false
    await loadVersions()
    await loadList()
    const latest = list.value.find((item) => item.id === selectedSkill.value?.id)
    if (latest) selectedSkill.value = latest
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '创建版本失败')
  } finally {
    versionSaving.value = false
  }
}

const activateVersion = async (row: SkillVersionInfo) => {
  if (!selectedSkill.value) return
  try {
    await ElMessageBox.confirm(
      `确认激活版本「${row.version}」？`,
      '激活版本',
      { confirmButtonText: '激活', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return
  }
  try {
    const data = await activateSkillVersion(selectedSkill.value.id, row.id)
    selectedSkill.value = data.skill
    ElMessage.success('版本已激活')
    await loadVersions()
    await loadList()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '激活失败')
  }
}

const versionDetailVisible = ref(false)
const versionDetail = ref<SkillVersionInfo | null>(null)
const openVersionDetail = (row: SkillVersionInfo) => {
  versionDetail.value = row
  versionDetailVisible.value = true
}

// ====== Tool management ======

const tools = ref<SkillToolInfo[]>([])
const toolsLoading = ref(false)
const toolsError = ref('')
const toolVersionId = ref<ToolVersionValue>('current')

const toolVersionOptions = computed(() => {
  const options: ToolVersionOption[] = [{ value: 'current', label: '当前版本' }, { value: 'all', label: '全部版本' }]
  return options.concat(versions.value.map((version) => ({
    value: version.id,
    label: version.version,
  })))
})

const visibleTools = computed(() => {
  if (toolVersionId.value === 'all') return tools.value
  const versionId = toolVersionId.value === 'current'
    ? selectedSkill.value?.current_version_id
    : toolVersionId.value
  if (!versionId) return []
  return tools.value.filter((tool) => tool.skill_version_id === versionId)
})

const loadTools = async () => {
  if (!selectedSkill.value) return
  toolsLoading.value = true
  toolsError.value = ''
  try {
    const data = await listSkillTools(selectedSkill.value.id, false)
    tools.value = data.list || []
  } catch (e: unknown) {
    toolsError.value = (e as { message?: string }).message || '加载 Tool 列表失败'
  } finally {
    toolsLoading.value = false
  }
}

const navigateToTools = async (row: SkillInfo) => {
  selectedSkill.value = row
  activeView.value = 'tools'
  toolVersionId.value = 'current'
  await Promise.all([loadVersions(), loadTools()])
}

const toolDialogVisible = ref(false)
const toolSaving = ref(false)
const editingToolId = ref(0)
const toolForm = reactive({
  tool_name: '',
  capability_key: '',
  runtime_tool_name: '',
  description: '',
  runtime_config_json: '',
  is_enabled: true,
})

const openEditTool = (row: SkillToolInfo) => {
  editingToolId.value = row.id
  toolForm.tool_name = row.tool_name
  toolForm.capability_key = row.capability_key
  toolForm.runtime_tool_name = row.runtime_tool_name
  toolForm.description = row.description || ''
  toolForm.runtime_config_json = formatJson(row.runtime_config_json || '{}')
  toolForm.is_enabled = row.is_enabled
  toolDialogVisible.value = true
}

const saveTool = async () => {
  if (!selectedSkill.value) return
  if (!validateJsonText(toolForm.runtime_config_json, 'runtime_config_json', true)) return
  toolSaving.value = true
  try {
    const payload: UpdateSkillToolPayload = {
      description: toolForm.description,
      runtime_config_json: toolForm.runtime_config_json || '{}',
      is_enabled: toolForm.is_enabled,
      is_enabled_set: true,
    }
    await updateSkillTool(selectedSkill.value.id, editingToolId.value, payload)
    ElMessage.success('Tool 已更新')
    toolDialogVisible.value = false
    await loadTools()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '保存 Tool 失败')
  } finally {
    toolSaving.value = false
  }
}

const toolTagType = (tool: SkillToolInfo): TagType => {
  if (!tool.is_enabled) return 'info'
  if (tool.skill_version_id === selectedSkill.value?.current_version_id) return 'success'
  return 'primary'
}

const goBackToList = () => {
  activeView.value = 'list'
  selectedSkill.value = null
  versions.value = []
  tools.value = []
  toolVersionId.value = 'current'
}

onMounted(() => {
  loadList()
})
</script>

<template>
  <div class="skill-manage-view">
    <div class="workspace-surface">
      <div class="workspace-surface__header">
        <div class="workspace-surface__header-copy">
          <p class="page-kicker">System Admin · Skill Registry</p>
          <h2 class="page-title">{{ viewTitle }}</h2>
          <p class="page-desc">
            用于系统管理员维护底层 Skill Registry、Manifest 版本与运行时 Tool，不作为普通 HR 的业务能力入口。
          </p>
        </div>
        <div class="workspace-surface__header-actions">
          <el-button v-if="activeView !== 'list'" text :icon="Sort" @click="goBackToList">
            返回列表
          </el-button>
          <el-button v-if="canManage && activeView === 'list'" type="primary" :icon="Plus" @click="openCreate">
            新增底层 SKILL
          </el-button>
        </div>
      </div>

      <div class="workspace-surface__divider"></div>

      <template v-if="activeView === 'list'">
        <el-card v-if="list.length === 0 && !loading" class="empty-card" shadow="never">
          <el-empty :description="error || '暂无底层 SKILL 配置'">
            <div class="empty-copy">
              当前还没有系统级 Skill Registry 条目。请由系统管理员添加底层 SKILL，并通过 Manifest 版本声明 Runtime Config、Tool 暴露与运行时绑定。
            </div>
            <div class="empty-actions">
              <el-button v-if="canManage" type="primary" :icon="Plus" @click="openCreate">新增底层 SKILL</el-button>
              <el-button v-if="canManage" @click="openCreateWithSource('git')">从 Git 导入</el-button>
              <el-button text :icon="Document" @click="exampleVisible = true">查看示例</el-button>
            </div>
          </el-empty>
        </el-card>

        <template v-else>
          <div class="workspace-surface__toolbar">
            <div class="workspace-surface__filters">
              <el-input
                v-model="keywordFilter"
                style="width: 260px"
                :prefix-icon="Search"
                clearable
                placeholder="搜索底层 SKILL 名称 / 标识"
              />
              <el-select v-model="sourceFilter" placeholder="全部来源" clearable style="width: 140px">
                <el-option
                  v-for="opt in SOURCE_OPTIONS"
                  :key="opt.value"
                  :value="opt.value"
                  :label="opt.label"
                />
              </el-select>
              <el-select v-model="statusFilter" placeholder="全部状态" clearable style="width: 140px">
                <el-option value="enabled" label="已启用" />
                <el-option value="disabled" label="已禁用" />
              </el-select>
            </div>
            <div class="workspace-surface__actions">
              <el-button @click="resetListFilters">重置</el-button>
              <el-button :icon="Refresh" @click="loadList">刷新</el-button>
            </div>
          </div>

          <div class="workspace-surface__body">
            <el-table
              v-loading="loading"
              :data="filteredSkills"
              stripe
              style="width: 100%"
              :empty-text="error || '暂无匹配的底层 SKILL 配置'"
              @row-click="openDetail"
            >
              <el-table-column label="Skill 信息" min-width="240">
                <template #default="{ row }: { row: SkillInfo }">
                  <div class="entity-cell">
                    <div class="entity-title">{{ row.display_name || row.name }}</div>
                    <div class="entity-sub">{{ row.name }}</div>
                    <div v-if="row.description" class="entity-desc">{{ row.description }}</div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="来源" min-width="220" show-overflow-tooltip>
                <template #default="{ row }: { row: SkillInfo }">
                  <el-tag size="small" effect="plain">{{ sourceLabel(row.source_type) }}</el-tag>
                  <span class="source-uri">{{ row.source_uri || '-' }}</span>
                </template>
              </el-table-column>
              <el-table-column label="当前版本" width="112">
                <template #default="{ row }: { row: SkillInfo }">
                  <el-tag v-if="row.current_version_id" size="small" type="success">#{{ row.current_version_id }}</el-tag>
                  <el-tag v-else size="small" type="info">未设置</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="Tools / 版本入口" width="174">
                <template #default="{ row }: { row: SkillInfo }">
                  <div class="inline-actions">
                    <el-button size="small" :icon="Document" @click.stop="navigateToVersions(row)">版本</el-button>
                    <el-button size="small" :icon="Tools" @click.stop="navigateToTools(row)">Tools</el-button>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="90">
                <template #default="{ row }: { row: SkillInfo }">
                  <el-switch
                    :model-value="row.is_enabled"
                    size="small"
                    :disabled="!canManage"
                    @click.stop="handleToggleEnabled(row)"
                  />
                </template>
              </el-table-column>
              <el-table-column label="更新时间" width="170">
                <template #default="{ row }: { row: SkillInfo }">
                  {{ formatTime(row.updated_at) }}
                </template>
              </el-table-column>
              <el-table-column label="操作" width="150" fixed="right">
                <template #default="{ row }: { row: SkillInfo }">
                  <el-button size="small" :icon="View" @click.stop="openDetail(row)">详情</el-button>
                  <el-button v-if="canManage" size="small" :icon="Edit" @click.stop="openEdit(row)">编辑</el-button>
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
      </template>

      <template v-else-if="activeView === 'versions'">
        <el-card class="table-card" shadow="never">
          <div class="filter-toolbar">
            <el-button v-if="canManage" type="primary" :icon="Plus" @click="openCreateVersion">创建版本</el-button>
            <div class="filter-actions">
              <el-button :icon="Refresh" @click="loadVersions">刷新</el-button>
            </div>
          </div>

          <el-table
            v-loading="versionsLoading"
            :data="versions"
            stripe
            style="width: 100%"
            :empty-text="versionsError || '暂无版本'"
          >
            <el-table-column prop="version" label="版本" width="140" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }: { row: SkillVersionInfo }">
                <el-tag v-if="row.id === selectedSkill?.current_version_id" type="success" size="small">当前</el-tag>
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column prop="runtime_type" label="运行类型" width="120" />
            <el-table-column label="Instruction" min-width="220" show-overflow-tooltip>
              <template #default="{ row }: { row: SkillVersionInfo }">
                {{ shortText(row.instruction) }}
              </template>
            </el-table-column>
            <el-table-column label="创建时间" width="170">
              <template #default="{ row }: { row: SkillVersionInfo }">
                {{ formatTime(row.created_at) }}
              </template>
            </el-table-column>
            <el-table-column label="操作" width="180" fixed="right">
              <template #default="{ row }: { row: SkillVersionInfo }">
                <el-button size="small" :icon="Document" @click="openVersionDetail(row)">详情</el-button>
                <el-button
                  size="small"
                  type="primary"
                  :icon="Check"
                  :disabled="row.id === selectedSkill?.current_version_id"
                  @click="activateVersion(row)"
                >
                  激活
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </template>

      <template v-else>
      <el-card class="table-card" shadow="never">
        <div class="filter-toolbar">
          <el-select v-model="toolVersionId" style="width: 180px">
            <el-option
              v-for="opt in toolVersionOptions"
              :key="String(opt.value)"
              :value="opt.value"
              :label="opt.label"
            />
          </el-select>
          <div class="filter-actions">
            <el-button :icon="Refresh" @click="loadTools">刷新</el-button>
          </div>
        </div>

        <el-table
          v-loading="toolsLoading"
          :data="visibleTools"
          stripe
          style="width: 100%"
          :empty-text="toolsError || '暂无 Tool'"
        >
          <el-table-column prop="tool_name" label="Tool" min-width="160" />
          <el-table-column prop="capability_key" label="Capability Key" min-width="180" show-overflow-tooltip />
          <el-table-column prop="runtime_tool_name" label="Runtime Tool" min-width="160" show-overflow-tooltip />
          <el-table-column label="Schema" min-width="180" show-overflow-tooltip>
            <template #default="{ row }: { row: SkillToolInfo }">
              <code>{{ shortText(formatJson(row.input_schema_json), 120) }}</code>
            </template>
          </el-table-column>
          <el-table-column label="Runtime Config" min-width="180" show-overflow-tooltip>
            <template #default="{ row }: { row: SkillToolInfo }">
              <code>{{ shortText(formatJson(row.runtime_config_json), 120) }}</code>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="110">
            <template #default="{ row }: { row: SkillToolInfo }">
              <el-tag :type="toolTagType(row)" size="small">
                {{ row.is_enabled ? '启用' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="更新时间" width="170">
            <template #default="{ row }: { row: SkillToolInfo }">
              {{ formatTime(row.updated_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="90" fixed="right">
            <template #default="{ row }: { row: SkillToolInfo }">
              <el-button v-if="canManage" size="small" :icon="Edit" @click="openEditTool(row)">编辑</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </template>
    </div>

    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="640px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <el-form :model="dialogForm" label-width="120px">
        <el-form-item label="SKILL 标识" required>
          <el-input v-model="dialogForm.name" placeholder="英文标识，例如：resume_parser" :disabled="isEditing" />
        </el-form-item>
        <el-form-item label="显示名称">
          <el-input v-model="dialogForm.display_name" placeholder="例如：Resume Parser Runtime Skill" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="dialogForm.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="Source Type">
          <el-select v-model="dialogForm.source_type" style="width: 100%" filterable allow-create>
            <el-option
              v-for="opt in SOURCE_OPTIONS"
              :key="opt.value"
              :value="opt.value"
              :label="opt.label"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="Source URI">
          <el-input v-model="dialogForm.source_uri" placeholder="本地路径、Git URL 或 HTTP 地址" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="dialogForm.is_enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveSkill">
          {{ isEditing ? '保存' : '创建' }}
        </el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="versionDialogVisible"
      title="创建 Manifest 版本"
      width="820px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <el-form :model="versionForm" label-width="120px">
        <el-form-item label="Manifest JSON" required>
          <el-input
            v-model="versionForm.manifest_json"
            type="textarea"
            :rows="18"
            class="json-editor"
            placeholder="{ ... }"
          />
          <div class="form-actions">
            <el-button size="small" @click="validateJsonField(versionForm.manifest_json, 'manifest_json')">
              校验 JSON
            </el-button>
            <el-button
              size="small"
              @click="versionForm.manifest_json = formatJsonField(versionForm.manifest_json, 'manifest_json')"
            >
              格式化
            </el-button>
          </div>
        </el-form-item>
        <el-form-item label="创建后激活">
          <el-switch v-model="versionForm.activate" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="versionDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="versionSaving" @click="saveVersion">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="versionDetailVisible" title="版本详情" width="820px" destroy-on-close>
      <template v-if="versionDetail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="版本">{{ versionDetail.version }}</el-descriptions-item>
          <el-descriptions-item label="运行类型">{{ versionDetail.runtime_type || '-' }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatTime(versionDetail.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="版本 ID">#{{ versionDetail.id }}</el-descriptions-item>
        </el-descriptions>
        <h3 class="detail-title">Manifest JSON</h3>
        <pre class="json-preview">{{ formatJson(versionDetail.manifest_json) }}</pre>
        <h3 class="detail-title">Input Schema</h3>
        <pre class="json-preview">{{ formatJson(versionDetail.input_schema_json) || '-' }}</pre>
        <h3 class="detail-title">Output Schema</h3>
        <pre class="json-preview">{{ formatJson(versionDetail.output_schema_json) || '-' }}</pre>
      </template>
    </el-dialog>

    <el-dialog
      v-model="toolDialogVisible"
      title="编辑 Tool"
      width="760px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <el-form :model="toolForm" label-width="150px">
        <el-form-item label="Tool">
          <el-input v-model="toolForm.tool_name" disabled />
        </el-form-item>
        <el-form-item label="Capability Key">
          <el-input v-model="toolForm.capability_key" disabled />
        </el-form-item>
        <el-form-item label="Runtime Tool">
          <el-input v-model="toolForm.runtime_tool_name" disabled />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="toolForm.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="Runtime Config JSON">
          <el-input v-model="toolForm.runtime_config_json" type="textarea" :rows="12" class="json-editor" />
          <div class="form-actions">
            <el-button size="small" @click="validateJsonField(toolForm.runtime_config_json, 'runtime_config_json', true)">
              校验 JSON
            </el-button>
            <el-button
              size="small"
              @click="toolForm.runtime_config_json = formatJsonField(toolForm.runtime_config_json, 'runtime_config_json', true)"
            >
              格式化
            </el-button>
          </div>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="toolForm.is_enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="toolDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="toolSaving" @click="saveTool">保存</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" title="底层 SKILL 配置详情" size="560px" :close-on-click-modal="true" destroy-on-close>
      <template v-if="detailSkill">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="显示名称">{{ detailSkill.display_name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="标识">{{ detailSkill.name }}</el-descriptions-item>
          <el-descriptions-item label="描述">{{ detailSkill.description || '-' }}</el-descriptions-item>
          <el-descriptions-item label="来源">
            <el-tag size="small" effect="plain">{{ sourceLabel(detailSkill.source_type) }}</el-tag>
            <span class="source-uri">{{ detailSkill.source_uri || '-' }}</span>
          </el-descriptions-item>
          <el-descriptions-item label="当前版本">
            <el-tag v-if="detailSkill.current_version_id" type="success" size="small">
              #{{ detailSkill.current_version_id }}
            </el-tag>
            <el-tag v-else type="info" size="small">未设置</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="detailSkill.is_enabled ? 'success' : 'info'" size="small">
              {{ detailSkill.is_enabled ? '启用' : '禁用' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatTime(detailSkill.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="更新时间">{{ formatTime(detailSkill.updated_at) }}</el-descriptions-item>
        </el-descriptions>
        <div class="drawer-actions">
          <el-button :icon="Document" @click="navigateToVersions(detailSkill)">Manifest 版本</el-button>
          <el-button :icon="Tools" @click="navigateToTools(detailSkill)">运行时 Tools</el-button>
          <el-button type="primary" :icon="Edit" @click="openEdit(detailSkill)">编辑</el-button>
        </div>
      </template>
    </el-drawer>

    <el-dialog v-model="exampleVisible" title="示例 Manifest" width="720px" destroy-on-close>
      <pre class="json-preview">{{ exampleManifest }}</pre>
      <template #footer>
        <el-button @click="exampleVisible = false">关闭</el-button>
        <el-button type="primary" @click="openCreateFromExample">基于示例新增底层 SKILL</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.skill-manage-view {
  height: 100%;
  min-height: 0;
  padding-bottom: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.page-header,
.page-actions,
.filter-toolbar,
.filter-actions,
.empty-actions,
.inline-actions,
.drawer-actions,
.form-actions {
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

.table-card,
.empty-card {
  border-radius: 8px;
}

.table-card {
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
.entity-desc {
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

.inline-actions,
.form-actions {
  gap: 8px;
}

.empty-card {
  padding: 18px;
}

.empty-copy {
  max-width: 520px;
  margin: 0 auto 16px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}

.empty-actions {
  justify-content: center;
  gap: 10px;
  flex-wrap: wrap;
}

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
  flex-shrink: 0;
}

.source-uri {
  margin-left: 8px;
  color: var(--el-text-color-regular);
}

.drawer-actions {
  justify-content: flex-end;
  gap: 8px;
  margin-top: 18px;
  flex-wrap: wrap;
}

.json-editor :deep(textarea),
.json-preview,
code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
}

.form-actions {
  width: 100%;
  margin-top: 8px;
}

.detail-title {
  margin: 16px 0 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.json-preview {
  max-height: 300px;
  overflow: auto;
  margin: 0;
  padding: 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  background: var(--el-fill-color-light);
  font-size: 12px;
  line-height: 1.5;
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
