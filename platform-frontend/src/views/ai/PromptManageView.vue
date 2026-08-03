<script setup lang="ts">
import { t } from '@shared/i18n'
import { onMounted, reactive, ref, computed } from 'vue'
import { formatShanghaiDateTime } from '@shared/utils/format'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown, Delete, Edit, MoreFilled, Plus, Search, View, Back } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import { PLATFORM_PERMISSIONS } from '@/permissions'
import { PagePanel } from '@/components/admin-console'
import {
  listPromptTemplates,
  createPromptTemplate,
  updatePromptTemplate,
  deletePromptTemplate,
  getPromptVersionHistory,
  rollbackPromptVersion,
} from '@/api/prompt'
import type {
  PromptTemplate,
  CreatePromptPayload,
  UpdatePromptPayload,
  PromptVersion,
} from '@shared/types/prompt'
import {
  PROMPT_AGENT_TYPE_GROUP_LABEL,
  conversationPromptAgentTypes,
  findPromptAgentTypeOption,
  normalizePromptAgentType,
  promptAgentTypeKind,
  promptAgentTypeKindLabel,
  promptAgentTypeLabel,
  structuredTaskPromptAgentTypes,
} from '@shared/constants/promptAgentTypes'

// ====== Auth state ======

const auth = useAuthStore()
const currentUserId = computed(() => auth.user?.user_id || 0)
const canManage = computed(() => auth.can(PLATFORM_PERMISSIONS.AI_CONFIG_MANAGE))

// ====== List State ======

const templateList = ref<PromptTemplate[]>([])
const templateTotal = ref(0)
const templatePage = ref(1)
const templatePageSize = ref(20)
const templateLoading = ref(false)
const templateError = ref('')
const agentTypeFilter = ref('')
const keywordFilter = ref('')
const statusFilter = ref('')

const loadList = async () => {
  templateLoading.value = true
  templateError.value = ''
  try {
    const data = await listPromptTemplates(
      templatePage.value,
      templatePageSize.value,
      agentTypeFilter.value || undefined,
    )
    templateList.value = data.list || []
    templateTotal.value = data.total || 0
  } catch (e: unknown) {
    templateError.value = (e as { message?: string }).message || '加载 Prompt 模板列表失败'
  } finally {
    templateLoading.value = false
  }
}

// ====== Helpers ======

const agentTypeLabel = promptAgentTypeLabel

const promptRoleLabel = (r: string): string => {
  const map: Record<string, string> = {
    system: '系统',
    user: '用户',
  }
  return map[r] || r || '-'
}

const formatTime = (s?: string): string => {
  return formatShanghaiDateTime(s)
}

// Extract {{variable}} placeholders from content text.
const extractVariables = (content: string): string[] => {
  const regex = /\{\{(\w+)\}\}/g
  const vars: string[] = []
  let match: RegExpExecArray | null
  while ((match = regex.exec(content)) !== null) {
    if (!vars.includes(match[1])) {
      vars.push(match[1])
    }
  }
  return vars
}

const filteredTemplates = computed(() => {
  const keyword = keywordFilter.value.trim().toLowerCase()
  return templateList.value.filter((item) => {
    const matchesKeyword = !keyword
      || item.name.toLowerCase().includes(keyword)
      || item.content.toLowerCase().includes(keyword)
    const matchesStatus = !statusFilter.value
      || (statusFilter.value === 'active' ? item.is_active : !item.is_active)
    return matchesKeyword && matchesStatus
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
  content: '',
  agent_type: 'hr_recruiting_agent',
  prompt_role: 'system',
  is_active: true,
  change_note: '',
})

const selectedAgentTypeOption = computed(() => findPromptAgentTypeOption(dialogForm.agent_type))
const isStructuredTaskType = computed(
  () => promptAgentTypeKind(dialogForm.agent_type) === 'structured_task',
)

const handleAgentTypeChange = (value: string) => {
  // Structured task pipelines only load active system prompts.
  if (promptAgentTypeKind(value) === 'structured_task') {
    dialogForm.prompt_role = 'system'
  }
}

// Variables extracted from the current content text.
const extractedVars = computed(() => extractVariables(dialogForm.content))

const resetDialogForm = () => {
  dialogForm.name = ''
  dialogForm.content = ''
  dialogForm.agent_type = 'hr_recruiting_agent'
  dialogForm.prompt_role = 'system'
  dialogForm.is_active = true
  dialogForm.change_note = ''
}

const openCreate = () => {
  isEditing.value = false
  editingId.value = 0
  dialogTitle.value = '新增 Prompt 模板'
  resetDialogForm()
  dialogVisible.value = true
}

const openEdit = (row: PromptTemplate) => {
  isEditing.value = true
  editingId.value = row.id
  dialogTitle.value = '编辑 Prompt 模板'
  dialogForm.name = row.name
  dialogForm.content = row.content
  dialogForm.agent_type = normalizePromptAgentType(row.agent_type)
  dialogForm.prompt_role = row.prompt_role
  dialogForm.is_active = row.is_active
  dialogForm.change_note = ''
  dialogVisible.value = true
}

/** Whether another enabled template covers the same type/role scope (legacy hr_agent shares HR scope). */
const hasOtherActivePrompt = (row: Pick<PromptTemplate, 'id' | 'agent_type' | 'prompt_role'>): boolean => {
  const scope = (agentType: string): string[] => {
    const t = agentType.trim().toLowerCase()
    if (t === 'hr_recruiting_agent' || t === 'hr_agent') return ['hr_recruiting_agent', 'hr_agent']
    return [t]
  }
  const rowScope = new Set(scope(row.agent_type))
  const role = row.prompt_role.trim().toLowerCase()
  return templateList.value.some((item) => {
    if (item.id === row.id || !item.is_active) return false
    if (item.prompt_role.trim().toLowerCase() !== role) return false
    return scope(item.agent_type).some((t) => rowScope.has(t))
  })
}

const save = async () => {
  if (!dialogForm.name) {
    ElMessage.warning(t('common.invalid_request'))
    return
  }
  if (!dialogForm.content) {
    ElMessage.warning(t('common.invalid_request'))
    return
  }
  if (
    isEditing.value
    && !dialogForm.is_active
    && !hasOtherActivePrompt({
      id: editingId.value,
      agent_type: dialogForm.agent_type,
      prompt_role: dialogForm.prompt_role,
    })
  ) {
    // If the row is already inactive, allow saving other fields without re-checking.
    const current = templateList.value.find((item) => item.id === editingId.value)
    if (current?.is_active) {
      ElMessage.warning(t('common.invalid_request'))
      return
    }
  }
  saving.value = true
  try {
    const variablesJson = JSON.stringify(extractedVars.value)
    if (isEditing.value) {
      const payload: UpdatePromptPayload = {
        name: dialogForm.name,
        content: dialogForm.content,
        variables_json: variablesJson,
        is_active: dialogForm.is_active,
        is_active_set: true,
        updated_by: currentUserId.value,
        change_note: dialogForm.change_note || undefined,
      }
      await updatePromptTemplate(editingId.value, payload)
      ElMessage.success(t('common.success'))
    } else {
      const payload: CreatePromptPayload = {
        name: dialogForm.name,
        content: dialogForm.content,
        variables_json: variablesJson,
        agent_type: dialogForm.agent_type,
        prompt_role: dialogForm.prompt_role,
        created_by: currentUserId.value,
      }
      await createPromptTemplate(payload)
      ElMessage.success(t('common.success'))
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

const handleDelete = async (row: PromptTemplate) => {
  try {
    await ElMessageBox.confirm(
      `确认删除 Prompt 模板「${row.name}」？此操作不可撤销。`,
      '删除确认',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return
  }
  try {
    await deletePromptTemplate(row.id)
    ElMessage.success(t('common.success'))
    await loadList()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '删除失败')
  }
}

// ====== Toggle active status ======

const handleToggleActive = async (row: PromptTemplate) => {
  if (row.is_active && !hasOtherActivePrompt(row)) {
    ElMessage.warning(t('common.invalid_request'))
    return
  }
  try {
    await updatePromptTemplate(row.id, {
      is_active: !row.is_active,
      is_active_set: true,
      updated_by: currentUserId.value,
      change_note: row.is_active ? '禁用模板' : '启用模板',
    })
    ElMessage.success(row.is_active ? '已禁用' : '已启用')
    await loadList()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '操作失败')
  }
}

// ====== Version History Dialog ======

const versionDialogVisible = ref(false)
const versionList = ref<PromptVersion[]>([])
const versionTotal = ref(0)
const versionPage = ref(1)
const versionPageSize = ref(20)
const versionLoading = ref(false)
const currentTemplateForVersion = ref<PromptTemplate | null>(null)

const openVersionHistory = async (row: PromptTemplate) => {
  currentTemplateForVersion.value = row
  versionPage.value = 1
  versionDialogVisible.value = true
  await loadVersions()
}

const loadVersions = async () => {
  if (!currentTemplateForVersion.value) return
  versionLoading.value = true
  try {
    const data = await getPromptVersionHistory(
      currentTemplateForVersion.value.id,
      versionPage.value,
      versionPageSize.value,
    )
    versionList.value = data.list || []
    versionTotal.value = data.total || 0
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '加载版本历史失败')
  } finally {
    versionLoading.value = false
  }
}

// ====== Version Content Preview (readonly) ======

const contentPreviewVisible = ref(false)
const contentPreviewTitle = ref('')
const contentPreviewContent = ref('')

const viewVersionContent = (v: PromptVersion) => {
  contentPreviewContent.value = v.content
  contentPreviewTitle.value = `版本 ${v.version} 内容（只读）`
  contentPreviewVisible.value = true
}

// ====== Rollback ======

const handleRollback = async (v: PromptVersion) => {
  if (!currentTemplateForVersion.value) return
  try {
    await ElMessageBox.confirm(
      `确认将「${currentTemplateForVersion.value.name}」回滚到版本 ${v.version}？\n\n回滚后当前内容将被替换为历史版本内容，版本号将自动递增。`,
      '回滚确认',
      {
        confirmButtonText: '确认回滚',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger',
      },
    )
  } catch {
    return
  }
  try {
    await rollbackPromptVersion(currentTemplateForVersion.value.id, {
      version: v.version,
      updated_by: currentUserId.value,
      change_note: `回滚到版本 ${v.version}`,
    })
    ElMessage.success(t('common.success'))
    versionDialogVisible.value = false
    await loadList()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '回滚失败')
  }
}

// ====== Init ======

onMounted(() => {
  loadList()
})
</script>

<template>
  <div class="console-page console-page--fill prompt-manage-view">
    <PagePanel>
      <div class="workspace-surface">

      <div class="workspace-surface__toolbar">
        <div class="workspace-surface__filters">
          <el-input v-model="keywordFilter" :prefix-icon="Search" clearable placeholder="搜索模板名称 / 内容" style="width: 260px" />
          <el-select
            v-model="agentTypeFilter"
            placeholder="全部绑定类型"
            clearable
            style="width: 220px"
            @change="() => { templatePage = 1; loadList() }"
          >
            <el-option value="" label="全部绑定类型" />
            <el-option-group :label="PROMPT_AGENT_TYPE_GROUP_LABEL.conversation">
              <el-option
                v-for="item in conversationPromptAgentTypes"
                :key="item.value"
                :value="item.value"
                :label="item.label"
              />
            </el-option-group>
            <el-option-group :label="PROMPT_AGENT_TYPE_GROUP_LABEL.structured_task">
              <el-option
                v-for="item in structuredTaskPromptAgentTypes"
                :key="item.value"
                :value="item.value"
                :label="item.label"
              />
            </el-option-group>
          </el-select>
          <el-select v-model="statusFilter" placeholder="全部状态" clearable style="width: 140px">
            <el-option value="active" label="启用" />
            <el-option value="inactive" label="禁用" />
          </el-select>
        </div>
        <div class="workspace-surface__actions">
          <el-button v-if="canManage" type="primary" :icon="Plus" @click="openCreate">新增模板</el-button>
        </div>
      </div>

      <div class="workspace-surface__body">
        <el-table
          v-loading="templateLoading"
          :data="filteredTemplates"
          class="console-table"
          stripe
          style="width: 100%"
          :empty-text="templateError || '暂无数据'"
        >
          <el-table-column label="模板信息" min-width="240">
            <template #default="{ row }: { row: PromptTemplate }">
              <div class="console-entity">
                <div class="console-entity__name">{{ row.name }}</div>
                <div class="console-entity__meta">变量 {{ extractVariables(row.content).length }} 个 / v{{ row.version }}</div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="角色" width="90">
            <template #default="{ row }: { row: PromptTemplate }">
              {{ promptRoleLabel(row.prompt_role) }}
            </template>
          </el-table-column>
          <el-table-column label="绑定类型" min-width="200">
            <template #default="{ row }: { row: PromptTemplate }">
              <div class="prompt-type-cell">
                <span class="prompt-type-cell__label">{{ agentTypeLabel(row.agent_type) }}</span>
                <el-tag
                  size="small"
                  effect="plain"
                  :type="promptAgentTypeKind(row.agent_type) === 'structured_task' ? 'warning' : 'info'"
                >
                  {{ promptAgentTypeKindLabel(row.agent_type) }}
                </el-tag>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="version" label="当前版本" width="100">
            <template #default="{ row }: { row: PromptTemplate }">
              <el-tag size="small" type="primary">v{{ row.version }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="80">
            <template #default="{ row }: { row: PromptTemplate }">
              <el-tag :type="row.is_active ? 'success' : 'info'" size="small">
                {{ row.is_active ? '启用' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="更新时间" width="170">
            <template #default="{ row }: { row: PromptTemplate }">
              {{ formatTime(row.updated_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="220" fixed="right">
            <template #default="{ row }: { row: PromptTemplate }">
              <el-button v-if="canManage" size="small" :icon="Edit" @click="openEdit(row)">
                编辑
              </el-button>
              <el-button size="small" :icon="View" @click="openVersionHistory(row)">
                版本历史
              </el-button>
              <el-dropdown v-if="canManage" trigger="click" @command="(cmd: string) => { if (cmd === 'toggle') handleToggleActive(row); if (cmd === 'delete') handleDelete(row) }">
                <el-button size="small">更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="toggle">{{ row.is_active ? '禁用' : '启用' }}</el-dropdown-item>
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
          v-model:current-page="templatePage"
          v-model:page-size="templatePageSize"
          :total="templateTotal"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @current-change="loadList"
          @size-change="(s: number) => { templatePageSize = s; templatePage = 1; loadList() }"
        />
      </div>
    </div>
    </PagePanel>

    <!-- ── Edit / Create Dialog ────────────────────────────────────── -->
    <el-drawer
      v-model="dialogVisible"
      :title="dialogTitle"
      size="680px"
      :close-on-click-modal="true"
      destroy-on-close
    >
      <el-form :model="dialogForm" label-width="120px">
        <el-form-item label="模板名称" required>
          <el-input v-model="dialogForm.name" placeholder="例如：HR 面试助手 System Prompt" />
        </el-form-item>
        <el-form-item label="绑定类型" required>
          <el-select
            v-model="dialogForm.agent_type"
            style="width: 100%"
            :disabled="isEditing"
            placeholder="选择用途：对话助手 或 系统内置任务"
            @change="handleAgentTypeChange"
          >
            <el-option-group :label="PROMPT_AGENT_TYPE_GROUP_LABEL.conversation">
              <el-option
                v-for="item in conversationPromptAgentTypes"
                :key="item.value"
                :value="item.value"
                :label="item.label"
              />
            </el-option-group>
            <el-option-group :label="PROMPT_AGENT_TYPE_GROUP_LABEL.structured_task">
              <el-option
                v-for="item in structuredTaskPromptAgentTypes"
                :key="item.value"
                :value="item.value"
                :label="item.label"
              />
            </el-option-group>
          </el-select>
          <p v-if="selectedAgentTypeOption" class="field-hint">
            {{ selectedAgentTypeOption.description }}
          </p>
          <p v-if="isStructuredTaskType" class="field-hint field-hint--task">
            系统内置任务提示词保存在数据库中，由后台按类型自动加载；不必在 Agent 管理里再建一个 Agent。
          </p>
          <p v-if="isEditing" class="field-hint">编辑时不可更改绑定类型（避免影响已上线任务）。如需换类型请新建模板。</p>
        </el-form-item>
        <el-form-item label="角色" required>
          <el-select v-model="dialogForm.prompt_role" style="width: 100%" :disabled="isEditing && isStructuredTaskType">
            <el-option value="system" label="系统（System）" />
            <el-option value="user" label="用户（User）" />
          </el-select>
          <p v-if="isStructuredTaskType" class="field-hint">
            系统内置任务请使用「系统」角色；系统只会加载启用中的 system 提示词。
          </p>
        </el-form-item>
        <el-form-item label="Prompt 内容" required>
          <div class="content-editor-wrap">
            <el-input
              v-model="dialogForm.content"
              type="textarea"
              :rows="12"
              placeholder="输入 Prompt 内容，使用 {{variable}} 插入变量占位符"
              class="prompt-textarea"
            />
          </div>
        </el-form-item>
        <el-form-item label="变量列表">
          <div class="variable-tags">
            <el-tag
              v-for="v in extractedVars"
              :key="v"
              type="warning"
              size="small"
              class="variable-tag"
            >
              <code>\{{ v }}</code>
            </el-tag>
            <span v-if="extractedVars.length === 0" class="no-vars">未检测到变量</span>
          </div>
        </el-form-item>
        <el-form-item v-if="isEditing" label="变更备注">
          <el-input
            v-model="dialogForm.change_note"
            placeholder="描述本次修改原因（选填）"
          />
        </el-form-item>
        <el-form-item v-if="isEditing" label="启用">
          <el-switch v-model="dialogForm.is_active" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">
          {{ isEditing ? '保存' : '创建' }}
        </el-button>
      </template>
    </el-drawer>

    <!-- ── Version History Dialog ──────────────────────────────────── -->
    <el-dialog
      v-model="versionDialogVisible"
      title="版本历史"
      width="720px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <template v-if="currentTemplateForVersion">
        <div class="version-info-bar">
          <span><strong>模板：</strong>{{ currentTemplateForVersion.name }}</span>
          <span><strong>当前版本：</strong>{{ currentTemplateForVersion.version }}</span>
        </div>
      </template>
      <el-table
        v-loading="versionLoading"
        :data="versionList"
        stripe
        border
        style="width: 100%"
      >
        <el-table-column prop="version" label="版本号" width="80" />
        <el-table-column prop="change_note" label="变更备注" min-width="160" show-overflow-tooltip>
          <template #default="{ row }: { row: PromptVersion }">
            {{ row.change_note || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="变更人" width="120">
          <template #default="{ row }: { row: PromptVersion }">
            {{ row.changed_by || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="变更时间" width="170">
          <template #default="{ row }: { row: PromptVersion }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }: { row: PromptVersion }">
            <el-button size="small" :icon="View" @click="viewVersionContent(row)">
              查看
            </el-button>
            <el-button v-if="canManage"
              size="small"
              type="danger"
              :icon="Back"
              @click="handleRollback(row)"
            >
              回滚
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-wrap">
        <el-pagination
          v-model:current-page="versionPage"
          v-model:page-size="versionPageSize"
          :total="versionTotal"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @current-change="loadVersions"
          @size-change="(s: number) => { versionPageSize = s; versionPage = 1; loadVersions() }"
        />
      </div>
    </el-dialog>

    <!-- ── Content Preview Dialog (read-only) ──────────────────────── -->
    <el-dialog
      v-model="contentPreviewVisible"
      :title="contentPreviewTitle"
      width="680px"
      :close-on-click-modal="false"
    >
      <pre class="content-preview">{{ contentPreviewContent }}</pre>
      <template #footer>
        <el-button @click="contentPreviewVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.prompt-manage-view {
  padding-bottom: 24px;
}

.console-page--fill .page-panel {
  flex: 1;
  min-height: 0;
}

.prompt-type-cell {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
}

.prompt-type-cell__label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  line-height: 1.3;
}

.field-hint {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.45;
  color: var(--text-muted, var(--el-text-color-secondary));
}

.field-hint--task {
  color: var(--el-color-warning-dark-2, #b88230);
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

.content-editor-wrap {
  width: 100%;
}

.prompt-textarea :deep(.el-textarea__inner) {
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, Courier, monospace;
  font-size: 13px;
  line-height: 1.6;
  tab-size: 2;
}

.variable-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-height: 28px;
  align-items: center;
}

.variable-tag code {
  font-size: 12px;
  font-weight: 600;
}

.no-vars {
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}

.version-info-bar {
  display: flex;
  gap: 24px;
  margin-bottom: 16px;
  padding: 8px 12px;
  background: var(--el-fill-color-lighter);
  border-radius: 4px;
  font-size: 14px;
}

.content-preview {
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  padding: 12px 16px;
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, Courier, monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 500px;
  overflow-y: auto;
  margin: 0;
}
</style>
