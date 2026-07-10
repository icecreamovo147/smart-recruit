<script setup lang="ts">
import { computed } from 'vue'
import type { AgentSkillCanvasFlow } from '@/types/agentSkill'

const props = defineProps<{
  flow: AgentSkillCanvasFlow
}>()

const edgeCount = computed(() => props.flow.edges.length)
const nodeCount = computed(() => props.flow.nodes.length)
const jsonPreview = computed(() => JSON.stringify(props.flow, null, 2))
</script>

<template>
  <aside class="agent-skill-preview-panel" aria-label="Flow 预览">
    <div class="agent-skill-preview-panel__header">
      <h3>Flow</h3>
      <span>{{ nodeCount }} 节点 / {{ edgeCount }} 连线</span>
    </div>
    <pre>{{ jsonPreview }}</pre>
  </aside>
</template>

<style scoped>
.agent-skill-preview-panel {
  border-top: 1px solid var(--border);
  background: var(--surface);
}

.agent-skill-preview-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
}

.agent-skill-preview-panel__header h3 {
  margin: 0;
  font-size: 13px;
  font-weight: 700;
}

.agent-skill-preview-panel__header span {
  color: var(--text-faint);
  font-size: 12px;
}

.agent-skill-preview-panel pre {
  max-height: 180px;
  margin: 0;
  overflow: auto;
  padding: 12px;
  background: var(--surface-muted);
  color: var(--text-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-size: 12px;
  line-height: 1.55;
}
</style>
