<script setup lang="ts">
import { computed } from 'vue'
import { Close, Position } from '@element-plus/icons-vue'
import type { Session } from '@/types/ai'
import type { LlmModel } from '@/types/llm'
import type { CapabilityInfo } from '@/types/agent'
import type { AvailableAgentSkill } from '@/types/agentSkill'

const props = defineProps<{
  input: string
  loading: boolean
  streaming: boolean
  modelList: LlmModel[]
  selectedModelId: number | null
  dataSource: string
  currentSession: Session | null
  skillCapabilities: CapabilityInfo[]
  selectedSkillKeys: string[]
  agentSkills: AvailableAgentSkill[]
  selectedAgentSkillIds: number[]
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
  !props.streaming && activeSlashIndex.value >= 0,
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
  if (!props.selectedSkillKeys.includes(cap.key)) {
    emit('update:selectedSkillKeys', [...props.selectedSkillKeys, cap.key])
  }
  clearSlashToken()
}

const selectAgentSkill = (skill: AvailableAgentSkill) => {
  if (!props.selectedAgentSkillIds.includes(skill.id)) {
    emit('update:selectedAgentSkillIds', [...props.selectedAgentSkillIds, skill.id])
  }
  clearSlashToken()
}

const removeSkill = (key: string) => {
  emit('update:selectedSkillKeys', props.selectedSkillKeys.filter((item) => item !== key))
}

const removeAgentSkill = (id: number) => {
  emit('update:selectedAgentSkillIds', props.selectedAgentSkillIds.filter((item) => item !== id))
}

</script>

<template>
  <div class="chat-composer">
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
        @click="removeAgentSkill(skill.id)"
      >
        <span>/{{ agentSkillLabel(skill) }}</span>
        <el-icon class="chat-composer__skill-close"><Close /></el-icon>
      </button>
    </div>
    <div class="chat-composer__input-area">
      <el-input
        :model-value="input"
        :disabled="streaming"
        :placeholder="placeholderText"
        type="textarea"
        :autosize="{ minRows: 2, maxRows: 6 }"
        resize="none"
        class="chat-composer__text-input"
        @keydown.enter.exact.prevent="streaming ? undefined : emit('submit')"
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
          @update:model-value="(val: number | null) => emit('update:selectedModelId', val)"
        >
          <el-option
            v-for="m in modelList"
            :key="m.id"
            :value="m.id"
            :label="m.display_name || m.model_name"
          />
        </el-select>
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
        :disabled="!input.trim()"
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
