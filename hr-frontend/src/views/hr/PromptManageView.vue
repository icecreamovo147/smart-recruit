<script setup lang="ts">
import { onMounted, reactive, ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Edit, Plus, Refresh, View, Back } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
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
} from '@/types/prompt'

// ====== Auth state ======

const auth = useAuthStore()
const currentUserId = computed(() => auth.user?.user_id || 0)

// ====== List State ======

const templateList = ref<PromptTemplate[]>([])
const templateTotal = ref(0)
const templatePage = ref(1)
const templatePageSize = ref(20)
const templateLoading = ref(false)
const templateError = ref('')
const agentTypeFilter = ref('')

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

const agentTypeLabel = (t: string): string => {
  const map: Record<string, string> = {
    hr_agent: 'HR',
    candidate_assistant: '候选人',
  }
  return map[t] || t || '-'
}

const promptRoleLabel = (r: string): string => {
  const map: Record<string, string> = {
    system: '系统',
    user: '用户',
  }
  return map[r] || r || '-'
}

const formatTime = (s?: string): string => {
  if (!s) return '-'
  return new Date(s).toLocaleString('zh-CN')
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

// ====== Edit / Create Dialog ======

const dialogVisible = ref(false)
const dialogTitle = ref('')
const saving = ref(false)
const isEditing = ref(false)
const editingId = ref(0)
const dialogForm = reactive({
  name: '',
  content: '',
  agent_type: 'hr_agent',
  prompt_role: 'system',
  is_active: true,
  change_note: '',
})

// Variables extracted from the current content text.
const extractedVars = computed(() => extractVariables(dialogForm.content))

const resetDialogForm = () => {
  dialogForm.name = ''
  dialogForm.content = ''
  dialogForm.agent_type = 'hr_agent'
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
  dialogForm.agent_type = row.agent_type
  dialogForm.prompt_role = row.prompt_role
  dialogForm.is_active = row.is_active
  dialogForm.change_note = ''
  dialogVisible.value = true
}

const save = async () => {
  if (!dialogForm.name) {
    ElMessage.warning('请输入模板名称')
    return
  }
  if (!dialogForm.content) {
    ElMessage.warning('请输入 Prompt 内容')
    return
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
      ElMessage.success('Prompt 模板已更新（版本已自动递增）')
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
      ElMessage.success('Prompt 模板已创建')
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
    ElMessage.success('Prompt 模板已删除')
    await loadList()
  } catch (e: unknown) {
    ElMessage.error((e as { message?: string }).message || '删除失败')
  }
}

// ====== Toggle active status ======

const handleToggleActive = async (row: PromptTemplate) => {
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
    ElMessage.success('已回滚到历史版本，版本号已自动递增')
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
  <div class="prompt-manage-view">
    <h2 class="page-title">Prompt 管理</h2>

    <div class="toolbar">
      <div class="toolbar-left">
        <el-button type="primary" :icon="Plus" @click="openCreate">新增模板</el-button>
        <el-select
          v-model="agentTypeFilter"
          placeholder="全部 Agent 类型"
          clearable
          style="width: 180px"
          @change="() => { templatePage = 1; loadList() }"
        >
          <el-option value="" label="全部 Agent 类型" />
          <el-option value="hr_agent" label="HR" />
          <el-option value="candidate_assistant" label="候选人" />
        </el-select>
      </div>
      <el-button :icon="Refresh" @click="loadList">刷新</el-button>
    </div>

    <!-- ── List Table ──────────────────────────────────────────────── -->
    <el-table
      v-loading="templateLoading"
      :data="templateList"
      stripe
      border
      style="width: 100%"
      :empty-text="templateError || '暂无数据'"
    >
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column label="角色" width="90">
        <template #default="{ row }: { row: PromptTemplate }">
          {{ promptRoleLabel(row.prompt_role) }}
        </template>
      </el-table-column>
      <el-table-column label="Agent 类型" width="110">
        <template #default="{ row }: { row: PromptTemplate }">
          {{ agentTypeLabel(row.agent_type) }}
        </template>
      </el-table-column>
      <el-table-column prop="version" label="当前版本" width="100" />
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
      <el-table-column label="操作" width="320" fixed="right">
        <template #default="{ row }: { row: PromptTemplate }">
          <el-button size="small" :icon="Edit" @click="openEdit(row)">
            编辑
          </el-button>
          <el-button size="small" :icon="View" @click="openVersionHistory(row)">
            版本历史
          </el-button>
          <el-button
            size="small"
            :type="row.is_active ? 'warning' : 'success'"
            @click="handleToggleActive(row)"
          >
            {{ row.is_active ? '禁用' : '启用' }}
          </el-button>
          <el-button size="small" type="danger" :icon="Delete" @click="handleDelete(row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-wrap">
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

    <!-- ── Edit / Create Dialog ────────────────────────────────────── -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="680px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <el-form :model="dialogForm" label-width="120px">
        <el-form-item label="模板名称" required>
          <el-input v-model="dialogForm.name" placeholder="例如：HR 面试助手 System Prompt" />
        </el-form-item>
        <el-form-item label="Agent 类型" required>
          <el-select v-model="dialogForm.agent_type" style="width: 100%">
            <el-option value="hr_agent" label="HR" />
            <el-option value="candidate_assistant" label="候选人" />
          </el-select>
        </el-form-item>
        <el-form-item label="角色" required>
          <el-select v-model="dialogForm.prompt_role" style="width: 100%">
            <el-option value="system" label="系统（System）" />
            <el-option value="user" label="用户（User）" />
          </el-select>
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
          {{ isEditing ? '保存（版本号自动递增）' : '创建' }}
        </el-button>
      </template>
    </el-dialog>

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
            <el-button
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
