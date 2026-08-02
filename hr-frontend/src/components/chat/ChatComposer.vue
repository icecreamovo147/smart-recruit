<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Box, Close, Position } from '@element-plus/icons-vue'
import type { Session, ContextUsageInfo } from '@/types/ai'
import type { LlmModel } from '@shared/types/llm'
import type { CapabilityInfo } from '@shared/types/agent'
import type { AvailableAgentSkill } from '@shared/types/agentSkill'
import {
  contextBudgetRatio,
  contextBudgetSeverity,
  contextUsageSourceLabel,
  contextUsageStageLabel,
  contextWindowProgressRatio,
  contextWindowRatio,
  contextWindowTokens,
  currentEffectiveContextTokens,
  formatCompactTokens,
  inputBudgetTokens,
} from '@/utils/contextUsage'

const hasEstimatedBreakdownDetails = computed(() => {
  const breakdown = visibleContextUsage.value?.breakdown
  if (!breakdown) return false
  return [
    breakdown.summary_tokens,
    breakdown.memory_tokens,
    breakdown.current_message_tokens,
    breakdown.skill_tokens,
    breakdown.tool_result_tokens,
    breakdown.tool_schema_tokens,
    breakdown.protocol_overhead_tokens,
  ].some((tokens) => Number(tokens) > 0)
})

const props = defineProps<{
  input: string
  loading: boolean
  streaming: boolean
  modelList: LlmModel[]
  selectedModelId: number | null
  contextUsage: ContextUsageInfo | null
  contextPreviewing: boolean
  dataSource: string
  currentSession: Session | null
  capabilities: CapabilityInfo[]
  selectedCapabilityKeys: string[]
  agentSkills: AvailableAgentSkill[]
  selectedAgentSkillVersionIds: number[]
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:input', value: string): void
  (e: 'update:selectedModelId', value: number | null): void
  (e: 'update:selectedCapabilityKeys', value: string[]): void
  (e: 'update:selectedAgentSkillVersionIds', value: number[]): void
  (e: 'submit'): void
  (e: 'stop'): void
}>()

const placeholderText = computed(() => {
  if (props.currentSession?.application_id) {
    return '例如：他的项目经历和岗位要求匹配吗？也可以说"通过这个候选人"'
  }
  return '例如：今天后端岗位投递了多少人？'
})

const activeSlashIndex = computed(() => {
  const value = props.input
  const slashIndex = value.lastIndexOf('/')
  const chineseSlashIndex = value.lastIndexOf('、')
  const index = Math.max(slashIndex, chineseSlashIndex)
  if (index < 0) return -1
  const token = value.slice(index)
  if (/\s/.test(token)) return -1
  return index
})

const slashQuery = computed(() => {
  if (activeSlashIndex.value < 0) return ''
  return props.input.slice(activeSlashIndex.value + 1).trim().toLowerCase()
})

const selectionMenuVisible = computed(() =>
  !props.disabled && !props.streaming && activeSlashIndex.value >= 0,
)

const filteredAgentSkills = computed(() => {
  const query = slashQuery.value
  const selectable = props.agentSkills.filter((skill) => Boolean(skill.current_version?.version_id))
  if (!query) return selectable
  return selectable.filter((skill) => {
    const haystack = [
      skill.display_name,
      skill.name,
      skill.description,
      skill.current_version?.version,
      skill.current_version?.category,
      skill.current_version?.scenario,
    ].filter(Boolean).join(' ').toLowerCase()
    return haystack.includes(query)
  })
})

const highlightedSkillIndex = ref(0)

watch(
  [selectionMenuVisible, slashQuery, () => filteredAgentSkills.value.length],
  () => {
    highlightedSkillIndex.value = 0
  },
)

const selectedCapabilities = computed(() =>
  props.selectedCapabilityKeys
    .map((key) => props.capabilities.find((cap) => cap.key === key))
    .filter((cap): cap is CapabilityInfo => Boolean(cap)),
)

const selectedAgentSkills = computed(() =>
  props.selectedAgentSkillVersionIds
    .map((versionId) => props.agentSkills.find(
      (skill) => skill.current_version?.version_id === versionId,
    ))
    .filter((skill): skill is AvailableAgentSkill => Boolean(skill)),
)

const capabilityLabel = (cap: CapabilityInfo) => cap.display_name || cap.name || cap.key
const agentSkillLabel = (skill: AvailableAgentSkill) => skill.display_name || skill.name
const agentSkillDescription = (skill: AvailableAgentSkill): string => {
  const label = agentSkillLabel(skill).trim()
  const description = skill.description?.trim() || ''
  return description && description !== label ? description : ''
}
const agentSkillVersionId = (skill: AvailableAgentSkill): number =>
  Number(skill.current_version?.version_id) || 0
const agentSkillMeta = (skill: AvailableAgentSkill): string => {
  const version = skill.current_version
  if (!version) return '不可用'
  const role = version.composition_role === 'supporting' ? '辅助技能' : '主技能'
  const risk = {
    low: '低风险',
    medium: '中风险',
    high: '高风险',
    critical: '极高风险',
  }[version.risk] || '低风险'
  return [
    `v${version.version}`,
    role,
    risk,
    `${version.core_estimated_tokens} Tokens`,
  ].join(' · ')
}

const clearSlashToken = () => {
  const index = activeSlashIndex.value
  if (index < 0) return
  const after = props.input.slice(index).match(/^\S*/)?.[0] || ''
  const nextInput = `${props.input.slice(0, index)}${props.input.slice(index + after.length).replace(/^\s+/, '')}`
  emit('update:input', nextInput)
}

const selectAgentSkill = (skill: AvailableAgentSkill) => {
  if (props.disabled) return
  const versionId = agentSkillVersionId(skill)
  const selected = selectedAgentSkills.value
  const role = skill.current_version?.composition_role === 'supporting' ? 'supporting' : 'primary'
  const roleAlreadySelected = selected.some(
    (item) => (item.current_version?.composition_role === 'supporting' ? 'supporting' : 'primary') === role,
  )
  const supportingWithoutPrimary = role === 'supporting' && !selected.some(
    (item) => item.current_version?.composition_role !== 'supporting',
  )
  if (
    versionId > 0
    && selected.length < 2
    && !roleAlreadySelected
    && !supportingWithoutPrimary
    && !props.selectedAgentSkillVersionIds.includes(versionId)
  ) {
    emit('update:selectedAgentSkillVersionIds', [...props.selectedAgentSkillVersionIds, versionId])
  }
  clearSlashToken()
}

const handleComposerKeydown = (event: KeyboardEvent) => {
  if (selectionMenuVisible.value) {
    const skillCount = filteredAgentSkills.value.length
    if (event.key === 'ArrowDown' && skillCount > 0) {
      event.preventDefault()
      highlightedSkillIndex.value = (highlightedSkillIndex.value + 1) % skillCount
      return
    }
    if (event.key === 'ArrowUp' && skillCount > 0) {
      event.preventDefault()
      highlightedSkillIndex.value = (highlightedSkillIndex.value - 1 + skillCount) % skillCount
      return
    }
    if (event.key === 'Enter') {
      event.preventDefault()
      const highlightedSkill = filteredAgentSkills.value[highlightedSkillIndex.value]
      if (highlightedSkill) selectAgentSkill(highlightedSkill)
      return
    }
    if (event.key === 'Escape') {
      event.preventDefault()
      clearSlashToken()
      return
    }
  }

  if (
    event.key === 'Enter'
    && !event.shiftKey
    && !event.altKey
    && !event.ctrlKey
    && !event.metaKey
  ) {
    event.preventDefault()
    if (!props.streaming && !props.contextPreviewing && !props.disabled) emit('submit')
  }
}

const removeCapability = (key: string) => {
  if (props.disabled) return
  emit('update:selectedCapabilityKeys', props.selectedCapabilityKeys.filter((item) => item !== key))
}

const removeAgentSkill = (versionId: number) => {
  if (props.disabled) return
  emit(
    'update:selectedAgentSkillVersionIds',
    props.selectedAgentSkillVersionIds.filter((item) => item !== versionId),
  )
}

const selectedModel = computed(() => props.selectedModelId == null
  ? props.modelList.find((model) => model.is_default) || props.modelList.find((model) => model.is_enabled)
  : props.modelList.find((model) => model.id === props.selectedModelId))
const contextUsageMatchesSelection = computed(() => {
  if (!props.contextUsage) return false
  if (!selectedModel.value) return true
  return props.contextUsage.model_id === selectedModel.value.id
})
const visibleContextUsage = computed(() => props.contextPreviewing || !contextUsageMatchesSelection.value
  ? null
  : props.contextUsage)
const sessionTokensUsed = computed(() => currentEffectiveContextTokens(visibleContextUsage.value))
const totalContextWindow = computed(() => contextWindowTokens(visibleContextUsage.value)
  || selectedModel.value?.context_window_tokens)
const windowRatio = computed(() => contextWindowRatio(visibleContextUsage.value))
const windowProgressRatio = computed(() => contextWindowProgressRatio(visibleContextUsage.value))
const availableInputBudget = computed(() => inputBudgetTokens(visibleContextUsage.value))
const budgetRatio = computed(() => contextBudgetRatio(visibleContextUsage.value))
const hasKnownBudget = computed(() => availableInputBudget.value != null)
const unknownConfiguration = computed(() =>
  Boolean(visibleContextUsage.value && (!hasKnownBudget.value || visibleContextUsage.value.budget_status === 'unknown_config')),
)
const windowUsagePercent = computed(() => windowRatio.value == null
  ? '无法计算'
  : `${Number((windowRatio.value * 100).toFixed(1))}%`)
const windowUsageLabel = computed(() => windowRatio.value == null
  ? windowUsagePercent.value
  : `${windowUsagePercent.value} 已用`)
const budgetUsagePercent = computed(() => budgetRatio.value == null
  ? '无法计算'
  : `${(budgetRatio.value * 100).toFixed(1)}%`)
const contextIndicatorLabel = computed(() => {
  if (props.contextPreviewing) return `Context 计算中 / ${formatCompactTokens(totalContextWindow.value)}`
  if (!visibleContextUsage.value) return `Context — / ${formatCompactTokens(totalContextWindow.value)}`
  return `Context ${formatCompactTokens(sessionTokensUsed.value)} / ${formatCompactTokens(totalContextWindow.value)}${windowRatio.value == null ? '' : `，${windowUsageLabel.value}`}`
})

const contextUsageSeverityClass = computed(() => {
  return `chat-composer__context-indicator--${contextBudgetSeverity(visibleContextUsage.value)}`
})

const contextUsageRatioClass = computed(() => {
  const severity = contextBudgetSeverity(visibleContextUsage.value)
  if (severity === 'danger' || severity === 'warning') return 'context-usage-popover__value--warning'
  if (severity === 'caution') return 'context-usage-popover__value--caution'
  return ''
})

const positive = (value: number | undefined): boolean => Number.isFinite(value) && Number(value) > 0

</script>

<template>
  <div class="chat-composer" :class="{ 'chat-composer--disabled': disabled }">
    <Transition name="chat-composer-skill-menu">
      <div
        v-if="selectionMenuVisible"
        class="chat-composer__skill-menu"
        role="listbox"
        aria-label="可用 Agent Skills"
      >
        <button
          v-for="(skill, index) in filteredAgentSkills"
          :key="`agent-version-${agentSkillVersionId(skill)}`"
          type="button"
          class="chat-composer__skill-option"
          :class="{
            'chat-composer__skill-option--highlighted': highlightedSkillIndex === index,
            'chat-composer__skill-option--selected': selectedAgentSkillVersionIds.includes(agentSkillVersionId(skill)),
          }"
          role="option"
          :aria-selected="selectedAgentSkillVersionIds.includes(agentSkillVersionId(skill))"
          :title="agentSkillMeta(skill)"
          @mouseenter="highlightedSkillIndex = index"
          @click="selectAgentSkill(skill)"
        >
          <el-icon class="chat-composer__skill-icon"><Box /></el-icon>
          <span class="chat-composer__skill-content">
            <strong class="chat-composer__skill-name">{{ agentSkillLabel(skill) }}</strong>
            <span v-if="agentSkillDescription(skill)" class="chat-composer__skill-description">
              {{ agentSkillDescription(skill) }}
            </span>
          </span>
          <span class="chat-composer__skill-source">smart-recruit</span>
        </button>
        <div v-if="filteredAgentSkills.length === 0" class="chat-composer__skill-empty">
          暂无匹配的 Skill
        </div>
      </div>
    </Transition>
    <div
      v-if="selectedCapabilities.length > 0 || selectedAgentSkills.length > 0"
      class="chat-composer__selected-skills"
    >
      <button
        v-for="capability in selectedCapabilities"
        :key="capability.key"
        type="button"
        class="chat-composer__skill-badge"
        :disabled="disabled"
        @click="removeCapability(capability.key)"
      >
        <span>/{{ capabilityLabel(capability) }}</span>
        <el-icon class="chat-composer__skill-close"><Close /></el-icon>
      </button>
      <button
        v-for="skill in selectedAgentSkills"
        :key="agentSkillVersionId(skill)"
        type="button"
        class="chat-composer__skill-badge"
        :disabled="disabled"
        @click="removeAgentSkill(agentSkillVersionId(skill))"
      >
        <span>/{{ agentSkillLabel(skill) }} v{{ skill.current_version?.version }}</span>
        <el-icon class="chat-composer__skill-close"><Close /></el-icon>
      </button>
    </div>
    <div class="chat-composer__input-area">
      <el-input
        :model-value="input"
        :disabled="streaming || disabled"
        :placeholder="placeholderText"
        type="textarea"
        :autosize="{ minRows: 2, maxRows: 6 }"
        resize="none"
        class="chat-composer__text-input"
        @keydown="handleComposerKeydown"
        @update:model-value="(val: string) => emit('update:input', val)"
      />
    </div>
    <div class="chat-composer__toolbar">
      <div class="chat-composer__toolbar-left">

        <el-select
          v-if="modelList.length > 0"
          :model-value="selectedModelId"
          size="small"
          placeholder="默认模型"
          class="chat-composer__model-select"
          clearable
          :disabled="streaming || contextPreviewing || disabled"
          @update:model-value="(val: number | null) => emit('update:selectedModelId', val)"
        >
          <el-option
            v-for="m in modelList"
            :key="m.id"
            :value="m.id"
            :label="m.display_name || m.model_name"
          />
        </el-select>
        <el-popover
          placement="top"
          trigger="hover"
          :width="340"
          popper-class="context-usage-popover"
          :disabled="!visibleContextUsage || contextPreviewing"
        >
          <template #reference>
            <button
              type="button"
              class="chat-composer__context-indicator"
              :class="contextUsageSeverityClass"
              :aria-label="contextIndicatorLabel"
              :title="contextIndicatorLabel"
            >
              <span class="chat-composer__context-label">Context</span>
              <span v-if="contextPreviewing" class="chat-composer__context-unknown">
                计算中 / {{ formatCompactTokens(totalContextWindow) }}
              </span>
              <span v-else-if="visibleContextUsage" class="chat-composer__context-value">
                {{ formatCompactTokens(sessionTokensUsed) }} / {{ formatCompactTokens(totalContextWindow) }}
              </span>
              <span v-else class="chat-composer__context-unknown">— / {{ formatCompactTokens(totalContextWindow) }}</span>
            </button>
          </template>
          <template v-if="visibleContextUsage">
            <div class="context-usage-popover__header">
              <span class="context-usage-popover__title">上下文窗口</span>
              <span class="context-usage-popover__badge">{{ windowUsageLabel }}</span>
            </div>
            <div v-if="unknownConfiguration" class="context-usage-popover__notice" role="status">
              模型上下文窗口未配置，已使用摘要 + 最近消息的安全降级策略。
            </div>
            <div class="context-usage-popover__grid">
              <div class="context-usage-popover__item">
                <span class="context-usage-popover__label">模型</span>
                <span class="context-usage-popover__value">{{ visibleContextUsage.model_name || '-' }}</span>
              </div>
              <div class="context-usage-popover__item">
                <span class="context-usage-popover__label">总上下文窗口</span>
                <span class="context-usage-popover__value">{{ positive(visibleContextUsage.context_window_tokens) ? formatCompactTokens(visibleContextUsage.context_window_tokens) : '未配置' }}</span>
              </div>
              <div class="context-usage-popover__item">
                <span class="context-usage-popover__label">最大输出预留</span>
                <span class="context-usage-popover__value">{{ positive(visibleContextUsage.max_output_tokens) ? formatCompactTokens(visibleContextUsage.max_output_tokens) : '—' }}</span>
              </div>
              <div class="context-usage-popover__item">
                <span class="context-usage-popover__label">安全余量</span>
                <span class="context-usage-popover__value">{{ positive(visibleContextUsage.safety_margin_tokens) ? formatCompactTokens(visibleContextUsage.safety_margin_tokens) : '—' }}</span>
              </div>
              <div class="context-usage-popover__item">
                <span class="context-usage-popover__label">可用输入预算</span>
                <span class="context-usage-popover__value">{{ formatCompactTokens(availableInputBudget) }}</span>
              </div>
              <div class="context-usage-popover__item">
                <span class="context-usage-popover__label">当前会话已用</span>
                <span class="context-usage-popover__value">{{ formatCompactTokens(sessionTokensUsed) }}</span>
              </div>
              <div class="context-usage-popover__item">
                <span class="context-usage-popover__label">窗口占用</span>
                <span class="context-usage-popover__value" :class="contextUsageRatioClass">
                  {{ windowUsagePercent }}
                </span>
              </div>
              <div class="context-usage-popover__item">
                <span class="context-usage-popover__label">安全预算占用</span>
                <span class="context-usage-popover__value" :class="contextUsageRatioClass">
                  {{ budgetUsagePercent }}
                </span>
              </div>
            </div>
            <div v-if="hasKnownBudget" class="context-usage-popover__progress" aria-hidden="true">
              <span :style="{ width: `${windowProgressRatio * 100}%` }"></span>
            </div>
            <template v-if="hasEstimatedBreakdownDetails && visibleContextUsage.breakdown">
              <div class="context-usage-popover__section-title">细分（估算）</div>
              <div class="context-usage-popover__breakdown">
                <div v-if="positive(visibleContextUsage.breakdown.system_prompt_tokens)" class="context-usage-popover__breakdown-item">
                  <span>系统指令</span><span>{{ formatCompactTokens(visibleContextUsage.breakdown.system_prompt_tokens) }}</span>
                </div>
                <div v-if="positive(visibleContextUsage.breakdown.recent_message_tokens)" class="context-usage-popover__breakdown-item">
                  <span>最近消息</span><span>{{ formatCompactTokens(visibleContextUsage.breakdown.recent_message_tokens) }}</span>
                </div>
                <div v-if="positive(visibleContextUsage.breakdown.summary_tokens)" class="context-usage-popover__breakdown-item">
                  <span>会话摘要</span><span>{{ formatCompactTokens(visibleContextUsage.breakdown.summary_tokens) }}</span>
                </div>
                <div v-if="positive(visibleContextUsage.breakdown.memory_tokens)" class="context-usage-popover__breakdown-item">
                  <span>长期记忆</span><span>{{ formatCompactTokens(visibleContextUsage.breakdown.memory_tokens) }}</span>
                </div>
                <div v-if="positive(visibleContextUsage.breakdown.current_message_tokens)" class="context-usage-popover__breakdown-item">
                  <span>当前消息</span><span>{{ formatCompactTokens(visibleContextUsage.breakdown.current_message_tokens) }}</span>
                </div>
                <div v-if="positive(visibleContextUsage.breakdown.skill_tokens)" class="context-usage-popover__breakdown-item">
                  <span>技能指令</span><span>{{ formatCompactTokens(visibleContextUsage.breakdown.skill_tokens) }}</span>
                </div>
                <div v-if="positive(visibleContextUsage.breakdown.tool_schema_tokens)" class="context-usage-popover__breakdown-item">
                  <span>工具定义</span><span>{{ formatCompactTokens(visibleContextUsage.breakdown.tool_schema_tokens) }}</span>
                </div>
                <div v-if="positive(visibleContextUsage.breakdown.tool_result_tokens)" class="context-usage-popover__breakdown-item">
                  <span>工具结果</span><span>{{ formatCompactTokens(visibleContextUsage.breakdown.tool_result_tokens) }}</span>
                </div>
                <div v-if="positive(visibleContextUsage.breakdown.protocol_overhead_tokens)" class="context-usage-popover__breakdown-item">
                  <span>协议开销</span><span>{{ formatCompactTokens(visibleContextUsage.breakdown.protocol_overhead_tokens) }}</span>
                </div>
              </div>
            </template>
            <div v-if="visibleContextUsage.included_message_count != null || visibleContextUsage.omitted_message_count != null || visibleContextUsage.summary_applied || visibleContextUsage.memory_applied" class="context-usage-popover__meta">
              <span v-if="visibleContextUsage.included_message_count != null">纳入 {{ visibleContextUsage.included_message_count }} 条消息</span>
              <span v-if="visibleContextUsage.omitted_message_count != null">省略 {{ visibleContextUsage.omitted_message_count }} 条消息</span>
              <span v-if="visibleContextUsage.summary_applied">已应用会话摘要</span>
              <span v-if="visibleContextUsage.memory_applied">已应用长期记忆</span>
            </div>
            <div class="context-usage-popover__footer">
              <span>{{ contextUsageStageLabel(visibleContextUsage.stage) }}</span>
              <span>{{ contextUsageSourceLabel(visibleContextUsage) }}</span>
            </div>
          </template>
        </el-popover>
        <div class="chat-composer__datasource">
          <span class="chat-composer__datasource-label">数据来源：</span>
          <span class="chat-composer__datasource-value">{{ dataSource }}</span>
        </div>
      </div>
      <el-button
        v-if="streaming"
        type="danger"
        plain
        class="chat-composer__send-btn"
        @click="emit('stop')"
      >
        中断
      </el-button>
      <el-button
        v-else
        type="primary"
        :icon="Position"
        :loading="loading"
        :disabled="!input.trim() || contextPreviewing || disabled"
        class="chat-composer__send-btn"
        @click="emit('submit')"
      >
        发送
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.chat-composer {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 16px;
  padding: 12px 14px;
  position: relative;
  min-height: 120px;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.05);
  transition:
    border-color var(--motion-normal) var(--motion-ease),
    box-shadow var(--motion-normal) var(--motion-ease);
}

.chat-composer:hover,
.chat-composer:focus-within {
  border-color: var(--el-color-primary-light-7);
  box-shadow: 0 10px 28px rgba(37, 99, 235, 0.08);
}

.chat-composer--disabled:hover,
.chat-composer--disabled:focus-within {
  border-color: var(--border);
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.05);
}

.chat-composer__skill-menu {
  position: absolute;
  left: 0;
  right: 0;
  bottom: calc(100% + 10px);
  z-index: 10;
  width: auto;
  max-height: 240px;
  box-sizing: border-box;
  overflow: auto;
  padding: 5px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 14px;
  box-shadow:
    0 14px 36px rgba(15, 23, 42, 0.11),
    0 2px 6px rgba(15, 23, 42, 0.04);
}

.chat-composer-skill-menu-enter-active,
.chat-composer-skill-menu-leave-active {
  transition:
    opacity var(--motion-normal) var(--motion-ease),
    transform var(--motion-normal) var(--motion-ease);
  transform-origin: bottom center;
}

.chat-composer-skill-menu-enter-from,
.chat-composer-skill-menu-leave-to {
  opacity: 0;
  transform: translateY(6px) scale(0.98);
}

.chat-composer-skill-menu-enter-to,
.chat-composer-skill-menu-leave-from {
  opacity: 1;
  transform: translateY(0) scale(1);
}

.chat-composer__skill-option {
  width: 100%;
  min-height: 42px;
  border: 0;
  border-radius: 10px;
  background: transparent;
  color: var(--text-primary);
  display: grid;
  grid-template-columns: 22px minmax(0, 1fr) auto;
  align-items: center;
  gap: 9px;
  padding: 7px 10px;
  cursor: pointer;
  text-align: left;
  transition:
    background-color var(--motion-fast) var(--motion-ease),
    color var(--motion-fast) var(--motion-ease);
}

.chat-composer__skill-option:hover,
.chat-composer__skill-option--highlighted,
.chat-composer__skill-option--selected {
  background: var(--surface-muted);
}

.chat-composer__skill-icon {
  width: 18px;
  height: 18px;
  color: var(--text-secondary);
  font-size: 17px;
}

.chat-composer__skill-content {
  display: flex;
  min-width: 0;
  align-items: baseline;
  gap: 9px;
}

.chat-composer__skill-name {
  flex: 0 0 auto;
  max-width: 38%;
  overflow: hidden;
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-composer__skill-description {
  min-width: 0;
  overflow: hidden;
  color: var(--text-muted);
  font-size: 12px;
  font-weight: 400;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-composer__skill-source {
  flex-shrink: 0;
  color: var(--text-faint);
  font-size: 11px;
  font-weight: 400;
  white-space: nowrap;
}

.chat-composer__skill-option--selected .chat-composer__skill-source {
  color: var(--text-muted);
}

.chat-composer__skill-empty {
  padding: 14px 12px;
  color: var(--text-faint);
  font-size: 13px;
  text-align: center;
}

.chat-composer__selected-skills {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 8px;
}

.chat-composer__skill-badge {
  min-width: 0;
  height: 28px;
  border: 1px solid rgba(183, 110, 0, 0.12);
  border-radius: 999px;
  background: #fff4e5;
  color: #b76e00;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.chat-composer__skill-badge span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-composer__skill-close {
  width: 13px;
  font-size: 11px;
  opacity: 0.58;
  transition: opacity var(--motion-fast) var(--motion-ease);
}

.chat-composer__skill-badge:hover .chat-composer__skill-close {
  opacity: 1;
}

.chat-composer__input-area {
  min-height: 54px;
}

.chat-composer__text-input {
  width: 100%;
}

.chat-composer__text-input :deep(.el-textarea__inner) {
  box-shadow: none;
  background: transparent;
  padding: 0;
  font-size: 14px;
  line-height: 1.65;
  color: var(--text-primary);
  border: 0;
  min-height: 48px !important;
}

.chat-composer__text-input :deep(.el-textarea__inner::placeholder) {
  color: var(--text-faint);
}

.chat-composer__send-btn {
  flex-shrink: 0;
  min-width: 68px;
  height: 34px;
  border-radius: 10px;
  padding: 0 12px;
}

.chat-composer__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 8px;
}

.chat-composer__toolbar-left {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.chat-composer__context-indicator {
  border: 0;
  font-family: inherit;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  font-size: 12px;
  cursor: pointer;
  user-select: none;
  transition: background var(--motion-fast) var(--motion-ease);
}

.chat-composer__context-indicator:focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: 2px;
}

.chat-composer__context-indicator--normal {
  background: rgba(52, 199, 89, 0.10);
  color: #34c759;
}

.chat-composer__context-indicator--caution {
  background: rgba(255, 204, 0, 0.12);
  color: #b8860b;
}

.chat-composer__context-indicator--warning {
  background: rgba(255, 149, 0, 0.12);
  color: #c66a00;
}

.chat-composer__context-indicator--danger {
  background: rgba(255, 69, 58, 0.10);
  color: #ff453a;
}

.chat-composer__context-indicator--unknown {
  background: rgba(142, 142, 147, 0.10);
  color: var(--text-faint);
}

.chat-composer__context-label {
  font-weight: 500;
}

.chat-composer__context-value {
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.chat-composer__context-unknown {
  font-weight: 400;
  opacity: 0.6;
}

.chat-composer__context-estimated {
  font-size: 10px;
  opacity: 0.7;
  font-weight: 400;
}

:global(.context-usage-popover) {
  padding: 12px 16px;
}

:global(.context-usage-popover__header) {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  font-weight: 600;
  font-size: 14px;
}

:global(.context-usage-popover__badge) {
  font-size: 10px;
  font-weight: 500;
  padding: 1px 6px;
  border-radius: 4px;
  background: rgba(255, 204, 0, 0.15);
  color: #b8860b;
}

:global(.context-usage-popover__notice) {
  margin-bottom: 10px;
  padding: 8px 10px;
  border-radius: 8px;
  background: var(--surface-muted);
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.5;
}

:global(.context-usage-popover__grid) {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 6px 16px;
  margin-bottom: 10px;
}

:global(.context-usage-popover__item) {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

:global(.context-usage-popover__label) {
  font-size: 11px;
  color: var(--text-faint);
}

:global(.context-usage-popover__value) {
  font-size: 13px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

:global(.context-usage-popover__value--warning) {
  color: #ff453a;
}

:global(.context-usage-popover__value--caution) {
  color: #b8860b;
}

:global(.context-usage-popover__progress) {
  height: 4px;
  overflow: hidden;
  margin: 2px 0 10px;
  border-radius: 999px;
  background: var(--surface-muted);
}

:global(.context-usage-popover__progress span) {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: currentColor;
  transition: width var(--motion-normal) var(--motion-ease);
}

:global(.context-usage-popover__section-title) {
  font-size: 12px;
  font-weight: 600;
  margin-bottom: 6px;
  color: var(--text-secondary);
}

:global(.context-usage-popover__breakdown) {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px 16px;
  margin-bottom: 8px;
}

:global(.context-usage-popover__breakdown-item) {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
}

:global(.context-usage-popover__breakdown-item span:last-child) {
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

:global(.context-usage-popover__meta) {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 10px;
  color: var(--text-muted);
  font-size: 11px;
}

:global(.context-usage-popover__footer) {
  display: flex;
  justify-content: space-between;
  font-size: 10px;
  color: var(--text-faint);
  border-top: 1px solid var(--border);
  padding-top: 6px;
  margin-top: 4px;
}

.chat-composer__model-select {
  width: 150px;
  flex-shrink: 0;
}

.chat-composer__model-select :deep(.el-select__wrapper) {
  min-height: 32px;
  border-radius: 999px;
  background: var(--surface-muted);
  box-shadow: 0 0 0 1px var(--border) inset;
}

.chat-composer__datasource {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 0;
  color: var(--text-muted);
}

.chat-composer__datasource-label {
  font-size: 12px;
  color: var(--text-faint);
  white-space: nowrap;
}

.chat-composer__datasource-value {
  min-width: 0;
  font-size: 12px;
  color: var(--text-secondary);
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:global(:root[data-theme='dark']) .chat-composer__skill-badge {
  border-color: rgba(245, 158, 11, 0.18);
  background: rgba(245, 158, 11, 0.14);
  color: #f8c471;
}

@media (max-width: 768px) {
  .chat-composer {
    padding: 10px 12px 12px;
  }

  .chat-composer__skill-menu {
    max-height: 220px;
    border-radius: 12px;
  }

  .chat-composer__skill-option {
    grid-template-columns: 22px minmax(0, 1fr);
    padding: 7px 9px;
  }

  .chat-composer__skill-content {
    display: grid;
    gap: 2px;
  }

  .chat-composer__skill-name { max-width: 100%; }
  .chat-composer__skill-source { display: none; }

  .chat-composer__toolbar {
    align-items: flex-end;
    gap: 8px;
  }

  .chat-composer__toolbar-left {
    flex-wrap: wrap;
    gap: 8px;
  }

  .chat-composer__model-select {
    width: 140px;
  }

  .chat-composer__datasource {
    order: 3;
    width: 100%;
  }
}
</style>
