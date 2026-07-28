<script setup lang="ts">
import { t } from '@shared/i18n'
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  PLATFORM_AI_CAPABILITY_SCHEMA_VERSION,
  PLATFORM_AI_MAX_INPUT_RATIO,
  PLATFORM_AI_MAX_SKILLS,
  PLATFORM_AI_MAX_SKILL_TOKENS,
  PLATFORM_AI_SKILL_POLICY_VERSION,
  createPlatformAICapabilityDraft,
  deletePlatformAICapabilityDraft,
  listPlatformAICapabilities,
  listPlatformAICapabilityVersions,
  publishPlatformAICapabilityVersion,
  updatePlatformAICapabilityDraft,
  type PlatformAICapability,
  type PlatformAICapabilityVersion,
} from '@/api/platformAI'
import { PLATFORM_PERMISSIONS } from '@/permissions'
import { useAuthStore } from '@/stores/auth'
import { listModels } from '@/api/llm'
import { listEmbeddingModels } from '@/api/embedding'
import { listAgentConfigs } from '@/api/agent'
import { listPromptTemplates } from '@/api/prompt'
import { listAgentSkills, listAgentSkillVersions } from '@/api/agentSkill'
import { listMcpToolPolicies } from '@/api/mcp'
import { formatShanghaiDateTime } from '@shared/utils/format'
import { PageHeader, PagePanel } from '@/components/admin-console'
import {
  applySnapshotToForm,
  buildCapabilitySnapshot,
  capabilityReleaseErrorMessage,
  createCapabilityReleaseForm,
  evaluationGateStatus,
  parseCapabilitySnapshot,
  selectedSkillOptions,
  serializeCapabilitySnapshot,
  toAgentSkillReleaseOptions,
  validateCapabilitySnapshot,
  validateSkillComposition,
  type AgentSkillReleaseOption,
} from './capabilityRelease'

const auth = useAuthStore()
const loading = ref(false)
const configurationLoading = ref(false)
const configurationLoaded = ref(false)
const draftSubmitting = ref(false)
const capabilities = ref<PlatformAICapability[]>([])
const selected = ref<PlatformAICapability | null>(null)
const versions = ref<PlatformAICapabilityVersion[]>([])
const editorVisible = ref(false)
const editingVersion = ref<PlatformAICapabilityVersion | null>(null)
const canManage = computed(() => auth.can(PLATFORM_PERMISSIONS.AI_RELEASE_MANAGE))
const canPublish = computed(() => auth.can(PLATFORM_PERMISSIONS.AI_RELEASE_PUBLISH))
type SelectOption = { value: number; label: string }
const options = reactive({
  llm: [] as SelectOption[],
  embedding: [] as SelectOption[],
  agents: [] as SelectOption[],
  prompts: [] as SelectOption[],
  agentSkillVersions: [] as AgentSkillReleaseOption[],
  mcpPolicies: [] as SelectOption[],
})
const form = reactive({
  change_note: '',
  ...createCapabilityReleaseForm(),
})

const audienceLabel = (value: string) => value === 'candidate' ? '候选人端' : '企业招聘端'
const capabilityStatusLabel = (value: string) => ({ active: '已启用', retired: '已停用' }[value] || value)
const versionStatusLabel = (value: string) => ({ draft: '草稿', published: '已发布', retired: '已停用' }[value] || value)
const versionStatusType = (value: string) => value === 'published' ? 'success' : value === 'draft' ? 'warning' : 'info'
const formatTime = (value?: string) => formatShanghaiDateTime(value)

const snapshotOf = (value: string) => parseCapabilitySnapshot(value)
const prettySnapshot = (value: string) => {
  try { return JSON.stringify(JSON.parse(value), null, 2) } catch { return value }
}
const snapshotStats = (value: string) => {
  const snapshot = snapshotOf(value)
  const policy = snapshot?.model_policy
  const refs = snapshot?.configuration_refs
  return {
    llm: policy?.allowed_llm_model_ids?.length || 0,
    defaultLlm: policy?.default_llm_model_id || 0,
    embedding: policy?.allowed_embedding_model_ids?.length || 0,
    agents: refs?.agent_ids?.length || 0,
    prompts: refs?.prompt_template_ids?.length || 0,
    skills: refs?.agent_skill_version_ids?.length || 0,
    mcp: refs?.mcp_policy_ids?.length || 0,
  }
}
const validateSnapshot = (value: string, requireEvaluation = false) =>
  validateCapabilitySnapshot(snapshotOf(value), options.agentSkillVersions, requireEvaluation, selected.value || undefined)
const gateOf = (value: string) => evaluationGateStatus(snapshotOf(value))
const skillsOf = (value: string) => selectedSkillOptions(snapshotOf(value), options.agentSkillVersions)
const publishDisabled = (version: PlatformAICapabilityVersion) =>
  version.status !== 'draft' || !validateSnapshot(version.snapshot_json, true).valid
const editorSnapshot = computed(() =>
  selected.value ? buildCapabilitySnapshot(selected.value, form) : null,
)
const editorGate = computed(() => evaluationGateStatus(editorSnapshot.value))
const editorCompositionErrors = computed(() =>
  validateSkillComposition(form.agent_skill_version_ids, options.agentSkillVersions),
)
const editorDraftValidation = computed(() =>
  validateCapabilitySnapshot(editorSnapshot.value, options.agentSkillVersions, false, selected.value || undefined),
)
const saveDraftDisabled = computed(() =>
  draftSubmitting.value || !selected.value || !form.change_note.trim() || !editorDraftValidation.value.valid,
)

const option = (id: number, label: string): SelectOption => ({ value: id, label: `${label} (#${id})` })
const loadConfigurationOptions = async () => {
  if (configurationLoaded.value || configurationLoading.value) return
  configurationLoading.value = true
  try {
    const [llm, embedding, agents, prompts, agentSkills, policies] = await Promise.all([
      listModels(1, 500),
      listEmbeddingModels(1, 500),
      listAgentConfigs(1, 500),
      listPromptTemplates(1, 500),
      listAgentSkills({ page: 1, page_size: 500 }),
      listMcpToolPolicies({ page: 1, page_size: 500 }),
    ])
    options.llm = (llm.list || []).filter(row => row.is_enabled).map(row => option(row.id, row.display_name || row.model_name))
    options.embedding = (embedding.list || []).filter(row => row.is_enabled).map(row => option(row.id, row.display_name || row.model_name))
    options.agents = (agents.list || []).filter(row => row.is_enabled).map(row => option(row.id, row.display_name || row.name))
    options.prompts = (prompts.list || []).filter(row => row.is_active).map(row => option(row.id, row.name))
    options.mcpPolicies = (policies.list || []).filter(row => row.is_enabled).map(row => option(row.id, `${row.server_name || row.server_id} / ${row.tool_name}`))
    options.agentSkillVersions = (await Promise.all((agentSkills.list || []).map(async row => {
      const result = await listAgentSkillVersions(row.id)
      return toAgentSkillReleaseOptions(row, result.list || [])
    }))).flat()
    configurationLoaded.value = true
  } finally {
    configurationLoading.value = false
  }
}

const loadVersions = async (capability: PlatformAICapability) => {
  selected.value = capability
  const result = await listPlatformAICapabilityVersions(capability.id)
  versions.value = result.list || []
}

const load = async () => {
  loading.value = true
  try {
    const [result] = await Promise.all([
      listPlatformAICapabilities(),
      loadConfigurationOptions(),
    ])
    capabilities.value = result.list || []
    if (capabilities.value.length) await loadVersions(capabilities.value[0])
  } catch (error) {
    ElMessage.error(capabilityReleaseErrorMessage(error))
  } finally {
    loading.value = false
  }
}

const syncDefaultLlmModel = () => {
  if (!form.allowed_llm_model_ids.includes(form.default_llm_model_id || 0)) form.default_llm_model_id = form.allowed_llm_model_ids[0]
}

const syncDefaultEmbeddingModel = () => {
  if (!form.allowed_embedding_model_ids.includes(form.default_embedding_model_id || 0)) form.default_embedding_model_id = form.allowed_embedding_model_ids[0]
}

const openEditor = async (version?: PlatformAICapabilityVersion) => {
  try {
    await loadConfigurationOptions()
  } catch {
    ElMessage.error(t('frontend.operation_failed'))
    return
  }
  editingVersion.value = version || null
  const baseline = versions.value.find((item) => item.id === selected.value?.current_published_version_id)
    || versions.value.find((item) => item.status === 'published')
  applySnapshotToForm(form, snapshotOf((version || baseline)?.snapshot_json || ''))
  form.change_note = version?.change_note || ''
  syncDefaultLlmModel()
  syncDefaultEmbeddingModel()
  editorVisible.value = true
}

const submitDraft = async () => {
  if (draftSubmitting.value) return
  const capability = selected.value
  const editingVersionID = editingVersion.value?.id
  if (!capability || !form.change_note.trim()) { ElMessage.warning(t('common.invalid_request')); return }
  const draftForm = {
    ...form,
    evaluation_suite_hash: '',
    evaluation_result_hash: '',
  }
  const draftSnapshot = buildCapabilitySnapshot(capability, draftForm)
  const validation = validateCapabilitySnapshot(draftSnapshot, options.agentSkillVersions, false, capability)
  if (!validation.valid) {
    ElMessage.warning(validation.errors[0] || t('common.invalid_request'))
    return
  }
  const payload = {
    snapshot_json: serializeCapabilitySnapshot(capability, draftForm),
    change_note: form.change_note.trim(),
  }
  draftSubmitting.value = true
  try {
    const result = editingVersionID
      ? await updatePlatformAICapabilityDraft(editingVersionID, payload, { silentError: true })
      : await createPlatformAICapabilityDraft(capability.id, payload, { silentError: true })
    if (selected.value?.id === capability.id) {
      const existingIndex = versions.value.findIndex((version) => version.id === result.version.id)
      versions.value = existingIndex >= 0
        ? versions.value.map((version, index) => index === existingIndex ? result.version : version)
        : [...versions.value, result.version]
    }
    editorVisible.value = false
    ElMessage.success(t('common.success'))
  } catch (error) {
    ElMessage.error(capabilityReleaseErrorMessage(error))
  } finally {
    draftSubmitting.value = false
  }
}

const publish = async (version: PlatformAICapabilityVersion) => {
  const validation = validateSnapshot(version.snapshot_json, true)
  if (validation.errors.length) {
    ElMessage.error(validation.errors[0])
    return
  }
  const warningText = validation.warnings.length ? `\n风险提示：${validation.warnings.join('；')}。` : ''
  await ElMessageBox.confirm(
    `确认发布 V${version.version}？发布后快照不可修改，套餐只能引用这个固定版本。${warningText}`,
    '发布 AI 能力版本',
    { type: 'warning', confirmButtonText: '确认发布', cancelButtonText: '取消' },
  )
  try {
    await publishPlatformAICapabilityVersion(version.id, { silentError: true })
    ElMessage.success(t('common.success'))
    if (selected.value) await loadVersions(selected.value)
    const result = await listPlatformAICapabilities()
    capabilities.value = result.list || []
  } catch (error) {
    ElMessage.error(capabilityReleaseErrorMessage(error))
  }
}

const deleteDraft = async (version: PlatformAICapabilityVersion) => {
  try {
    await ElMessageBox.confirm(
      `确认删除 V${version.version} 草稿？删除后无法恢复。`,
      '删除能力版本草稿',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消', confirmButtonClass: 'el-button--danger' },
    )
  } catch {
    return
  }
  try {
    await deletePlatformAICapabilityDraft(version.id)
    ElMessage.success(t('common.success'))
    if (selected.value) await loadVersions(selected.value)
  } catch (error) {
    ElMessage.error(capabilityReleaseErrorMessage(error))
  }
}

onMounted(load)
</script>

<template>
  <section class="console-page ai-control-page" v-loading="loading">
    <PagePanel>
      <PageHeader
        kicker="CAPABILITY RELEASE"
        title="AI 能力发布"
        description="编排已治理的模型和 Agent 资产，发布不可变能力版本，再由套餐固定引用。"
      >
        <template #primary>
          <el-button v-if="canManage && selected" type="primary" :loading="configurationLoading" @click="openEditor()">创建版本草稿</el-button>
        </template>
      </PageHeader>

      <div class="release-layout">
        <aside class="surface-card capability-list">
          <button v-for="item in capabilities" :key="item.id" type="button" :class="{ active: selected?.id === item.id }" @click="loadVersions(item)">
            <span><strong>{{ item.name }}</strong><small>{{ item.capability_key }}</small></span>
            <el-tag size="small" :type="item.audience === 'candidate' ? 'success' : 'primary'">{{ audienceLabel(item.audience) }}</el-tag>
          </button>
        </aside>

        <main class="surface-card version-panel">
          <header v-if="selected"><div><h2>{{ selected.name }}</h2><p>{{ selected.description || '暂无能力说明' }} · {{ audienceLabel(selected.audience) }}</p></div><el-tag>{{ capabilityStatusLabel(selected.status) }}</el-tag></header>
          <section v-for="version in versions" :key="version.id" class="version-card">
            <div class="version-title"><div><strong>V{{ version.version }}</strong><small>{{ version.change_note || '暂无变更说明' }}</small></div><div class="version-statuses"><el-tag v-if="version.id === selected?.current_published_version_id" type="primary">当前生效</el-tag><el-tag :type="versionStatusType(version.status)">{{ versionStatusLabel(version.status) }}</el-tag></div></div>
            <div class="snapshot-summary">
              <div><span>LLM 模型池</span><strong>{{ snapshotStats(version.snapshot_json).llm }}</strong><small>默认 #{{ snapshotStats(version.snapshot_json).defaultLlm || '-' }}</small></div>
              <div><span>Embedding</span><strong>{{ snapshotStats(version.snapshot_json).embedding }}</strong><small>允许模型</small></div>
              <div><span>Agent / Prompt</span><strong>{{ snapshotStats(version.snapshot_json).agents }} / {{ snapshotStats(version.snapshot_json).prompts }}</strong><small>固定资产</small></div>
              <div><span>Skill / MCP</span><strong>{{ snapshotStats(version.snapshot_json).skills }} / {{ snapshotStats(version.snapshot_json).mcp }}</strong><small>固定版本与策略</small></div>
            </div>
            <div class="evaluation-gate">
              <div>
                <span>确定性评测门禁</span>
                <el-tag :type="gateOf(version.snapshot_json).passed ? 'success' : 'danger'">
                  {{ gateOf(version.snapshot_json).label }}
                </el-tag>
              </div>
              <small>{{ gateOf(version.snapshot_json).reason }}</small>
              <code>Suite {{ snapshotOf(version.snapshot_json)?.skill_runtime_policy?.evaluation_suite_hash || '-' }}</code>
              <code>Result {{ snapshotOf(version.snapshot_json)?.skill_runtime_policy?.evaluation_result_hash || '-' }}</code>
            </div>
            <div v-if="skillsOf(version.snapshot_json).length" class="selected-skills">
              <article v-for="skill in skillsOf(version.snapshot_json)" :key="skill.value" class="selected-skill-card">
                <div>
                  <strong>{{ skill.label }}</strong>
                  <el-tag size="small" :type="skill.skillEnabled ? 'success' : 'danger'">
                    {{ skill.skillEnabled ? '已启用' : '已停用' }}
                  </el-tag>
                </div>
                <small>{{ skill.agentType }} / {{ skill.scenario || '-' }}</small>
                <div class="skill-metadata">
                  <span>{{ skill.role }}</span>
                  <span>{{ skill.risk }}</span>
                  <span>Core {{ skill.coreEstimatedTokens }} tokens</span>
                  <span>Package {{ skill.packageEstimatedTokens }} tokens</span>
                </div>
                <code>{{ skill.compiledHash }}</code>
              </article>
            </div>
            <el-alert v-if="validateSnapshot(version.snapshot_json).errors.length" :title="validateSnapshot(version.snapshot_json).errors.join('；')" type="error" :closable="false" show-icon />
            <el-alert v-else-if="validateSnapshot(version.snapshot_json).warnings.length" :title="validateSnapshot(version.snapshot_json).warnings.join('；')" type="warning" :closable="false" show-icon />
            <details><summary>查看原始发布快照</summary><pre>{{ prettySnapshot(version.snapshot_json) }}</pre></details>
            <footer>
              <span>快照 {{ version.snapshot_hash ? version.snapshot_hash.slice(0, 12) : '-' }} · 发布 {{ formatTime(version.published_at) }}</span>
              <div>
                <el-button v-if="canManage && version.status === 'draft'" link type="primary" @click="openEditor(version)">编辑草稿</el-button>
                <el-button v-if="canManage && version.status === 'draft'" link type="danger" @click="deleteDraft(version)">删除草稿</el-button>
                <el-tooltip
                  v-if="canPublish && version.status === 'draft'"
                  :disabled="!publishDisabled(version)"
                  :content="validateSnapshot(version.snapshot_json, true).errors.join('；')"
                  placement="top"
                >
                  <span>
                    <el-button
                      link
                      type="success"
                      :disabled="publishDisabled(version)"
                      data-testid="publish-capability-version"
                      @click="publish(version)"
                    >
                      发布并冻结
                    </el-button>
                  </span>
                </el-tooltip>
              </div>
            </footer>
          </section>
          <el-empty v-if="selected && !versions.length" description="尚未创建能力版本" />
        </main>
      </div>
    </PagePanel>

    <el-dialog v-model="editorVisible" class="capability-release-dialog" :title="`${selected?.name || ''} · ${editingVersion ? `编辑 V${editingVersion.version}` : '新建版本'}`" width="880px" top="4vh">
      <el-alert title="草稿保存时由后端执行确定性评测并写入只读 Hash；发布后快照不可修改。" type="info" :closable="false" show-icon />
      <el-form class="editor-form" label-position="top">
        <el-form-item label="变更说明" required><el-input v-model="form.change_note" maxlength="500" show-word-limit /></el-form-item>
        <el-divider content-position="left">不可变快照身份</el-divider>
        <div class="identity-grid">
          <el-form-item label="Schema Version">
            <el-input :model-value="String(PLATFORM_AI_CAPABILITY_SCHEMA_VERSION)" disabled />
          </el-form-item>
          <el-form-item label="Capability Key">
            <el-input :model-value="selected?.capability_key || ''" disabled />
          </el-form-item>
          <el-form-item label="Audience">
            <el-input :model-value="selected?.audience || ''" disabled />
          </el-form-item>
        </div>
        <el-divider content-position="left">模型策略</el-divider>
        <el-form-item label="允许调用的 LLM 模型" required><el-select v-model="form.allowed_llm_model_ids" multiple filterable class="full-width" @change="syncDefaultLlmModel"><el-option v-for="item in options.llm" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item label="默认 LLM 模型" required><el-select v-model="form.default_llm_model_id" filterable class="full-width" placeholder="请选择默认 LLM 模型"><el-option v-for="item in options.llm.filter(option => form.allowed_llm_model_ids.includes(option.value))" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item label="允许调用的 Embedding 模型"><el-select v-model="form.allowed_embedding_model_ids" multiple filterable class="full-width" @change="syncDefaultEmbeddingModel"><el-option v-for="item in options.embedding" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item label="默认 Embedding 模型"><el-select v-model="form.default_embedding_model_id" clearable filterable class="full-width" placeholder="请选择默认 Embedding 模型"><el-option v-for="item in options.embedding.filter(option => form.allowed_embedding_model_ids.includes(option.value))" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-divider content-position="left">固定配置引用</el-divider>
        <el-form-item label="Agent"><el-select v-model="form.agent_ids" multiple filterable class="full-width"><el-option v-for="item in options.agents" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item label="Prompt 模板"><el-select v-model="form.prompt_template_ids" multiple filterable class="full-width"><el-option v-for="item in options.prompts" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item label="Agent Skill 精确版本">
          <el-select
            v-model="form.agent_skill_version_ids"
            multiple
            filterable
            class="full-width"
            data-testid="agent-skill-version-select"
          >
            <el-option
              v-for="item in options.agentSkillVersions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
              :disabled="!item.skillEnabled"
            >
              <div class="skill-option">
                <span>
                  <strong>{{ item.label }}</strong>
                  <small>{{ item.agentType }} / {{ item.scenario || '-' }}</small>
                </span>
                <span>
                  <el-tag size="small">{{ item.role }}</el-tag>
                  <el-tag size="small" :type="item.risk === 'critical' || item.risk === 'high' ? 'warning' : 'info'">{{ item.risk }}</el-tag>
                  <el-tag size="small" :type="item.skillEnabled ? 'success' : 'danger'">{{ item.skillEnabled ? '已启用' : '已停用' }}</el-tag>
                </span>
              </div>
            </el-option>
          </el-select>
          <div v-if="form.agent_skill_version_ids.length" class="selected-skills editor-selected-skills">
            <article
              v-for="skill in options.agentSkillVersions.filter(item => form.agent_skill_version_ids.includes(item.value))"
              :key="skill.value"
              class="selected-skill-card"
            >
              <div><strong>{{ skill.label }}</strong><el-tag size="small">{{ skill.role }}</el-tag></div>
              <small>{{ skill.agentType }} / {{ skill.scenario || '-' }} · {{ skill.risk }}</small>
              <div class="skill-metadata">
                <span>Core {{ skill.coreEstimatedTokens }} tokens</span>
                <span>Package {{ skill.packageEstimatedTokens }} tokens</span>
              </div>
              <code>{{ skill.compiledHash }}</code>
            </article>
          </div>
          <el-alert
            v-if="editorCompositionErrors.length"
            :title="editorCompositionErrors.join('；')"
            type="error"
            :closable="false"
            show-icon
          />
        </el-form-item>
        <el-form-item label="MCP 策略"><el-select v-model="form.mcp_policy_ids" multiple filterable class="full-width"><el-option v-for="item in options.mcpPolicies" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-divider content-position="left">Skill Package v2 运行策略</el-divider>
        <div class="policy-grid">
          <el-form-item label="Policy Version">
            <el-input :model-value="PLATFORM_AI_SKILL_POLICY_VERSION" disabled />
          </el-form-item>
          <el-form-item label="最大 Skill Tokens" required>
            <el-input-number v-model="form.max_skill_tokens" :min="1" :max="PLATFORM_AI_MAX_SKILL_TOKENS" :step="100" controls-position="right" />
          </el-form-item>
          <el-form-item label="最大输入占比" required>
            <el-input-number v-model="form.max_input_ratio" :min="0.01" :max="PLATFORM_AI_MAX_INPUT_RATIO" :step="0.01" :precision="2" controls-position="right" />
          </el-form-item>
          <el-form-item label="最大 Skill 数量">
            <el-input :model-value="String(PLATFORM_AI_MAX_SKILLS)" disabled />
          </el-form-item>
        </div>
        <el-divider content-position="left">确定性评测门禁</el-divider>
        <el-alert
          :title="editorGate.reason"
          :type="editorGate.passed ? 'success' : 'warning'"
          :closable="false"
          show-icon
        />
        <el-form-item label="Evaluation Suite Hash">
          <el-input :model-value="form.evaluation_suite_hash" readonly placeholder="保存草稿后由后端评测器写入" />
        </el-form-item>
        <el-form-item label="Evaluation Result Hash">
          <el-input :model-value="form.evaluation_result_hash" readonly placeholder="保存草稿后由后端评测器写入" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">取消</el-button>
        <el-button
          type="primary"
          :disabled="saveDraftDisabled"
          :loading="draftSubmitting"
          data-testid="save-capability-draft"
          @click="submitDraft"
        >
          保存草稿
        </el-button>
      </template>
    </el-dialog>
  </section>
</template>

<style scoped>
.full-width { width: 100%; }

.ai-control-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.ai-control-page > .page-panel {
  flex: 1;
  min-height: 0;
}

.version-panel > header,
.version-title,
.version-card footer {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 20px;
}

.version-panel h2 {
  margin: 0;
}

.version-panel header p {
  margin: 7px 0 0;
  color: var(--el-text-color-secondary);
}

.release-layout {
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr);
  gap: 20px;
}

.capability-list {
  display: grid;
  gap: 8px;
  padding: 10px;
  height: max-content;
  border: 1px solid var(--surface-soft-border);
  border-radius: var(--surface-soft-radius, 12px);
  background: var(--surface-soft-bg);
  box-shadow: var(--surface-soft-shadow);
}

.capability-list button {
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
  padding: 14px;
  border: 1px solid transparent;
  border-radius: 10px;
  background: transparent;
  text-align: left;
  color: inherit;
  cursor: pointer;
  transition: border-color .15s ease, background-color .15s ease, box-shadow .15s ease;
}

.capability-list button:hover:not(.active) {
  border-color: var(--surface-soft-border);
  background: var(--control-hover-bg);
}

.capability-list button.active {
  border-color: color-mix(in srgb, var(--brand) 32%, var(--surface-soft-border));
  background: var(--el-color-primary-light-9);
  box-shadow: inset 3px 0 0 var(--brand);
}

.capability-list button.active:hover {
  background: var(--el-color-primary-light-9);
}

.capability-list span,
.version-title > div {
  display: grid;
  gap: 4px;
}

.version-title > .version-statuses {
  display: flex;
  align-items: center;
  gap: 8px;
}

.capability-list small,
.version-title small,
.version-card footer {
  color: var(--el-text-color-secondary);
}

.version-panel {
  padding: 22px;
  border: 1px solid var(--surface-soft-border);
  border-radius: var(--surface-soft-radius, 12px);
  background: var(--surface-soft-bg);
  box-shadow: var(--surface-soft-shadow);
}

.version-card {
  display: grid;
  gap: 14px;
  margin-top: 16px;
  padding: 18px;
  border: 1px solid var(--surface-soft-border);
  border-radius: var(--surface-soft-radius, 12px);
}

.snapshot-summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.snapshot-summary > div {
  display: grid;
  gap: 4px;
  padding: 13px;
  border-radius: 10px;
  background: var(--el-fill-color-lighter);
}

.snapshot-summary span,
.snapshot-summary small {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.snapshot-summary strong {
  font-size: 20px;
}

.evaluation-gate,
.selected-skill-card {
  display: grid;
  gap: 8px;
  padding: 13px;
  border: 1px solid var(--surface-soft-border);
  border-radius: 10px;
  background: var(--el-fill-color-lighter);
}

.evaluation-gate > div,
.selected-skill-card > div,
.skill-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
}

.evaluation-gate small,
.selected-skill-card small {
  color: var(--el-text-color-secondary);
}

.evaluation-gate code,
.selected-skill-card code {
  overflow: hidden;
  color: var(--el-text-color-secondary);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.selected-skills {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.skill-metadata {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-start !important;
  gap: 6px !important;
}

.skill-metadata span {
  padding: 3px 7px;
  border-radius: 6px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color);
  font-size: 11px;
}

.identity-grid,
.policy-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 14px;
}

.identity-grid {
  grid-template-columns: 120px minmax(0, 1fr) 180px;
}

.skill-option {
  width: 100%;
}

.skill-option > span {
  display: flex;
  align-items: center;
  gap: 6px;
}

.skill-option > span:first-child {
  min-width: 0;
  flex-direction: column;
  align-items: flex-start;
}

.skill-option small {
  color: var(--el-text-color-secondary);
}

.editor-selected-skills {
  width: 100%;
  margin-top: 10px;
}

.version-card details summary {
  cursor: pointer;
  color: var(--el-color-primary);
  font-size: 13px;
}

.version-card pre {
  max-height: 330px;
  overflow: auto;
  padding: 14px;
  border-radius: 9px;
  background: var(--el-fill-color-lighter);
  font-size: 12px;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
}

.version-card footer {
  font-size: 12px;
  align-items: center;
}

.editor-form {
  margin-top: 18px;
}

.editor-form :deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

:global(.el-overlay-dialog .el-dialog.capability-release-dialog),
:global(.capability-release-dialog .el-dialog__header),
:global(.capability-release-dialog .el-dialog__body),
:global(.capability-release-dialog .el-dialog__footer) {
  background: var(--surface-solid-bg);
}

:global(.capability-release-dialog .el-dialog__body) {
  max-height: calc(88vh - 132px);
  overflow-y: auto;
}

:global(.capability-release-dialog .el-dialog__headerbtn) {
  top: 14px;
  right: 16px;
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border: 1px solid transparent;
  border-radius: 8px;
  color: var(--text-muted);
  background: transparent;
  transition: color .15s ease, border-color .15s ease, background-color .15s ease;
}

:global(.capability-release-dialog .el-dialog__headerbtn:hover) {
  color: var(--brand);
  border-color: var(--control-border);
  background: var(--control-hover-bg);
}

:global(.capability-release-dialog .el-dialog__close) {
  font-size: 16px;
}

@media (max-width: 1100px) {
  .snapshot-summary {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 900px) {
  .release-layout {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 600px) {
  .snapshot-summary,
  .selected-skills,
  .identity-grid,
  .policy-grid {
    grid-template-columns: 1fr;
  }
}
</style>
