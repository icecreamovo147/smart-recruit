<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  capability: string
  label?: string
}>()

const normalized = computed(() => props.capability.trim().toLowerCase())

const labelText = computed(() => {
  if (props.label) return props.label
  const labels: Record<string, string> = {
    chat: '对话',
    tools: '工具调用',
    tool: '工具调用',
    vision: '视觉',
    json: 'JSON',
    reasoning: '推理',
    embedding: '向量',
    rerank: '重排',
    audio: '音频',
  }
  return labels[normalized.value] || props.capability
})
</script>

<template>
  <el-tag class="admin-capability-tag" :class="`admin-capability-tag--${normalized}`" size="small" effect="plain" round>
    {{ labelText }}
  </el-tag>
</template>

<style scoped>
.admin-capability-tag {
  --admin-capability-color: var(--text-muted);
  border-color: color-mix(in srgb, var(--admin-capability-color) 26%, transparent);
  background: color-mix(in srgb, var(--admin-capability-color) 7%, var(--surface));
  color: var(--admin-capability-color);
  font-weight: 650;
}

.admin-capability-tag--chat {
  --admin-capability-color: var(--brand);
}

.admin-capability-tag--tools,
.admin-capability-tag--tool {
  --admin-capability-color: #0f766e;
}

.admin-capability-tag--vision {
  --admin-capability-color: #7c3aed;
}

.admin-capability-tag--json,
.admin-capability-tag--reasoning {
  --admin-capability-color: #b45309;
}

.admin-capability-tag--embedding,
.admin-capability-tag--rerank {
  --admin-capability-color: #475569;
}

.admin-capability-tag--audio {
  --admin-capability-color: #be185d;
}

:root[data-theme='dark'] .admin-capability-tag--tools,
:root[data-theme='dark'] .admin-capability-tag--tool {
  --admin-capability-color: #2dd4bf;
}

:root[data-theme='dark'] .admin-capability-tag--vision {
  --admin-capability-color: #a78bfa;
}

:root[data-theme='dark'] .admin-capability-tag--json,
:root[data-theme='dark'] .admin-capability-tag--reasoning {
  --admin-capability-color: #fbbf24;
}

:root[data-theme='dark'] .admin-capability-tag--embedding,
:root[data-theme='dark'] .admin-capability-tag--rerank {
  --admin-capability-color: #cbd5e1;
}

:root[data-theme='dark'] .admin-capability-tag--audio {
  --admin-capability-color: #f472b6;
}
</style>
