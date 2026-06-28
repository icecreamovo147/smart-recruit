<script setup lang="ts">
import { computed } from 'vue'
import type { Session } from '@/types/ai'
import type { LlmModel } from '@/types/llm'
import type { CapabilityInfo } from '@/types/agent'

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
}>()

const emit = defineEmits<{
  (e: 'update:input', value: string): void
  (e: 'update:selectedModelId', value: number | null): void
  (e: 'update:selectedSkillKeys', value: string[]): void
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
  const index = value.lastIndexOf('/')
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

const selectedSkillCapabilities = computed(() =>
  props.selectedSkillKeys
    .map((key) => props.skillCapabilities.find((cap) => cap.key === key))
    .filter((cap): cap is CapabilityInfo => Boolean(cap)),
)

const skillLabel = (cap: CapabilityInfo) => cap.display_name || cap.name || cap.key

const selectSkill = (cap: CapabilityInfo) => {
  if (!props.selectedSkillKeys.includes(cap.key)) {
    emit('update:selectedSkillKeys', [...props.selectedSkillKeys, cap.key])
  }
  const index = activeSlashIndex.value
  if (index >= 0) {
    const after = props.input.slice(index).match(/^\S*/)?.[0] || ''
    const nextInput = `${props.input.slice(0, index)}${props.input.slice(index + after.length).replace(/^\s+/, '')}`
    emit('update:input', nextInput)
  }
}

const removeSkill = (key: string) => {
  emit('update:selectedSkillKeys', props.selectedSkillKeys.filter((item) => item !== key))
}
</script>

<template>
  <div class="chat-composer">
    <div v-if="skillMenuVisible" class="chat-composer__skill-menu">
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
      <div v-if="filteredSkillCapabilities.length === 0" class="chat-composer__skill-empty">
        {{ skillCapabilities.length === 0 ? '暂无可用 Skill' : '暂无匹配 Skill' }}
      </div>
    </div>
    <div class="chat-composer__input-row">
      <el-input
        :model-value="input"
        :disabled="streaming"
        :placeholder="placeholderText"
        class="chat-composer__text-input"
        @keyup.enter="streaming ? undefined : emit('submit')"
        @update:model-value="(val: string) => emit('update:input', val)"
      />
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
        :loading="loading"
        :disabled="!input.trim()"
        class="chat-composer__send-btn"
        @click="emit('submit')"
      >
        发送
      </el-button>
    </div>
    <div v-if="selectedSkillCapabilities.length > 0" class="chat-composer__selected-skills">
      <el-tag
        v-for="skill in selectedSkillCapabilities"
        :key="skill.key"
        closable
        size="small"
        type="success"
        @close="removeSkill(skill.key)"
      >
        {{ skillLabel(skill) }}
      </el-tag>
    </div>
    <div class="chat-composer__toolbar">
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
        <span class="chat-composer__datasource-label">数据来源</span>
        <span class="chat-composer__datasource-value">{{ dataSource }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.chat-composer {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 12px 16px 14px;
  position: relative;
}

.chat-composer__skill-menu {
  position: absolute;
  left: 16px;
  right: 16px;
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
  gap: 6px;
  margin-top: 10px;
}

.chat-composer__input-row {
  display: flex;
  gap: 10px;
  align-items: center;
}

.chat-composer__text-input {
  flex: 1;
}

.chat-composer__text-input :deep(.el-input__wrapper) {
  box-shadow: none;
  background: transparent;
  padding: 0;
}

.chat-composer__text-input :deep(.el-input__inner) {
  font-size: 14px;
}

.chat-composer__send-btn {
  flex-shrink: 0;
  min-width: 72px;
  height: 36px;
}

.chat-composer__toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 10px;
}

.chat-composer__model-select {
  width: 160px;
  flex-shrink: 0;
}

.chat-composer__datasource {
  display: flex;
  align-items: center;
  gap: 6px;
}

.chat-composer__datasource-label {
  font-size: 11px;
  color: var(--text-faint);
  white-space: nowrap;
  letter-spacing: 0.3px;
}

.chat-composer__datasource-value {
  font-size: 12px;
  color: var(--text-secondary);
  font-weight: 500;
}

@media (max-width: 768px) {
  .chat-composer {
    padding: 10px 12px 12px;
  }

  .chat-composer__input-row {
    flex-wrap: wrap;
  }

  .chat-composer__toolbar {
    flex-wrap: wrap;
    gap: 8px;
  }

  .chat-composer__model-select {
    width: 140px;
  }
}
</style>
