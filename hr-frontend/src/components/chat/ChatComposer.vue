<script setup lang="ts">
import { computed } from 'vue'
import { Close, Position } from '@element-plus/icons-vue'
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
  skillCapabilities: CapabilityInfo[]
  selectedSkillKeys: string[]
  agentSkills: AvailableAgentSkill[]
  selectedAgentSkillIds: number[]
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:input', value: string): void
  (e: 'update:selectedModelId', value: number | null): void
  (e: 'update:selectedSkillKeys', value: string[]): void
  (e: 'update:selectedAgentSkillIds', value: number[]): void
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

const skillMenuVisible = computed(() =>
  !props.disabled && !props.streaming && activeSlashIndex.value >= 0,
)

const filteredSkillCapabilities = computed(() => {
  const query = slashQuery.value
  if (!query) return props.skillCapabilities
  return props.skillCapabilities.filter((cap) => {
    const haystack = [
      cap.display_name,
      cap.name,
      cap.key,
      cap.description,
      cap.skill_name,
    ].filter(Boolean).join(' ').toLowerCase()
    return haystack.includes(query)
  })
})

const filteredAgentSkills = computed(() => {
  const query = slashQuery.value
  if (!query) return props.agentSkills
  return props.agentSkills.filter((skill) => {
    const haystack = [
      skill.display_name,
      skill.name,
      skill.description,
      ...(skill.trigger_keywords || []),
    ].filter(Boolean).join(' ').toLowerCase()
    return haystack.includes(query)
  })
})

const selectedSkillCapabilities = computed(() =>
  props.selectedSkillKeys
    .map((key) => props.skillCapabilities.find((cap) => cap.key === key))
    .filter((cap): cap is CapabilityInfo => Boolean(cap)),
)

const selectedAgentSkills = computed(() =>
  props.selectedAgentSkillIds
    .map((id) => props.agentSkills.find((skill) => skill.id === id))
    .filter((skill): skill is AvailableAgentSkill => Boolean(skill)),
)

const skillLabel = (cap: CapabilityInfo) => cap.display_name || cap.name || cap.key
const agentSkillLabel = (skill: AvailableAgentSkill) => skill.display_name || skill.name

const clearSlashToken = () => {
  const index = activeSlashIndex.value
  if (index < 0) return
  const after = props.input.slice(index).match(/^\S*/)?.[0] || ''
  const nextInput = `${props.input.slice(0, index)}${props.input.slice(index + after.length).replace(/^\s+/, '')}`
  emit('update:input', nextInput)
}

const selectSkill = (cap: CapabilityInfo) => {
  if (props.disabled) return
  if (!props.selectedSkillKeys.includes(cap.key)) {
    emit('update:selectedSkillKeys', [...props.selectedSkillKeys, cap.key])
  }
  clearSlashToken()
}

const selectAgentSkill = (skill: AvailableAgentSkill) => {
  if (props.disabled) return
  if (!props.selectedAgentSkillIds.includes(skill.id)) {
    emit('update:selectedAgentSkillIds', [...props.selectedAgentSkillIds, skill.id])
  }
  clearSlashToken()
}

const removeSkill = (key: string) => {
  if (props.disabled) return
  emit('update:selectedSkillKeys', props.selectedSkillKeys.filter((item) => item !== key))
}

const removeAgentSkill = (id: number) => {
  if (props.disabled) return
  emit('update:selectedAgentSkillIds', props.selectedAgentSkillIds.filter((item) => item !== id))
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
      <div v-if="skillMenuVisible" class="chat-composer__skill-menu">
        <template v-if="skillCapabilities.length > 0">
          <button
            v-for="skill in filteredSkillCapabilities"
            :key="skill.key"
            type="button"
            class="chat-composer__skill-option"
            :class="{ 'chat-composer__skill-option--selected': selectedSkillKeys.includes(skill.key) }"
            @click="selectSkill(skill)"
          >
            <span class="chat-composer__skill-name">{{ skillLabel(skill) }}</span>
            <span class="chat-composer__skill-meta">{{ skill.runtime_type || 'skill' }}</span>
          </button>
        </template>
        <div v-if="skillCapabilities.length > 0 && filteredSkillCapabilities.length === 0" class="chat-composer__skill-empty">
          暂无匹配 Tool Skill
        </div>
        <button
          v-for="skill in filteredAgentSkills"
          :key="`agent-${skill.id}`"
          type="button"
          class="chat-composer__skill-option"
          :class="{ 'chat-composer__skill-option--selected': selectedAgentSkillIds.includes(skill.id) }"
          @click="selectAgentSkill(skill)"
        >
          <span class="chat-composer__skill-name">{{ agentSkillLabel(skill) }}</span>
          <span class="chat-composer__skill-meta">Skill</span>
        </button>
        <div v-if="filteredAgentSkills.length === 0" class="chat-composer__skill-empty">
          暂无匹配 Skill
        </div>
      </div>
    </Transition>
    <div
      v-if="selectedSkillCapabilities.length > 0 || selectedAgentSkills.length > 0"
      class="chat-composer__selected-skills"
    >
      <button
        v-for="skill in selectedSkillCapabilities"
        :key="skill.key"
        type="button"
        class="chat-composer__skill-badge"
        :disabled="disabled"
        @click="removeSkill(skill.key)"
      >
        <span>/{{ skillLabel(skill) }}</span>
        <el-icon class="chat-composer__skill-close"><Close /></el-icon>
      </button>
      <button
        v-for="skill in selectedAgentSkills"
        :key="skill.id"
        type="button"
        class="chat-composer__skill-badge"
        :disabled="disabled"
        @click="removeAgentSkill(skill.id)"
      >
        <span>/{{ agentSkillLabel(skill) }}</span>
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
        @keydown.enter.exact.prevent="streaming || contextPreviewing || disabled ? undefined : emit('submit')"
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
  bottom: calc(100% + 8px);
  z-index: 10;
  max-height: 240px;
  overflow: auto;
  padding: 6px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 8px;
  box-shadow: 0 16px 40px rgba(15, 23, 42, 0.16);
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
  min-height: 38px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 10px;
  cursor: pointer;
  text-align: left;
}

.chat-composer__skill-option:hover,
.chat-composer__skill-option--selected {
  background: var(--surface-muted);
}

.chat-composer__skill-name {
  font-size: 13px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-composer__skill-meta {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-faint);
  text-transform: uppercase;
}

.chat-composer__skill-empty {
  padding: 10px;
  color: var(--text-faint);
  font-size: 13px;
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
