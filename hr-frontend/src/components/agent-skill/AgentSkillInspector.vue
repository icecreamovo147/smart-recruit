<script setup lang="ts">
import { computed } from 'vue'
import type { AgentSkillCanvasNode, AgentSkillNodeType, AgentSkillNodeTypeOption } from '@/types/agentSkill'

const props = defineProps<{
  node: AgentSkillCanvasNode | null
  nodeTypes: AgentSkillNodeTypeOption[]
}>()

const emit = defineEmits<{
  updateNode: [node: AgentSkillCanvasNode]
  deleteNode: [id: string]
  startConnect: [id: string]
  clearSelection: []
}>()

const selectedTypeMeta = computed(() => props.nodeTypes.find((item) => item.type === props.node?.type))

type AgentSkillNodePatch = Omit<Partial<AgentSkillCanvasNode>, 'data'> & {
  data?: Partial<AgentSkillCanvasNode['data']>
}

function patchNode(patch: AgentSkillNodePatch) {
  if (!props.node) return
  emit('updateNode', {
    ...props.node,
    ...patch,
    data: {
      ...props.node.data,
      ...patch.data,
    },
  })
}

function updateType(type: string | number | boolean | undefined) {
  if (typeof type !== 'string') return
  patchNode({ type: type as AgentSkillNodeType })
}
</script>

<template>
  <aside class="agent-skill-inspector" aria-label="节点属性">
    <div class="agent-skill-inspector__header">
      <h3>属性</h3>
      <button v-if="node" type="button" @click="emit('clearSelection')">取消选择</button>
    </div>

    <div v-if="node" class="agent-skill-inspector__body">
      <label class="agent-skill-field">
        <span>标题</span>
        <input
          :value="node.data.title"
          type="text"
          placeholder="节点标题"
          @input="patchNode({ data: { title: ($event.target as HTMLInputElement).value } })"
        >
      </label>

      <label class="agent-skill-field">
        <span>类型</span>
        <select :value="node.type" @change="updateType(($event.target as HTMLSelectElement).value)">
          <option v-for="item in nodeTypes" :key="item.type" :value="item.type">{{ item.label }}</option>
        </select>
      </label>

      <label class="agent-skill-field">
        <span>内容</span>
        <textarea
          :value="node.data.content"
          :placeholder="selectedTypeMeta?.placeholder || '补充这个节点的执行要求、上下文或输出约束'"
          rows="9"
          @input="patchNode({ data: { content: ($event.target as HTMLTextAreaElement).value } })"
        />
      </label>

      <div class="agent-skill-inspector__meta">
        <span>ID</span>
        <code>{{ node.id }}</code>
      </div>

      <div class="agent-skill-inspector__actions">
        <button type="button" class="agent-skill-inspector__primary" @click="emit('startConnect', node.id)">
          设为连线源
        </button>
        <button type="button" class="agent-skill-inspector__danger" @click="emit('deleteNode', node.id)">
          删除节点
        </button>
      </div>
    </div>

    <div v-else class="agent-skill-inspector__empty">
      选择画布中的节点后编辑标题、类型和内容。
    </div>
  </aside>
</template>

<style scoped>
.agent-skill-inspector {
  min-width: 0;
  border-left: 1px solid var(--border);
  background: var(--surface);
}

.agent-skill-inspector__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 14px;
  border-bottom: 1px solid var(--border);
}

.agent-skill-inspector__header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 700;
}

.agent-skill-inspector__header button,
.agent-skill-inspector__actions button {
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 7px 10px;
  background: var(--surface);
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
}

.agent-skill-inspector__body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
}

.agent-skill-field {
  display: grid;
  gap: 6px;
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.agent-skill-field input,
.agent-skill-field select,
.agent-skill-field textarea {
  width: 100%;
  min-width: 0;
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 9px 10px;
  background: var(--surface);
  color: var(--text-primary);
  font: inherit;
  font-weight: 500;
  line-height: 1.45;
  outline: none;
}

.agent-skill-field textarea {
  resize: vertical;
}

.agent-skill-field input:focus,
.agent-skill-field select:focus,
.agent-skill-field textarea:focus {
  border-color: var(--brand);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--brand-soft) 70%, transparent);
}

.agent-skill-inspector__meta {
  display: grid;
  gap: 5px;
  min-width: 0;
  color: var(--text-faint);
  font-size: 12px;
}

.agent-skill-inspector__meta code {
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 7px;
  background: var(--surface-muted);
  color: var(--text-muted);
  text-overflow: ellipsis;
}

.agent-skill-inspector__actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.agent-skill-inspector__actions .agent-skill-inspector__primary {
  border-color: color-mix(in srgb, var(--brand) 48%, var(--border));
  background: var(--brand);
  color: #fff;
}

.agent-skill-inspector__actions .agent-skill-inspector__danger {
  border-color: color-mix(in srgb, #dc2626 38%, var(--border));
  color: #dc2626;
}

.agent-skill-inspector__empty {
  padding: 16px 14px;
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.55;
}
</style>
