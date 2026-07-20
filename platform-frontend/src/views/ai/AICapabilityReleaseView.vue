<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  createPlatformAICapabilityDraft,
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
import { listSkills, listSkillVersions } from '@/api/skill'
import { listAgentSkills, listAgentSkillVersions } from '@/api/agentSkill'
import { listMcpToolPolicies } from '@/api/mcp'

const auth = useAuthStore()
const loading = ref(false)
const configurationLoading = ref(false)
const configurationLoaded = ref(false)
const capabilities = ref<PlatformAICapability[]>([])
const selected = ref<PlatformAICapability | null>(null)
const versions = ref<PlatformAICapabilityVersion[]>([])
const editorVisible = ref(false)
const editingVersion = ref<PlatformAICapabilityVersion | null>(null)
const canManage = computed(() => auth.can(PLATFORM_PERMISSIONS.AI_RELEASE_MANAGE))
const canPublish = computed(() => auth.can(PLATFORM_PERMISSIONS.AI_RELEASE_PUBLISH))
type SelectOption = { value: number; label: string }
const options = reactive({ llm: [] as SelectOption[], embedding: [] as SelectOption[], agents: [] as SelectOption[], prompts: [] as SelectOption[], agentSkillVersions: [] as SelectOption[], aiSkillVersions: [] as SelectOption[], mcpPolicies: [] as SelectOption[] })
const form = reactive({ change_note: '', allowed_llm_model_ids: [] as number[], default_llm_model_id: 0, allowed_embedding_model_ids: [] as number[], default_embedding_model_id: 0, agent_ids: [] as number[], prompt_template_ids: [] as number[], agent_skill_version_ids: [] as number[], ai_skill_version_ids: [] as number[], mcp_policy_ids: [] as number[] })

const audienceLabel = (value: string) => value === 'candidate' ? '候选人端' : '企业招聘端'
const formatTime = (value?: string) => value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
type CapabilitySnapshot = {
  model_policy?: { allowed_llm_model_ids?: number[]; default_llm_model_id?: number; allowed_embedding_model_ids?: number[]; default_embedding_model_id?: number }
  configuration_refs?: { agent_ids?: number[]; prompt_template_ids?: number[]; agent_skill_version_ids?: number[]; ai_skill_version_ids?: number[]; mcp_policy_ids?: number[] }
}
const snapshotOf = (value: string): CapabilitySnapshot => {
  try { return JSON.parse(value) as CapabilitySnapshot } catch { return {} }
}
const prettySnapshot = (value: string) => {
  try { return JSON.stringify(JSON.parse(value), null, 2) } catch { return value }
}
const snapshotStats = (value: string) => {
  const snapshot = snapshotOf(value)
  const policy = snapshot.model_policy || {}
  const refs = snapshot.configuration_refs || {}
  return {
    llm: policy.allowed_llm_model_ids?.length || 0,
    defaultLlm: policy.default_llm_model_id || 0,
    embedding: policy.allowed_embedding_model_ids?.length || 0,
    agents: refs.agent_ids?.length || 0,
    prompts: refs.prompt_template_ids?.length || 0,
    skills: (refs.agent_skill_version_ids?.length || 0) + (refs.ai_skill_version_ids?.length || 0),
    mcp: refs.mcp_policy_ids?.length || 0,
  }
}
const validateSnapshot = (value: string) => {
  const stats = snapshotStats(value)
  const snapshot = snapshotOf(value)
  const allowed = snapshot.model_policy?.allowed_llm_model_ids || []
  const errors: string[] = []
  const warnings: string[] = []
  if (!stats.llm) errors.push('未配置可用 LLM 模型池')
  if (!stats.defaultLlm || !allowed.includes(stats.defaultLlm)) errors.push('默认 LLM 不属于模型池')
  if (!stats.agents && !stats.prompts) warnings.push('未固定 Agent 或 Prompt 引用')
  if (!stats.skills) warnings.push('未固定任何 Skill 版本')
  return { errors, warnings }
}

const option = (id: number, label: string): SelectOption => ({ value: id, label: `${label} (#${id})` })
const loadConfigurationOptions = async () => {
  if (configurationLoaded.value || configurationLoading.value) return
  configurationLoading.value = true
  try {
  const [llm, embedding, agents, prompts, skills, agentSkills, policies] = await Promise.all([
    listModels(1, 500), listEmbeddingModels(1, 500), listAgentConfigs(1, 500), listPromptTemplates(1, 500), listSkills(1, 500), listAgentSkills({ page: 1, page_size: 500 }), listMcpToolPolicies({ page: 1, page_size: 500 }),
  ])
  options.llm = (llm.list || []).filter(row => row.is_enabled).map(row => option(row.id, row.display_name || row.model_name))
  options.embedding = (embedding.list || []).filter(row => row.is_enabled).map(row => option(row.id, row.display_name || row.model_name))
  options.agents = (agents.list || []).filter(row => row.is_enabled).map(row => option(row.id, row.display_name || row.name))
  options.prompts = (prompts.list || []).filter(row => row.is_active).map(row => option(row.id, row.name))
  options.mcpPolicies = (policies.list || []).filter(row => row.is_enabled).map(row => option(row.id, `${row.server_name || row.server_id} / ${row.tool_name}`))
  options.aiSkillVersions = (await Promise.all((skills.list || []).filter(row => row.is_enabled).map(async row => {
    const result = await listSkillVersions(row.id)
    return (result.list || []).map(version => option(version.id, `${row.display_name || row.name} · ${version.version}`))
  }))).flat()
  options.agentSkillVersions = (await Promise.all((agentSkills.list || []).filter(row => row.is_enabled).map(async row => {
    const result = await listAgentSkillVersions(row.id)
    return (result.list || []).map(version => option(version.id, `${row.display_name || row.name} · ${version.version}`))
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
    const result = await listPlatformAICapabilities()
    capabilities.value = result.list || []
    if (capabilities.value.length) await loadVersions(capabilities.value[0])
  } finally { loading.value = false }
}

const openEditor = async (version?: PlatformAICapabilityVersion) => {
	try {
		await loadConfigurationOptions()
	} catch {
		ElMessage.error('发布配置选项加载失败，请检查模型与资产管理权限后重试')
		return
	}
	editingVersion.value = version || null
	const snapshot = version ? snapshotOf(version.snapshot_json) : { model_policy: {}, configuration_refs: {} }
	form.allowed_llm_model_ids = snapshot.model_policy?.allowed_llm_model_ids || []
	form.default_llm_model_id = snapshot.model_policy?.default_llm_model_id || 0
	form.allowed_embedding_model_ids = snapshot.model_policy?.allowed_embedding_model_ids || []
	form.default_embedding_model_id = snapshot.model_policy?.default_embedding_model_id || 0
	form.agent_ids = snapshot.configuration_refs?.agent_ids || []
	form.prompt_template_ids = snapshot.configuration_refs?.prompt_template_ids || []
	form.agent_skill_version_ids = snapshot.configuration_refs?.agent_skill_version_ids || []
	form.ai_skill_version_ids = snapshot.configuration_refs?.ai_skill_version_ids || []
	form.mcp_policy_ids = snapshot.configuration_refs?.mcp_policy_ids || []
  form.change_note = version?.change_note || ''
  editorVisible.value = true
}

const submitDraft = async () => {
  if (!selected.value || !form.change_note.trim()) { ElMessage.warning('请填写版本变更说明'); return }
	if (!form.allowed_llm_model_ids.length || !form.default_llm_model_id || !form.allowed_llm_model_ids.includes(form.default_llm_model_id)) { ElMessage.warning('请选择模型池，并确保默认模型属于模型池'); return }
	const normalized = JSON.stringify({ schema_version: 1, capability_key: selected.value.capability_key, audience: selected.value.audience, model_policy: { allowed_llm_model_ids: form.allowed_llm_model_ids, default_llm_model_id: form.default_llm_model_id, allowed_embedding_model_ids: form.allowed_embedding_model_ids, default_embedding_model_id: form.default_embedding_model_id || 0 }, configuration_refs: { agent_ids: form.agent_ids, prompt_template_ids: form.prompt_template_ids, agent_skill_version_ids: form.agent_skill_version_ids, ai_skill_version_ids: form.ai_skill_version_ids, mcp_policy_ids: form.mcp_policy_ids } })
  if (editingVersion.value) {
    await updatePlatformAICapabilityDraft(editingVersion.value.id, { snapshot_json: normalized, change_note: form.change_note.trim() })
  } else {
    await createPlatformAICapabilityDraft(selected.value.id, { snapshot_json: normalized, change_note: form.change_note.trim() })
  }
  editorVisible.value = false
  ElMessage.success('能力版本草稿已保存')
  await loadVersions(selected.value)
}

const publish = async (version: PlatformAICapabilityVersion) => {
  const validation = validateSnapshot(version.snapshot_json)
  if (validation.errors.length) {
    ElMessage.error(`无法发布：${validation.errors.join('；')}`)
    return
  }
  const warningText = validation.warnings.length ? `\n风险提示：${validation.warnings.join('；')}。` : ''
  await ElMessageBox.confirm(
    `确认发布 V${version.version}？发布后快照不可修改，套餐只能引用这个固定版本。${warningText}`,
    '发布 AI 能力版本',
    { type: 'warning', confirmButtonText: '确认发布', cancelButtonText: '取消' },
  )
  await publishPlatformAICapabilityVersion(version.id)
  ElMessage.success('AI 能力版本已发布')
  if (selected.value) await loadVersions(selected.value)
  const result = await listPlatformAICapabilities()
  capabilities.value = result.list || []
}

onMounted(load)
</script>

<template>
  <section class="console-page ai-control-page" v-loading="loading">
    <header class="page-heading">
      <div><span class="eyebrow">CAPABILITY RELEASE</span><h1>AI 能力发布</h1><p>编排已治理的模型和 Agent 资产，发布不可变能力版本，再由套餐固定引用。</p></div>
      <el-button v-if="canManage && selected" type="primary" :loading="configurationLoading" @click="openEditor()">创建版本草稿</el-button>
    </header>

    <div class="release-layout">
      <aside class="surface-card capability-list">
        <button v-for="item in capabilities" :key="item.id" type="button" :class="{ active: selected?.id === item.id }" @click="loadVersions(item)">
          <span><strong>{{ item.name }}</strong><small>{{ item.capability_key }}</small></span>
          <el-tag size="small" :type="item.audience === 'candidate' ? 'success' : 'primary'">{{ audienceLabel(item.audience) }}</el-tag>
        </button>
      </aside>

      <main class="surface-card version-panel">
        <header v-if="selected"><div><h2>{{ selected.name }}</h2><p>{{ selected.description || '暂无能力说明' }} · {{ audienceLabel(selected.audience) }}</p></div><el-tag>{{ selected.status }}</el-tag></header>
        <section v-for="version in versions" :key="version.id" class="version-card">
          <div class="version-title"><div><strong>V{{ version.version }}</strong><small>{{ version.change_note || '暂无变更说明' }}</small></div><el-tag :type="version.status === 'published' ? 'success' : version.status === 'draft' ? 'warning' : 'info'">{{ version.status }}</el-tag></div>
          <div class="snapshot-summary">
            <div><span>LLM 模型池</span><strong>{{ snapshotStats(version.snapshot_json).llm }}</strong><small>默认 #{{ snapshotStats(version.snapshot_json).defaultLlm || '-' }}</small></div>
            <div><span>Embedding</span><strong>{{ snapshotStats(version.snapshot_json).embedding }}</strong><small>允许模型</small></div>
            <div><span>Agent / Prompt</span><strong>{{ snapshotStats(version.snapshot_json).agents }} / {{ snapshotStats(version.snapshot_json).prompts }}</strong><small>固定资产</small></div>
            <div><span>Skill / MCP</span><strong>{{ snapshotStats(version.snapshot_json).skills }} / {{ snapshotStats(version.snapshot_json).mcp }}</strong><small>固定版本与策略</small></div>
          </div>
          <el-alert v-if="validateSnapshot(version.snapshot_json).errors.length" :title="validateSnapshot(version.snapshot_json).errors.join('；')" type="error" :closable="false" show-icon />
          <details><summary>查看原始发布快照</summary><pre>{{ prettySnapshot(version.snapshot_json) }}</pre></details>
          <footer><span>快照 {{ version.snapshot_hash ? version.snapshot_hash.slice(0, 12) : '-' }} · 发布 {{ formatTime(version.published_at) }}</span><div><el-button v-if="canManage && version.status === 'draft'" link type="primary" @click="openEditor(version)">编辑草稿</el-button><el-button v-if="canPublish && version.status === 'draft'" link type="success" @click="publish(version)">发布并冻结</el-button></div></footer>
        </section>
        <el-empty v-if="selected && !versions.length" description="尚未创建能力版本" />
      </main>
    </div>

    <el-dialog v-model="editorVisible" :title="`${selected?.name || ''} · ${editingVersion ? `编辑 V${editingVersion.version}` : '新建版本'}`" width="760px">
      <el-alert title="模型池只应引用平台已启用的模型；发布后版本不可修改，企业与候选人的选择范围以该快照为准。" type="info" :closable="false" show-icon />
      <el-form class="editor-form" label-position="top">
        <el-form-item label="变更说明" required><el-input v-model="form.change_note" maxlength="500" show-word-limit /></el-form-item>
        <el-form-item label="允许调用的 LLM 模型" required><el-select v-model="form.allowed_llm_model_ids" multiple filterable class="full-width"><el-option v-for="item in options.llm" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item label="默认 LLM 模型" required><el-select v-model="form.default_llm_model_id" filterable class="full-width"><el-option v-for="item in options.llm.filter(option => form.allowed_llm_model_ids.includes(option.value))" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item label="允许调用的 Embedding 模型"><el-select v-model="form.allowed_embedding_model_ids" multiple filterable class="full-width"><el-option v-for="item in options.embedding" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item label="默认 Embedding 模型"><el-select v-model="form.default_embedding_model_id" clearable filterable class="full-width"><el-option v-for="item in options.embedding.filter(option => form.allowed_embedding_model_ids.includes(option.value))" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-divider content-position="left">固定配置引用</el-divider>
        <el-form-item label="Agent"><el-select v-model="form.agent_ids" multiple filterable class="full-width"><el-option v-for="item in options.agents" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item label="Prompt 模板"><el-select v-model="form.prompt_template_ids" multiple filterable class="full-width"><el-option v-for="item in options.prompts" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item label="Agent Skill 版本"><el-select v-model="form.agent_skill_version_ids" multiple filterable class="full-width"><el-option v-for="item in options.agentSkillVersions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item label="AI Skill 版本"><el-select v-model="form.ai_skill_version_ids" multiple filterable class="full-width"><el-option v-for="item in options.aiSkillVersions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item label="MCP 策略"><el-select v-model="form.mcp_policy_ids" multiple filterable class="full-width"><el-option v-for="item in options.mcpPolicies" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
      </el-form>
      <template #footer><el-button @click="editorVisible = false">取消</el-button><el-button type="primary" @click="submitDraft">保存草稿</el-button></template>
    </el-dialog>
  </section>
</template>

<style scoped>
.full-width{width:100%}
.ai-control-page{display:grid;gap:20px}.page-heading,.version-panel>header,.version-title,.version-card footer{display:flex;justify-content:space-between;align-items:flex-start;gap:20px}.page-heading h1,.version-panel h2{margin:0}.page-heading p,.version-panel header p{margin:7px 0 0;color:var(--el-text-color-secondary)}.eyebrow{font-size:12px;letter-spacing:.12em;color:var(--el-color-primary)}.release-layout{display:grid;grid-template-columns:300px minmax(0,1fr);gap:20px}.capability-list{padding:10px;height:max-content}.capability-list button{width:100%;display:flex;justify-content:space-between;align-items:center;gap:10px;padding:14px;border:0;border-radius:10px;background:transparent;text-align:left;color:inherit;cursor:pointer}.capability-list button:hover,.capability-list button.active{background:var(--el-fill-color-light)}.capability-list span,.version-title>div{display:grid;gap:4px}.capability-list small,.version-title small,.version-card footer{color:var(--el-text-color-secondary)}.version-panel{padding:22px}.version-card{display:grid;gap:14px;margin-top:16px;padding:18px;border:1px solid var(--el-border-color-lighter);border-radius:12px}.snapshot-summary{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:10px}.snapshot-summary>div{display:grid;gap:4px;padding:13px;border-radius:10px;background:var(--el-fill-color-lighter)}.snapshot-summary span,.snapshot-summary small{color:var(--el-text-color-secondary);font-size:12px}.snapshot-summary strong{font-size:20px}.version-card details summary{cursor:pointer;color:var(--el-color-primary);font-size:13px}.version-card pre{max-height:330px;overflow:auto;padding:14px;border-radius:9px;background:var(--el-fill-color-lighter);font-size:12px;line-height:1.55;white-space:pre-wrap;word-break:break-word}.version-card footer{font-size:12px;align-items:center}.editor-form{margin-top:18px}.editor-form :deep(textarea){font-family:ui-monospace,SFMono-Regular,Menlo,monospace}@media(max-width:1100px){.snapshot-summary{grid-template-columns:repeat(2,1fr)}}@media(max-width:900px){.release-layout{grid-template-columns:1fr}.page-heading{flex-direction:column}}@media(max-width:600px){.snapshot-summary{grid-template-columns:1fr}}
</style>
