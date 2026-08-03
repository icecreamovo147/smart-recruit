<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowDown,
  CircleCheck,
  Edit,
  Plus,
  Search,
  Tickets,
  View,
  WarningFilled,
} from '@element-plus/icons-vue'
import { t } from '@shared/i18n'
import type {
  AgentSkillInfo,
  AgentSkillPackageInfo,
  AgentSkillVersionInfo,
  CreateAgentSkillPayload,
  CreateAgentSkillVersionPayload,
  UpdateAgentSkillPayload,
} from '@shared/types/agentSkill'
import { formatShanghaiDateTime } from '@shared/utils/format'
import { debugLog } from '@shared/utils/debugLog'
import * as agentSkillApi from '@/api/agentSkill'
import { EmptyGuide, PagePanel } from '@/components/admin-console'
import AgentSkillPackageEditor from '@/components/agent-skill/AgentSkillPackageEditor.vue'
import {
  buildAgentSkillPackageDraft,
  createEmptyPackageEditorModel,
  packageInfoToEditorModel,
  validatePackageEditor,
  type AgentSkillEditorValidation,
  type AgentSkillPackageEditorModel,
} from '@/components/agent-skill/packageEditor'
import { PLATFORM_PERMISSIONS } from '@/permissions'
import { useAuthStore } from '@/stores/auth'

const api = agentSkillApi
const canManage = computed(() => useAuthStore().can(PLATFORM_PERMISSIONS.AI_CONFIG_MANAGE))

const list = ref<AgentSkillInfo[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const statusFilter = ref('')
const loading = ref(false)
const saving = ref(false)
const previewLoading = ref(false)
const savedPreviewLoading = ref(false)
const versionsLoading = ref(false)
const statusChangingId = ref<number | null>(null)
const activatingVersionId = ref<number | null>(null)
const regeneratingVersionId = ref<number | null>(null)

const editorDialogVisible = ref(false)
const previewDrawerVisible = ref(false)
const savedPreviewDrawerVisible = ref(false)
const versionsDrawerVisible = ref(false)
const editingSkill = ref<AgentSkillInfo | null>(null)
const savedPreviewSkill = ref<AgentSkillInfo | null>(null)
const savedPreviewVersion = ref<AgentSkillVersionInfo | null>(null)
const versionSkill = ref<AgentSkillInfo | null>(null)
const versions = ref<AgentSkillVersionInfo[]>([])
const selectedVersion = ref<AgentSkillVersionInfo | null>(null)

const editorModel = ref<AgentSkillPackageEditorModel>(createEmptyPackageEditorModel())
const registryDisplayName = ref('')
const registryDescription = ref('')
const version = ref('1.0.0')
const changeNote = ref('')
const isEnabled = ref(true)
const isManualInvocable = ref(true)
const validation = ref<AgentSkillEditorValidation>({ valid: false, errors: [], warnings: [] })
const serverErrors = ref<string[]>([])
const previewPackage = ref<AgentSkillPackageInfo | null>(null)

const isEditing = computed(() => editingSkill.value !== null)
const currentVersion = computed(() => (
  versions.value.find((item) => item.id === versionSkill.value?.current_version_id) || null
))

const riskLabel = (value?: string) => ({
  low: '低风险',
  medium: '中风险',
  high: '高风险',
  critical: '关键风险',
}[value || ''] || value || '-')

const riskTagType = (value?: string) => {
  if (value === 'high' || value === 'critical') return 'danger'
  if (value === 'medium') return 'warning'
  return 'info'
}

const roleLabel = (value?: string) => value === 'supporting' ? 'Supporting' : 'Primary'

const activationLabel = (value?: string) => ({
  auto: '自动',
  confirm: '确认后启用',
  manual_only: '仅手动确认',
}[value || ''] || value || '-')

const formatTime = (value?: string) => formatShanghaiDateTime(value)

const getErrorMessage = (error: unknown, fallback: string) => (
  (error as { response?: { data?: { message?: string; msg?: string } }; message?: string })?.response?.data?.message
  || (error as { response?: { data?: { msg?: string } } })?.response?.data?.msg
  || (error as { message?: string })?.message
  || fallback
)

const getServerErrors = (error: unknown) => {
  const data = (error as {
    response?: { data?: { errors?: unknown; details?: unknown; message?: string; msg?: string } }
  })?.response?.data
  const candidates = [data?.errors, data?.details]
  for (const value of candidates) {
    if (Array.isArray(value)) return value.map((item) => String(item))
    if (typeof value === 'string' && value.trim()) return [value]
  }
  const message = data?.message || data?.msg
  return message ? [message] : []
}

const nextVersionText = (value?: string) => {
  const semver = value?.trim().match(/^(\d+)\.(\d+)\.(\d+)$/)
  if (!semver) return '1.0.0'
  return `${semver[1]}.${semver[2]}.${Number(semver[3]) + 1}`
}

const loadList = async () => {
  loading.value = true
  try {
    const data = await api.listAgentSkills({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
      enabled_only: statusFilter.value === 'enabled' ? true : undefined,
    })
    const rows = data.list || []
    list.value = statusFilter.value === 'disabled'
      ? rows.filter((item) => !item.is_enabled)
      : rows
    total.value = data.total || list.value.length
  } catch (error: unknown) {
    debugLog.skill.error('agent_skill_list_failed', { error: getErrorMessage(error, '') })
    ElMessage.error(getErrorMessage(error, t('common.operation_failed')))
  } finally {
    loading.value = false
  }
}

const resetEditor = () => {
  editingSkill.value = null
  editorModel.value = createEmptyPackageEditorModel()
  registryDisplayName.value = ''
  registryDescription.value = ''
  version.value = '1.0.0'
  changeNote.value = ''
  isEnabled.value = true
  isManualInvocable.value = true
  validation.value = validatePackageEditor(editorModel.value)
  serverErrors.value = []
  previewPackage.value = null
}

const openCreate = () => {
  resetEditor()
  editorDialogVisible.value = true
}

const loadVersions = async (skillId: number) => {
  const response = await api.listAgentSkillVersions(skillId)
  return response.list || []
}

const findCurrentVersion = (
  skill: AgentSkillInfo,
  versionList: AgentSkillVersionInfo[],
) => versionList.find((item) => item.id === skill.current_version_id) || null

const openEdit = async (row: AgentSkillInfo) => {
  if (saving.value) return
  saving.value = true
  try {
    const detail = (await api.getAgentSkill(row.id)).skill
    const versionList = await loadVersions(row.id)
    const current = findCurrentVersion(detail, versionList)
    if (!current) {
      ElMessage.warning(t('common.invalid_request'))
      return
    }
    editingSkill.value = detail
    registryDisplayName.value = detail.display_name
    registryDescription.value = detail.description
    editorModel.value = packageInfoToEditorModel(current.package)
    editorModel.value.skillName = detail.name
    version.value = nextVersionText(current.version)
    changeNote.value = ''
    isEnabled.value = detail.is_enabled
    isManualInvocable.value = detail.is_manual_invocable
    validation.value = validatePackageEditor(editorModel.value)
    serverErrors.value = []
    previewPackage.value = null
    editorDialogVisible.value = true
  } catch (error: unknown) {
    debugLog.skill.error('agent_skill_edit_load_failed', { skill_id: row.id, error: getErrorMessage(error, '') })
    ElMessage.error(getErrorMessage(error, t('common.operation_failed')))
  } finally {
    saving.value = false
  }
}

const refreshPreview = async () => {
  validation.value = validatePackageEditor(editorModel.value)
  serverErrors.value = []
  previewPackage.value = null
  if (!validation.value.valid) return false

  previewLoading.value = true
  try {
    const result = await api.previewAgentSkill({
      package: buildAgentSkillPackageDraft(editorModel.value),
    })
    previewPackage.value = result.package
    return true
  } catch (error: unknown) {
    serverErrors.value = getServerErrors(error)
    if (!serverErrors.value.length) {
      serverErrors.value = [getErrorMessage(error, t('common.operation_failed'))]
    }
    debugLog.skill.error('agent_skill_preview_failed', { error: getErrorMessage(error, '') })
    return false
  } finally {
    previewLoading.value = false
  }
}

const openPreview = async () => {
  previewDrawerVisible.value = true
  await refreshPreview()
}

const registryPayload = (): UpdateAgentSkillPayload => ({
  display_name: registryDisplayName.value.trim(),
  display_name_set: true,
  description: registryDescription.value.trim(),
  description_set: true,
  is_enabled: isEnabled.value,
  is_enabled_set: true,
  is_manual_invocable: isManualInvocable.value,
  is_manual_invocable_set: true,
})

const createPayload = (): CreateAgentSkillPayload => ({
  version: version.value.trim(),
  package: buildAgentSkillPackageDraft(editorModel.value),
  is_enabled: isEnabled.value,
  is_enabled_set: true,
  is_manual_invocable: isManualInvocable.value,
  is_manual_invocable_set: true,
  change_note: changeNote.value.trim() || undefined,
  activate: true,
})

const versionPayload = (): CreateAgentSkillVersionPayload => ({
  version: version.value.trim(),
  package: buildAgentSkillPackageDraft(editorModel.value),
  change_note: changeNote.value.trim() || undefined,
  activate: true,
})

const saveSkill = async () => {
  if (saving.value) return
  if (!version.value.trim()) {
    ElMessage.warning(t('common.invalid_request'))
    return
  }
  const previewValid = await refreshPreview()
  if (!previewValid) {
    previewDrawerVisible.value = true
    return
  }

  if (isEditing.value) {
    try {
      await ElMessageBox.confirm(
        '本次保存会创建并激活一个新的不可变版本，现有版本不会被覆盖。',
        '创建新版本',
        { confirmButtonText: '创建并激活', cancelButtonText: '取消', type: 'warning' },
      )
    } catch {
      return
    }
  }

  saving.value = true
  try {
    if (editingSkill.value) {
      await api.updateAgentSkill(editingSkill.value.id, registryPayload())
      await api.createAgentSkillVersion(editingSkill.value.id, versionPayload())
    } else {
      await api.createAgentSkill(createPayload())
    }
    ElMessage.success(t('common.success'))
    editorDialogVisible.value = false
    resetEditor()
    await loadList()
  } catch (error: unknown) {
    serverErrors.value = getServerErrors(error)
    ElMessage.error(getErrorMessage(error, t('common.operation_failed')))
  } finally {
    saving.value = false
  }
}

const toggleStatus = async (row: AgentSkillInfo) => {
  if (statusChangingId.value !== null) return
  statusChangingId.value = row.id
  try {
    await api.updateAgentSkillStatus(row.id, { is_enabled: !row.is_enabled })
    ElMessage.success(t('common.success'))
    await loadList()
  } catch (error: unknown) {
    ElMessage.error(getErrorMessage(error, t('common.operation_failed')))
  } finally {
    statusChangingId.value = null
  }
}

const regenerateEmbedding = async (row: AgentSkillInfo) => {
  const versionId = row.current_version_id
  if (!versionId || regeneratingVersionId.value !== null) return
  try {
    await ElMessageBox.confirm(
      `确认重新生成版本 #${versionId} 的 Package 和 Section Embedding？`,
      '重新生成 Embedding',
      { confirmButtonText: '重新生成', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return
  }
  regeneratingVersionId.value = versionId
  try {
    const result = await api.regenerateAgentSkillVersionEmbedding(versionId)
    if (result.failed_count > 0 || result.success_count <= 0) {
      ElMessage.error(t('frontend.operation_failed'))
      return
    }
    ElMessage.success(t('common.success'))
  } catch (error: unknown) {
    ElMessage.error(getErrorMessage(error, t('common.operation_failed')))
  } finally {
    regeneratingVersionId.value = null
  }
}

const openSavedPreview = async (row: AgentSkillInfo) => {
  savedPreviewDrawerVisible.value = true
  savedPreviewLoading.value = true
  savedPreviewSkill.value = row
  savedPreviewVersion.value = null
  try {
    const detail = (await api.getAgentSkill(row.id)).skill
    const versionList = await loadVersions(row.id)
    savedPreviewSkill.value = detail
    savedPreviewVersion.value = findCurrentVersion(detail, versionList)
  } catch (error: unknown) {
    ElMessage.error(getErrorMessage(error, t('common.operation_failed')))
  } finally {
    savedPreviewLoading.value = false
  }
}

const openVersions = async (row: AgentSkillInfo) => {
  versionsDrawerVisible.value = true
  versionsLoading.value = true
  versionSkill.value = row
  versions.value = []
  selectedVersion.value = null
  try {
    const detail = (await api.getAgentSkill(row.id)).skill
    const versionList = await loadVersions(row.id)
    versionSkill.value = detail
    versions.value = versionList
    selectedVersion.value = findCurrentVersion(detail, versionList) || versionList[0] || null
  } catch (error: unknown) {
    ElMessage.error(getErrorMessage(error, t('common.operation_failed')))
  } finally {
    versionsLoading.value = false
  }
}

const activateVersion = async (item: AgentSkillVersionInfo) => {
  if (!versionSkill.value || activatingVersionId.value !== null) return
  try {
    await ElMessageBox.confirm(
      `确认将版本 #${item.id} ${item.version} 设为当前版本？`,
      '切换当前版本',
      { confirmButtonText: '设为当前', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return
  }
  activatingVersionId.value = item.id
  try {
    await api.activateAgentSkillVersion(versionSkill.value.id, item.id)
    versionSkill.value = { ...versionSkill.value, current_version_id: item.id }
    selectedVersion.value = item
    ElMessage.success(t('common.success'))
    await loadList()
  } catch (error: unknown) {
    ElMessage.error(getErrorMessage(error, t('common.operation_failed')))
  } finally {
    activatingVersionId.value = null
  }
}

onMounted(loadList)
</script>

<template>
  <section class="agent-skill-page">
    <PagePanel>
      <div class="workspace-surface">
        <div class="workspace-surface__toolbar">
          <div class="workspace-surface__filters">
            <el-input
              v-model="keyword"
              class="filter-input"
              placeholder="搜索名称或唯一标识"
              clearable
              :prefix-icon="Search"
              @keyup.enter="loadList"
            />
            <el-select v-model="statusFilter" class="filter-select" placeholder="状态" clearable @change="loadList">
              <el-option label="已启用" value="enabled" />
              <el-option label="已停用" value="disabled" />
            </el-select>
          </div>
          <div class="workspace-surface__actions">
            <el-button :icon="Search" @click="loadList">查询</el-button>
            <el-button v-if="canManage" type="primary" :icon="Plus" @click="openCreate">新建 Package v2 Skill</el-button>
          </div>
        </div>

        <div class="workspace-surface__body">
          <div class="agent-skill-table-wrap">
            <el-table v-loading="loading" class="console-table agent-skill-table" :data="list" row-key="id" height="100%">
              <el-table-column label="Skill" min-width="210">
                <template #default="{ row }">
                  <strong class="skill-name">{{ row.display_name || row.name }}</strong>
                  <div class="muted-line">{{ row.name }}</div>
                </template>
              </el-table-column>
              <el-table-column prop="description" label="注册描述" min-width="240" show-overflow-tooltip />
              <el-table-column label="当前版本治理" min-width="280">
                <template #default="{ row }">
                  <div v-if="row.current_version" class="governance-cell">
                    <div>
                      <el-tag size="small" type="info">{{ row.current_version.agent_type }}</el-tag>
                      <el-tag size="small" :type="riskTagType(row.current_version.risk)">
                        {{ riskLabel(row.current_version.risk) }}
                      </el-tag>
                      <el-tag size="small">{{ roleLabel(row.current_version.composition_role) }}</el-tag>
                    </div>
                    <div class="muted-line">
                      {{ row.current_version.category || 'general' }}
                      <span v-if="row.current_version.scenario"> · {{ row.current_version.scenario }}</span>
                      · {{ activationLabel(row.current_version.activation_policy) }}
                    </div>
                  </div>
                  <el-tag v-else size="small" type="info">无当前版本</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="版本" width="150">
                <template #default="{ row }">
                  <template v-if="row.current_version">
                    <div>#{{ row.current_version.version_id }} · {{ row.current_version.version }}</div>
                    <div class="muted-line">{{ row.current_version.package_estimated_tokens }} tokens</div>
                  </template>
                  <span v-else>-</span>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="100">
                <template #default="{ row }">
                  <el-tag size="small" :type="row.is_enabled ? 'success' : 'info'">
                    {{ row.is_enabled ? '已启用' : '已停用' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="更新时间" width="180">
                <template #default="{ row }">{{ formatTime(row.updated_at || row.created_at) }}</template>
              </el-table-column>
              <el-table-column label="操作" width="210" fixed="right">
                <template #default="{ row }">
                  <el-button v-if="canManage" size="small" :icon="Edit" @click="openEdit(row)">新版本</el-button>
                  <el-dropdown
                    trigger="click"
                    @command="(command: string) => {
                      if (command === 'preview') openSavedPreview(row)
                      if (command === 'versions') openVersions(row)
                      if (command === 'embedding') regenerateEmbedding(row)
                      if (command === 'toggle') toggleStatus(row)
                    }"
                  >
                    <el-button size="small">
                      更多<el-icon class="el-icon--right"><ArrowDown /></el-icon>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="preview" :icon="View">预览当前 Package</el-dropdown-item>
                        <el-dropdown-item command="versions" :icon="Tickets">版本管理</el-dropdown-item>
                        <el-dropdown-item
                          command="embedding"
                          divided
                          :disabled="!row.current_version_id || regeneratingVersionId === row.current_version_id"
                        >
                          重新生成当前版本 Embedding
                        </el-dropdown-item>
                        <el-dropdown-item command="toggle" divided>
                          {{ row.is_enabled ? '停用' : '启用' }}
                        </el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </template>
              </el-table-column>
            </el-table>
          </div>
          <EmptyGuide
            v-if="!loading && list.length === 0"
            title="暂无 Agent Skill"
            description="创建第一个由 Manifest、Core 和 Reference Sections 组成的 Package v2 Skill。"
          />
        </div>

        <div class="workspace-surface__pagination">
          <el-pagination
            v-model:current-page="page"
            v-model:page-size="pageSize"
            :total="total"
            :page-sizes="[10, 20, 50]"
            layout="total, sizes, prev, pager, next"
            @current-change="loadList"
            @size-change="() => { page = 1; loadList() }"
          />
        </div>
      </div>
    </PagePanel>

    <el-dialog
      v-model="editorDialogVisible"
      :title="isEditing ? '创建 Agent Skill 新版本' : '新建 Agent Skill'"
      width="min(1240px, 96vw)"
      top="3vh"
      class="agent-skill-editor-dialog"
      :close-on-click-modal="false"
      destroy-on-close
      @closed="resetEditor"
    >
      <div class="editor-dialog-body">
        <div class="editor-intro">
          <div>
            <h2>{{ isEditing ? '基于当前 Package 创建不可变新版本' : 'Agent Skill Package v2' }}</h2>
            <p>平台只提交结构化 Package Draft；Markdown 编译、Hash 和 Token 均由服务端生成。</p>
          </div>
          <el-button :icon="View" :loading="previewLoading" @click="openPreview">服务端预览</el-button>
        </div>

        <div class="version-settings">
          <label class="compact-field">
            <span>版本号</span>
            <el-input v-model="version" placeholder="1.0.0" />
          </label>
          <label class="compact-field">
            <span>启用注册表</span>
            <el-switch v-model="isEnabled" />
          </label>
          <label class="compact-field">
            <span>允许手动选择</span>
            <el-switch v-model="isManualInvocable" />
          </label>
        </div>

        <section class="registry-settings">
          <div>
            <h3>可变注册信息</h3>
            <p v-if="isEditing">仅影响平台列表展示，不会改变已创建版本的 Manifest 或运行时行为。</p>
            <p v-else>首版创建时由 Manifest 初始化；创建后可与不可变版本信息独立维护。</p>
          </div>
          <template v-if="isEditing">
            <label class="compact-field">
              <span>注册表显示名称</span>
              <el-input v-model="registryDisplayName" />
            </label>
            <label class="compact-field">
              <span>注册表描述</span>
              <el-input v-model="registryDescription" type="textarea" :rows="2" />
            </label>
          </template>
          <el-alert
            v-else
            type="info"
            :closable="false"
            title="新建 Skill 的注册名称和描述将采用下方首版 Manifest 值。"
          />
        </section>

        <AgentSkillPackageEditor v-model="editorModel" :skill-name-readonly="isEditing" />

        <label class="compact-field">
          <span>版本变更说明</span>
          <el-input v-model="changeNote" type="textarea" :rows="2" placeholder="记录本版本相对上一版本的变化" />
        </label>

        <div v-if="serverErrors.length" class="validation-panel validation-panel--error">
          <strong>服务端校验未通过</strong>
          <ul><li v-for="item in serverErrors" :key="item">{{ item }}</li></ul>
        </div>

        <div class="dialog-actions">
          <el-button @click="editorDialogVisible = false">取消</el-button>
          <el-button :icon="View" :loading="previewLoading" @click="openPreview">预览</el-button>
          <el-button v-if="canManage" type="primary" :icon="CircleCheck" :loading="saving" @click="saveSkill">
            {{ isEditing ? '创建并激活新版本' : '创建并激活' }}
          </el-button>
        </div>
      </div>
    </el-dialog>

    <el-drawer
      v-model="previewDrawerVisible"
      title="服务端编译预览"
      size="min(680px, 94vw)"
      destroy-on-close
    >
      <div class="preview-body" v-loading="previewLoading">
        <div class="validation-panel" :class="{ 'validation-panel--error': !validation.valid || serverErrors.length }">
          <div class="validation-title">
            <el-icon><CircleCheck v-if="validation.valid && !serverErrors.length" /><WarningFilled v-else /></el-icon>
            <strong>{{ validation.valid && !serverErrors.length ? '客户端与服务端校验通过' : '需要修正' }}</strong>
          </div>
          <ul v-if="validation.errors.length || validation.warnings.length || serverErrors.length">
            <li v-for="item in validation.errors" :key="`local-${item}`">{{ item }}</li>
            <li v-for="item in serverErrors" :key="`server-${item}`">{{ item }}</li>
            <li v-for="item in validation.warnings" :key="`warning-${item}`" class="warning-item">{{ item }}</li>
          </ul>
        </div>

        <template v-if="previewPackage">
          <div class="compile-stats">
            <div><span>Compiled Hash</span><code>{{ previewPackage.compiled_hash }}</code></div>
            <div><span>Core Tokens</span><strong>{{ previewPackage.core_estimated_tokens }}</strong></div>
            <div><span>Package Tokens</span><strong>{{ previewPackage.package_estimated_tokens }}</strong></div>
            <div><span>Sections</span><strong>{{ previewPackage.sections.length }}</strong></div>
          </div>
          <div class="section-token-list">
            <span v-for="section in previewPackage.sections" :key="section.section_key">
              {{ section.section_key }} · {{ section.estimated_tokens }} tokens
            </span>
          </div>
          <pre class="markdown-preview">{{ previewPackage.compiled_markdown }}</pre>
        </template>
        <EmptyGuide
          v-else-if="!previewLoading"
          title="暂无编译结果"
          description="修正校验问题后重新预览，Hash 和 Token 只显示服务端返回值。"
        />
      </div>
    </el-drawer>

    <el-drawer
      v-model="savedPreviewDrawerVisible"
      title="当前 Package 预览"
      size="min(680px, 94vw)"
      destroy-on-close
    >
      <div class="preview-body" v-loading="savedPreviewLoading">
        <template v-if="savedPreviewVersion">
          <div class="compile-stats">
            <div><span>版本</span><strong>#{{ savedPreviewVersion.id }} · {{ savedPreviewVersion.version }}</strong></div>
            <div><span>Compiled Hash</span><code>{{ savedPreviewVersion.package.compiled_hash }}</code></div>
            <div><span>Core Tokens</span><strong>{{ savedPreviewVersion.package.core_estimated_tokens }}</strong></div>
            <div><span>Package Tokens</span><strong>{{ savedPreviewVersion.package.package_estimated_tokens }}</strong></div>
          </div>
          <pre class="markdown-preview">{{ savedPreviewVersion.package.compiled_markdown }}</pre>
        </template>
        <EmptyGuide
          v-else-if="!savedPreviewLoading"
          title="暂无当前版本"
          :description="`${savedPreviewSkill?.display_name || savedPreviewSkill?.name || '该 Skill'} 尚未激活版本。`"
        />
      </div>
    </el-drawer>

    <el-drawer
      v-model="versionsDrawerVisible"
      title="Agent Skill 版本管理"
      size="min(1180px, 96vw)"
      class="agent-skill-versions-drawer"
      destroy-on-close
    >
      <div class="version-drawer" v-loading="versionsLoading">
        <div class="version-hero">
          <div>
            <h2>{{ versionSkill?.display_name || versionSkill?.name || 'Agent Skill' }}</h2>
            <p>版本创建后不可修改。可查看精确 Package Hash、Token 和编译结果，或切换当前版本。</p>
          </div>
          <el-tag v-if="versionSkill?.current_version_id" type="success">
            当前 #{{ versionSkill.current_version_id }}
          </el-tag>
        </div>

        <EmptyGuide v-if="!versionsLoading && !versions.length" title="暂无版本" description="创建 Skill 时会生成首个不可变版本。" />
        <div v-else class="version-layout">
          <aside class="version-list">
            <button
              v-for="item in versions"
              :key="item.id"
              type="button"
              :class="{ active: selectedVersion?.id === item.id }"
              @click="selectedVersion = item"
            >
              <div>
                <strong>#{{ item.id }} · {{ item.version }}</strong>
                <el-tag v-if="item.id === versionSkill?.current_version_id" size="small" type="success">当前</el-tag>
              </div>
              <span>{{ item.change_note || '无变更说明' }}</span>
              <small>{{ formatTime(item.created_at) }}</small>
            </button>
          </aside>

          <section v-if="selectedVersion" class="version-detail">
            <div class="version-detail__head">
              <div>
                <h3>#{{ selectedVersion.id }} · {{ selectedVersion.version }}</h3>
                <p>
                  {{ roleLabel(selectedVersion.package.manifest.composition.role) }}
                  · {{ riskLabel(selectedVersion.package.manifest.risk) }}
                  · {{ activationLabel(selectedVersion.package.manifest.activation_policy) }}
                </p>
              </div>
              <el-button
                v-if="canManage && selectedVersion.id !== versionSkill?.current_version_id"
                type="primary"
                :loading="activatingVersionId === selectedVersion.id"
                @click="activateVersion(selectedVersion)"
              >
                设为当前
              </el-button>
            </div>
            <div class="compile-stats">
              <div><span>Compiled Hash</span><code>{{ selectedVersion.package.compiled_hash }}</code></div>
              <div><span>Core Tokens</span><strong>{{ selectedVersion.package.core_estimated_tokens }}</strong></div>
              <div><span>Package Tokens</span><strong>{{ selectedVersion.package.package_estimated_tokens }}</strong></div>
              <div><span>Sections</span><strong>{{ selectedVersion.package.sections.length }}</strong></div>
            </div>
            <pre class="version-markdown">{{ selectedVersion.package.compiled_markdown }}</pre>
          </section>
        </div>
      </div>
    </el-drawer>
  </section>
</template>

<style scoped>
.agent-skill-page {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  padding-bottom: 24px;
}

.agent-skill-page > .page-panel {
  flex: 1;
  min-height: 0;
}

.filter-input {
  width: 260px;
}

.filter-select {
  width: 140px;
}

.agent-skill-table-wrap,
.agent-skill-table {
  height: 100%;
  min-height: 0;
}

.skill-name {
  color: var(--text-primary);
}

.muted-line {
  margin-top: 4px;
  color: var(--text-faint);
  font-size: 12px;
  line-height: 1.4;
}

.governance-cell {
  display: grid;
  gap: 4px;
}

.governance-cell > div:first-child {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

:global(.agent-skill-editor-dialog) {
  display: flex;
  max-height: 94vh;
  flex-direction: column;
}

:global(.agent-skill-editor-dialog .el-dialog__body) {
  min-height: 0;
  overflow: auto;
  padding-top: 8px;
}

.editor-dialog-body,
.preview-body {
  display: grid;
  gap: 16px;
}

.editor-intro,
.version-hero,
.version-detail__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.editor-intro h2,
.version-hero h2,
.version-detail__head h3 {
  margin: 0;
  color: var(--text-primary);
  font-size: 18px;
}

.editor-intro p,
.version-hero p,
.version-detail__head p {
  margin: 5px 0 0;
  color: var(--text-faint);
  font-size: 12px;
}

.version-settings {
  display: grid;
  grid-template-columns: minmax(220px, 1fr) 160px 180px;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
}

.registry-settings {
  display: grid;
  grid-template-columns: minmax(220px, 0.8fr) minmax(220px, 1fr) minmax(260px, 1.5fr);
  align-items: start;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
}

.registry-settings h3 {
  margin: 0;
  color: var(--text-primary);
  font-size: 14px;
}

.registry-settings p {
  margin: 5px 0 0;
  color: var(--text-faint);
  font-size: 12px;
  line-height: 1.5;
}

.compact-field {
  display: grid;
  gap: 6px;
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.dialog-actions {
  position: sticky;
  bottom: -20px;
  z-index: 2;
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 14px 0 4px;
  background: var(--surface);
}

.validation-panel {
  padding: 12px;
  border: 1px solid color-mix(in srgb, var(--el-color-success) 40%, var(--border));
  border-radius: 8px;
  background: color-mix(in srgb, var(--el-color-success) 8%, var(--surface));
  color: var(--text-secondary);
}

.validation-panel--error {
  border-color: color-mix(in srgb, var(--el-color-danger) 45%, var(--border));
  background: color-mix(in srgb, var(--el-color-danger) 8%, var(--surface));
}

.validation-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.validation-panel ul {
  margin: 8px 0 0;
  padding-left: 20px;
}

.warning-item {
  color: var(--el-color-warning-dark-2);
}

.compile-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.compile-stats > div {
  display: grid;
  min-width: 0;
  gap: 5px;
  padding: 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-muted);
}

.compile-stats span {
  color: var(--text-faint);
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
}

.compile-stats code {
  overflow: hidden;
  color: var(--text-primary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.section-token-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.section-token-list span {
  padding: 5px 8px;
  border-radius: 999px;
  background: var(--surface-muted);
  color: var(--text-secondary);
  font-size: 12px;
}

.markdown-preview,
.version-markdown {
  min-height: 420px;
  margin: 0;
  padding: 16px;
  overflow: auto;
  border-radius: 8px;
  background: var(--surface-muted);
  color: var(--text-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1.65;
  white-space: pre-wrap;
  word-break: break-word;
}

:global(.agent-skill-versions-drawer .el-drawer__body) {
  min-height: 0;
  overflow: hidden;
}

.version-drawer {
  display: flex;
  height: calc(100vh - 96px);
  min-height: 0;
  flex-direction: column;
  gap: 16px;
}

.version-layout {
  display: grid;
  flex: 1;
  min-height: 0;
  grid-template-columns: 330px minmax(0, 1fr);
  gap: 14px;
  overflow: hidden;
}

.version-list {
  display: flex;
  min-height: 0;
  flex-direction: column;
  gap: 8px;
  overflow: auto;
}

.version-list button {
  display: grid;
  gap: 7px;
  width: 100%;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-muted);
  color: var(--text-secondary);
  cursor: pointer;
  font: inherit;
  text-align: left;
}

.version-list button.active {
  border-color: var(--el-color-primary);
  box-shadow: inset 3px 0 0 var(--el-color-primary);
}

.version-list button > div {
  display: flex;
  align-items: center;
  gap: 8px;
}

.version-list span,
.version-list small {
  color: var(--text-faint);
  font-size: 12px;
}

.version-detail {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  gap: 12px;
  overflow: hidden;
  padding: 14px;
  border: 1px solid var(--border);
  border-radius: 8px;
}

.version-markdown {
  flex: 1;
  min-height: 0;
}

@media (max-width: 880px) {
  .version-settings,
  .registry-settings,
  .compile-stats,
  .version-layout {
    grid-template-columns: 1fr;
  }

  .version-list {
    max-height: 260px;
  }
}
</style>
