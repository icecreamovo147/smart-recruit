<script setup lang="ts">
import type { AgentSkillNodeType, AgentSkillNodeTypeOption } from '@shared/types/agentSkill'

defineProps<{
  nodeTypes: AgentSkillNodeTypeOption[]
}>()

const emit = defineEmits<{
  add: [type: AgentSkillNodeType]
}>()

function handleDragStart(event: DragEvent, type: AgentSkillNodeType) {
  event.dataTransfer?.setData('application/x-agent-skill-node-type', type)
  event.dataTransfer?.setData('text/plain', type)
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'copy'
  }
}
</script>

<template>
  <aside class="agent-skill-node-palette" aria-label="Agent Skill 节点库">
    <div class="agent-skill-node-palette__header">
      <h3>节点库</h3>
      <span>{{ nodeTypes.length }} 类</span>
    </div>

    <button
      v-for="item in nodeTypes"
      :key="item.type"
      class="agent-skill-node-palette__item"
      type="button"
      draggable="true"
      @click="emit('add', item.type)"
      @dragstart="handleDragStart($event, item.type)"
    >
      <span class="agent-skill-node-palette__label">{{ item.label }}</span>
      <span class="agent-skill-node-palette__type">{{ item.type }}</span>
      <span class="agent-skill-node-palette__desc">{{ item.description }}</span>
    </button>
  </aside>
</template>

<style scoped>
.agent-skill-node-palette {
  min-width: 0;
  border-right: 1px solid var(--border);
  background: var(--surface);
}

.agent-skill-node-palette__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 14px;
  border-bottom: 1px solid var(--border);
}

.agent-skill-node-palette__header h3 {
  margin: 0;
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 700;
}

.agent-skill-node-palette__header span,
.agent-skill-node-palette__type {
  color: var(--text-faint);
  font-size: 12px;
}

.agent-skill-node-palette__item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  width: calc(100% - 20px);
  margin: 10px;
  gap: 4px 8px;
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 10px;
  background: var(--surface-muted);
  color: var(--text-primary);
  text-align: left;
  cursor: grab;
  transition:
    border-color var(--motion-fast, 120ms) ease,
    background-color var(--motion-fast, 120ms) ease;
}

.agent-skill-node-palette__item:hover {
  border-color: var(--brand);
  background: color-mix(in srgb, var(--brand-soft) 34%, var(--surface));
}

.agent-skill-node-palette__item:active {
  cursor: grabbing;
}

.agent-skill-node-palette__label {
  overflow: hidden;
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-skill-node-palette__desc {
  grid-column: 1 / -1;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.45;
}
</style>
