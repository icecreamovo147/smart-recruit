<script setup lang="ts">
import { computed } from 'vue'
import type {
  AgentRunResultMetadata,
  AgentSkillRuntimeEvidence,
  AgentSkillSectionRuntimeEvidence,
} from '@shared/types/agentRun'

const props = defineProps<{
  metadata?: AgentRunResultMetadata | null
}>()

const evidenceItems = computed<AgentSkillRuntimeEvidence[]>(() =>
  Array.isArray(props.metadata?.agent_skill_runtime_evidence)
    ? props.metadata.agent_skill_runtime_evidence.filter((item) => Number(item?.version_id) > 0)
    : [],
)

const contextUsage = computed(() => props.metadata?.context_usage || null)
const hasEvidence = computed(() => evidenceItems.value.length > 0 || Boolean(contextUsage.value))

const reasonLabels: Record<string, string> = {
  core_included: '核心指令已包含',
  core_budget_exceeded: '核心指令超出 Token 预算',
  section_included: 'Reference Section 已包含',
  section_budget_exceeded: 'Reference Section 超出 Token 预算',
  section_not_relevant: 'Reference Section 相关性不足',
  section_integrity_failed: 'Reference Section 完整性校验失败',
  package_integrity_failed: 'Package 完整性校验失败',
  package_invalid: 'Package 内容不合法',
  package_lookup_failed: 'Package 查询失败',
  package_store_unavailable: 'Package 存储不可用',
  version_unavailable: '指定版本不可用',
  skill_v2_disabled: 'Agent Skill Package v2 未启用',
  skill_disabled: 'Agent Skill 已停用',
  skill_limit_exceeded: 'Agent Skill 数量超过限制',
  manual_invocation_disabled: '不允许手动调用',
  manual_selection_invalid: '手动选择的版本不合法',
  manual_selection_limit_exceeded: '手动选择数量超过限制',
  composition_conflict: 'Package 组合冲突',
  composition_role_invalid: 'Package 组合角色不合法',
  composition_agent_type_mismatch: 'Package Agent 类型不匹配',
  composition_scenario_mismatch: 'Package 场景不匹配',
  primary_limit_exceeded: 'Primary Package 数量超过限制',
  supporting_limit_exceeded: 'Supporting Package 数量超过限制',
  supporting_requires_primary: 'Supporting Package 缺少 Primary',
  below_relevance_gate: '相关性低于自动召回门槛',
}

const roleLabels: Record<string, string> = {
  primary: 'Primary',
  supporting: 'Supporting',
}

const riskLabels: Record<string, string> = {
  low: '低风险',
  medium: '中风险',
  high: '高风险',
  critical: '严重风险',
}

const activationLabels: Record<string, string> = {
  auto: '自动启用',
  confirm: '确认后启用',
  manual_only: '仅手动启用',
}

const selectionLabels: Record<string, string> = {
  auto: '自动选择',
  manual: '手动选择',
  confirmed: '确认后选择',
}

const formatInteger = (value: unknown): string =>
  Math.max(0, Number(value) || 0).toLocaleString('zh-CN')

const formatScore = (value: unknown): string => {
  const score = Number(value)
  return value !== undefined && value !== null && value !== '' && Number.isFinite(score)
    ? score.toFixed(4)
    : '—'
}

const reasonText = (reason: string): string => {
  const normalized = String(reason || '').trim()
  if (!normalized) return '未记录'
  const label = reasonLabels[normalized]
  return label ? `${label}（${normalized}）` : normalized
}

const packageStatusType = (item: AgentSkillRuntimeEvidence): 'success' | 'danger' =>
  item.included ? 'success' : 'danger'

const sectionStatusType = (section: AgentSkillSectionRuntimeEvidence): 'success' | 'info' =>
  section.included ? 'success' : 'info'

const contextPromptTokens = computed(() => {
  const usage = contextUsage.value
  if (!usage) return 0
  return Number(usage.prompt_tokens_actual) > 0
    ? Number(usage.prompt_tokens_actual)
    : Number(usage.prompt_tokens_estimated) || 0
})

const contextPromptTokenKind = computed(() =>
  Number(contextUsage.value?.prompt_tokens_actual) > 0 ? '实际' : '预估',
)
</script>

<template>
  <section v-if="hasEvidence" class="skill-trace" data-testid="trace-skill-evidence">
    <div class="skill-trace__title">Agent Skill 运行证据</div>

    <div v-if="contextUsage" class="skill-trace__token-grid">
      <div class="skill-trace__metric">
        <span>上下文输入（{{ contextPromptTokenKind }}）</span>
        <strong>{{ formatInteger(contextPromptTokens) }} tokens</strong>
      </div>
      <div class="skill-trace__metric">
        <span>Skill Token</span>
        <strong>{{ formatInteger(contextUsage.breakdown?.skill_tokens) }} tokens</strong>
      </div>
      <div class="skill-trace__metric">
        <span>模型输出</span>
        <strong>{{ formatInteger(contextUsage.completion_tokens_actual) }} tokens</strong>
      </div>
      <div class="skill-trace__metric">
        <span>剩余上下文</span>
        <strong>{{ formatInteger(contextUsage.remaining_tokens_estimated) }} tokens</strong>
      </div>
    </div>

    <article
      v-for="evidence in evidenceItems"
      :key="`${evidence.version_id}-${evidence.compiled_hash}`"
      class="skill-package"
    >
      <header class="skill-package__header">
        <div>
          <strong>{{ evidence.display_name || evidence.skill_name || `Package #${evidence.skill_id}` }}</strong>
          <span class="skill-package__version">v{{ evidence.version || '—' }} · Version ID #{{ evidence.version_id }}</span>
        </div>
        <el-tag :type="packageStatusType(evidence)" size="small" effect="plain">
          {{ evidence.included ? 'Included' : 'Dropped' }}
        </el-tag>
      </header>

      <div class="skill-package__meta">
        <span>{{ roleLabels[evidence.composition_role] || evidence.composition_role || '角色未记录' }}</span>
        <span>{{ riskLabels[evidence.risk] || evidence.risk || '风险未记录' }}</span>
        <span>{{ activationLabels[evidence.activation_policy] || evidence.activation_policy || '启用策略未记录' }}</span>
        <span>{{ selectionLabels[evidence.selection_mode] || evidence.selection_mode || '选择方式未记录' }}</span>
      </div>

      <div class="skill-package__facts">
        <div>
          <span>Core Token</span>
          <strong>{{ formatInteger(evidence.core_estimated_tokens) }}</strong>
        </div>
        <div>
          <span>实际加载 Token</span>
          <strong>{{ formatInteger(evidence.loaded_tokens) }}</strong>
        </div>
        <div>
          <span>相关性模式</span>
          <strong>{{ evidence.relevance_mode || '未记录' }}</strong>
        </div>
        <div>
          <span>向量分</span>
          <strong>{{ formatScore(evidence.vector_score) }}</strong>
        </div>
        <div>
          <span>词法 / 元数据</span>
          <strong>{{ formatScore(evidence.lexical_score) }} / {{ formatScore(evidence.metadata_score) }}</strong>
        </div>
        <div>
          <span>相关性 / 最终分</span>
          <strong>{{ formatScore(evidence.relevance_score) }} / {{ formatScore(evidence.final_rank_score) }}</strong>
        </div>
      </div>

      <div class="skill-package__reason">
        <span>Package 决策</span>
        <strong>{{ reasonText(evidence.decision_reason) }}</strong>
      </div>
      <div class="skill-package__hash">
        <span>Compiled Hash</span>
        <code>{{ evidence.compiled_hash || '未记录' }}</code>
      </div>

      <div class="skill-sections">
        <div class="skill-sections__title">Reference Sections</div>
        <div v-if="!evidence.sections?.length" class="skill-sections__empty">没有 Reference Section 运行记录</div>
        <div
          v-for="section in evidence.sections || []"
          :key="`${evidence.version_id}-${section.section_id}`"
          class="skill-section"
          :class="{ 'skill-section--dropped': !section.included }"
        >
          <div class="skill-section__header">
            <strong>{{ section.section_key || `Section #${section.section_id}` }}</strong>
            <el-tag :type="sectionStatusType(section)" size="small" effect="plain">
              {{ section.included ? 'Included' : 'Dropped' }}
            </el-tag>
          </div>
          <div class="skill-section__details">
            <span>{{ formatInteger(section.estimated_tokens) }} tokens</span>
            <span>Rank {{ formatScore(section.final_rank_score) }}</span>
            <span>{{ reasonText(section.decision_reason) }}</span>
          </div>
          <code class="skill-section__hash">{{ section.content_hash || 'Hash 未记录' }}</code>
        </div>
      </div>
    </article>
  </section>
</template>

<style scoped>
.skill-trace {
  margin-top: 14px;
  border-top: 1px solid var(--el-border-color-lighter);
  padding-top: 14px;
}

.skill-trace__title,
.skill-sections__title {
  color: var(--el-text-color-primary);
  font-size: 13px;
  font-weight: 650;
}

.skill-trace__token-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
  margin-top: 10px;
}

.skill-trace__metric,
.skill-package__facts > div {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 8px 10px;
  background: var(--el-fill-color-extra-light);
}

.skill-trace__metric span,
.skill-package__facts span,
.skill-package__reason span,
.skill-package__hash span {
  display: block;
  color: var(--el-text-color-secondary);
  font-size: 11px;
  margin-bottom: 3px;
}

.skill-trace__metric strong,
.skill-package__facts strong {
  color: var(--el-text-color-primary);
  font-size: 12px;
}

.skill-package {
  margin-top: 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 12px;
  background: var(--el-bg-color);
}

.skill-package__header,
.skill-section__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.skill-package__header strong {
  display: block;
  color: var(--el-text-color-primary);
  font-size: 13px;
}

.skill-package__version {
  display: block;
  margin-top: 2px;
  color: var(--el-text-color-secondary);
  font-size: 11px;
}

.skill-package__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 12px;
  margin-top: 9px;
  color: var(--el-text-color-regular);
  font-size: 11px;
}

.skill-package__facts {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  margin-top: 10px;
}

.skill-package__reason,
.skill-package__hash {
  margin-top: 9px;
}

.skill-package__reason strong {
  color: var(--el-text-color-primary);
  font-size: 12px;
}

.skill-package__hash code,
.skill-section__hash {
  display: block;
  color: var(--el-text-color-secondary);
  font-family: var(--el-font-family-monospace, ui-monospace, SFMono-Regular, Menlo, monospace);
  font-size: 10px;
  overflow-wrap: anywhere;
}

.skill-sections {
  margin-top: 12px;
}

.skill-sections__empty {
  margin-top: 7px;
  color: var(--el-text-color-secondary);
  font-size: 11px;
}

.skill-section {
  margin-top: 7px;
  border-left: 3px solid var(--el-color-success-light-3);
  border-radius: 4px;
  padding: 8px 10px;
  background: var(--el-color-success-light-9);
}

.skill-section--dropped {
  border-left-color: var(--el-border-color);
  background: var(--el-fill-color-light);
}

.skill-section__header strong {
  color: var(--el-text-color-primary);
  font-size: 12px;
}

.skill-section__details {
  display: flex;
  flex-wrap: wrap;
  gap: 5px 12px;
  margin-top: 6px;
  color: var(--el-text-color-regular);
  font-size: 11px;
}

.skill-section__hash {
  margin-top: 5px;
}

@media (max-width: 720px) {
  .skill-trace__token-grid,
  .skill-package__facts {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
