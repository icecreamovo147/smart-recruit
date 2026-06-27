<script setup lang="ts">
import type { Session } from '@/types/ai'
import type { LlmModel } from '@/types/llm'

const props = defineProps<{
  input: string
  loading: boolean
  streaming: boolean
  modelList: LlmModel[]
  selectedModelId: number | null
  dataSource: string
  currentSession: Session | null
}>()

const emit = defineEmits<{
  (e: 'update:input', value: string): void
  (e: 'update:selectedModelId', value: number | null): void
  (e: 'submit'): void
  (e: 'stop'): void
}>()

const placeholderText = computed(() => {
  if (props.currentSession?.application_id) {
    return '例如：他的项目经历和岗位要求匹配吗？也可以说"通过这个候选人"'
  }
  return '例如：今天后端岗位投递了多少人？'
})
</script>

<template>
  <div class="chat-composer">
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

<script lang="ts">
import { computed } from 'vue'
</script>

<style scoped>
.chat-composer {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 12px 16px 14px;
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
