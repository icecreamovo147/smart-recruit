<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import AgentSkillInspector from './AgentSkillInspector.vue'
import AgentSkillNodePalette from './AgentSkillNodePalette.vue'
import AgentSkillPreviewPanel from './AgentSkillPreviewPanel.vue'
import type {
  AgentSkillCanvasEdge,
  AgentSkillCanvasFlow,
  AgentSkillCanvasNode,
  AgentSkillNodeType,
  AgentSkillNodeTypeOption,
} from '@/types/agentSkill'

const NODE_WIDTH = 184
const NODE_HEIGHT = 104

const DEFAULT_NODE_TYPES: AgentSkillNodeTypeOption[] = [
  { type: 'trigger', label: '触发场景', description: '定义何时适合使用这个 Agent Skill', placeholder: '例如：当 HR 要求比较候选人与岗位 JD 的匹配度时使用。' },
  { type: 'context', label: '上下文', description: '列出助手应读取或引用的业务信息', placeholder: '例如：候选人简历、岗位职责、任职要求、当前投递状态。' },
  { type: 'instruction', label: '执行指令', description: '说明助手应该如何推理和组织回答', placeholder: '例如：先提取硬性条件，再比较项目经历，最后给出风险提示。' },
  { type: 'condition', label: '条件分支', description: '描述不同输入情况下的处理规则', placeholder: '例如：如果简历缺少关键字段，需要明确说明无法判断。' },
  { type: 'output', label: '输出格式', description: '规定回答结构、字段和语气', placeholder: '例如：输出匹配结论、证据、疑点、下一步建议四段。' },
  { type: 'constraint', label: '约束边界', description: '声明安全、合规、不可做事项', placeholder: '例如：不得编造候选人经历，不得输出歧视性判断。' },
]

const props = defineProps<{
  modelValue?: AgentSkillCanvasFlow
  selectedNodeId?: string | null
  nodeTypes?: AgentSkillNodeTypeOption[]
  showPreview?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [flow: AgentSkillCanvasFlow]
  'update:selectedNodeId': [id: string | null]
  change: [flow: AgentSkillCanvasFlow]
}>()

const canvasRef = ref<HTMLElement | null>(null)
const selectedNodeId = ref<string | null>(null)
const connectingSourceId = ref<string | null>(null)
const dragState = ref<{ id: string; offsetX: number; offsetY: number } | null>(null)
const internalFlow = ref<AgentSkillCanvasFlow>(normalizeFlow(props.modelValue))

const selectedNode = computed(() => internalFlow.value.nodes.find((node) => node.id === selectedNodeId.value) || null)
const resolvedNodeTypes = computed(() => props.nodeTypes?.length ? props.nodeTypes : DEFAULT_NODE_TYPES)
const shouldShowPreview = computed(() => props.showPreview ?? true)
const nodeTypeMap = computed(() => resolvedNodeTypes.value.reduce<Record<string, AgentSkillNodeTypeOption>>((acc, item) => {
  acc[item.type] = item
  return acc
}, {}))

watch(() => props.modelValue, (next) => {
  internalFlow.value = normalizeFlow(next)
  if (selectedNodeId.value && !internalFlow.value.nodes.some((node) => node.id === selectedNodeId.value)) {
    selectedNodeId.value = null
  }
}, { deep: true })

watch(() => props.selectedNodeId, (next) => {
  if (next !== undefined && next !== selectedNodeId.value) {
    selectedNodeId.value = next || null
  }
})

watch(selectedNodeId, (next) => {
  emit('update:selectedNodeId', next)
})

function normalizeFlow(flow?: AgentSkillCanvasFlow): AgentSkillCanvasFlow {
  return {
    format: 'canvas.v1',
    version: flow?.version || '1.0.0',
    type: 'agent-skill',
    nodes: flow?.nodes?.map((node) => ({
      id: node.id,
      type: node.type,
      position: {
        x: Number.isFinite(node.position.x) ? node.position.x : 120,
        y: Number.isFinite(node.position.y) ? node.position.y : 80,
      },
      data: {
        title: node.data.title || node.type,
        content: node.data.content || '',
      },
    })) || [],
    edges: flow?.edges?.filter((edge) => edge.source && edge.target).map((edge) => ({ ...edge })) || [],
    viewport: flow?.viewport ? { ...flow.viewport } : undefined,
  }
}

function commit(next: AgentSkillCanvasFlow) {
  internalFlow.value = next
  emit('update:modelValue', next)
  emit('change', next)
}

function createId(prefix: string) {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`
}

function createNode(type: AgentSkillNodeType, x?: number, y?: number): AgentSkillCanvasNode {
  const meta = nodeTypeMap.value[type]
  const index = internalFlow.value.nodes.length
  return {
    id: createId(type),
    type,
    position: {
      x: Math.max(16, Math.round(x ?? 80 + (index % 3) * 220)),
      y: Math.max(16, Math.round(y ?? 70 + Math.floor(index / 3) * 150)),
    },
    data: {
      title: meta?.label || type,
      content: '',
    },
  }
}

function addNode(type: AgentSkillNodeType, position?: { x: number; y: number }) {
  const node = createNode(type, position?.x, position?.y)
  commit({
    ...internalFlow.value,
    nodes: [...internalFlow.value.nodes, node],
  })
  selectedNodeId.value = node.id
}

function updateNode(nextNode: AgentSkillCanvasNode) {
  commit({
    ...internalFlow.value,
    nodes: internalFlow.value.nodes.map((node) => (node.id === nextNode.id ? nextNode : node)),
  })
}

function deleteNode(id: string) {
  commit({
    ...internalFlow.value,
    nodes: internalFlow.value.nodes.filter((node) => node.id !== id),
    edges: internalFlow.value.edges.filter((edge) => edge.source !== id && edge.target !== id),
  })
  if (selectedNodeId.value === id) selectedNodeId.value = null
  if (connectingSourceId.value === id) connectingSourceId.value = null
}

function selectNode(id: string) {
  if (connectingSourceId.value && connectingSourceId.value !== id) {
    addEdge(connectingSourceId.value, id)
    connectingSourceId.value = null
  }
  selectedNodeId.value = id
}

function startConnect(id: string) {
  connectingSourceId.value = id
  selectedNodeId.value = id
}

function addEdge(source: string, target: string) {
  if (source === target) return
  const exists = internalFlow.value.edges.some((edge) => edge.source === source && edge.target === target)
  if (exists) return
  const edge: AgentSkillCanvasEdge = {
    id: createId('edge'),
    source,
    target,
  }
  commit({
    ...internalFlow.value,
    edges: [...internalFlow.value.edges, edge],
  })
}

function deleteEdge(id: string) {
  commit({
    ...internalFlow.value,
    edges: internalFlow.value.edges.filter((edge) => edge.id !== id),
  })
}

function getCanvasPoint(event: MouseEvent | DragEvent) {
  const rect = canvasRef.value?.getBoundingClientRect()
  if (!rect) return { x: 0, y: 0 }
  return {
    x: event.clientX - rect.left,
    y: event.clientY - rect.top,
  }
}

function handleDrop(event: DragEvent) {
  const type = event.dataTransfer?.getData('application/x-agent-skill-node-type') || event.dataTransfer?.getData('text/plain')
  if (!resolvedNodeTypes.value.some((item) => item.type === type)) return
  const point = getCanvasPoint(event)
  addNode(type as AgentSkillNodeType, {
    x: point.x - NODE_WIDTH / 2,
    y: point.y - 28,
  })
}

function handleNodePointerDown(event: PointerEvent, node: AgentSkillCanvasNode) {
  if (event.button !== 0) return
  const target = event.target as HTMLElement
  if (target.closest('button')) return
  selectNode(node.id)
  const point = getCanvasPoint(event)
  dragState.value = {
    id: node.id,
    offsetX: point.x - node.position.x,
    offsetY: point.y - node.position.y,
  }
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
}

function handlePointerMove(event: PointerEvent) {
  if (!dragState.value) return
  const point = getCanvasPoint(event)
  const nextX = Math.max(8, Math.round(point.x - dragState.value.offsetX))
  const nextY = Math.max(8, Math.round(point.y - dragState.value.offsetY))
  commit({
    ...internalFlow.value,
    nodes: internalFlow.value.nodes.map((node) => (
      node.id === dragState.value?.id
        ? { ...node, position: { x: nextX, y: nextY } }
        : node
    )),
  })
}

function handlePointerUp() {
  dragState.value = null
}

function edgePoints(edge: AgentSkillCanvasEdge) {
  const source = internalFlow.value.nodes.find((node) => node.id === edge.source)
  const target = internalFlow.value.nodes.find((node) => node.id === edge.target)
  if (!source || !target) return null
  return {
    x1: source.position.x + NODE_WIDTH,
    y1: source.position.y + NODE_HEIGHT / 2,
    x2: target.position.x,
    y2: target.position.y + NODE_HEIGHT / 2,
    midX: (source.position.x + NODE_WIDTH + target.position.x) / 2,
    midY: (source.position.y + target.position.y + NODE_HEIGHT) / 2,
  }
}

function edgePath(edge: AgentSkillCanvasEdge) {
  const points = edgePoints(edge)
  if (!points) return ''
  const controlOffset = Math.max(48, Math.abs(points.x2 - points.x1) / 2)
  return `M ${points.x1} ${points.y1} C ${points.x1 + controlOffset} ${points.y1}, ${points.x2 - controlOffset} ${points.y2}, ${points.x2} ${points.y2}`
}
</script>

<template>
  <section class="agent-skill-canvas-editor">
    <AgentSkillNodePalette :node-types="resolvedNodeTypes" @add="addNode" />

    <main class="agent-skill-canvas-editor__main">
      <div class="agent-skill-canvas-editor__toolbar">
        <div>
          <h2>画布编排</h2>
          <span v-if="connectingSourceId">连线源已选择，点击目标节点建立连接</span>
          <span v-else>拖入节点，移动位置，选择节点后编辑属性</span>
        </div>
        <button type="button" :disabled="!connectingSourceId" @click="connectingSourceId = null">取消连线</button>
      </div>

      <div
        ref="canvasRef"
        class="agent-skill-canvas"
        @dragover.prevent
        @drop.prevent="handleDrop"
        @pointermove="handlePointerMove"
        @pointerup="handlePointerUp"
        @pointerleave="handlePointerUp"
        @click.self="selectedNodeId = null"
      >
        <svg class="agent-skill-canvas__edges">
          <defs>
            <marker id="agent-skill-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
              <path d="M 0 0 L 10 5 L 0 10 z" />
            </marker>
          </defs>
          <g v-for="edge in internalFlow.edges" :key="edge.id">
            <path class="agent-skill-canvas__edge" :d="edgePath(edge)" marker-end="url(#agent-skill-arrow)" />
            <foreignObject
              v-if="edgePoints(edge)"
              :x="edgePoints(edge)!.midX - 34"
              :y="edgePoints(edge)!.midY - 12"
              width="68"
              height="24"
            >
              <button class="agent-skill-canvas__edge-delete" type="button" @click="deleteEdge(edge.id)">删除</button>
            </foreignObject>
          </g>
        </svg>

        <article
          v-for="node in internalFlow.nodes"
          :key="node.id"
          class="agent-skill-canvas-node"
          :class="{
            'agent-skill-canvas-node--selected': selectedNodeId === node.id,
            'agent-skill-canvas-node--source': connectingSourceId === node.id,
          }"
          :style="{ left: `${node.position.x}px`, top: `${node.position.y}px` }"
          @pointerdown="handleNodePointerDown($event, node)"
        >
          <div class="agent-skill-canvas-node__head">
            <span>{{ nodeTypeMap[node.type]?.label || node.type }}</span>
            <button type="button" @click.stop="startConnect(node.id)">连线</button>
          </div>
          <h3>{{ node.data.title || nodeTypeMap[node.type]?.label || node.type }}</h3>
          <p>{{ node.data.content || '未填写内容' }}</p>
        </article>

        <div v-if="internalFlow.nodes.length === 0" class="agent-skill-canvas__empty">
          从左侧拖入节点，或点击节点库快速创建。
        </div>
      </div>

      <AgentSkillPreviewPanel v-if="shouldShowPreview" :flow="internalFlow" />
    </main>

    <AgentSkillInspector
      :node="selectedNode"
      :node-types="resolvedNodeTypes"
      @update-node="updateNode"
      @delete-node="deleteNode"
      @start-connect="startConnect"
      @clear-selection="selectedNodeId = null"
    />
  </section>
</template>

<style scoped>
.agent-skill-canvas-editor {
  display: grid;
  grid-template-columns: 220px minmax(520px, 1fr) 300px;
  height: 100%;
  min-height: 620px;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  box-shadow: var(--admin-console-card-shadow, 0 10px 28px rgba(15, 23, 42, 0.06));
}

.agent-skill-canvas-editor__main {
  display: grid;
  grid-template-rows: auto minmax(420px, 1fr) auto;
  min-width: 0;
  min-height: 0;
  background: var(--surface-muted);
}

.agent-skill-canvas-editor__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--border);
  background: var(--surface);
}

.agent-skill-canvas-editor__toolbar h2 {
  margin: 0;
  color: var(--text-primary);
  font-size: 15px;
  font-weight: 700;
}

.agent-skill-canvas-editor__toolbar span {
  display: block;
  margin-top: 3px;
  color: var(--text-muted);
  font-size: 12px;
}

.agent-skill-canvas-editor__toolbar button {
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 7px 10px;
  background: var(--surface);
  color: var(--text-secondary);
  cursor: pointer;
}

.agent-skill-canvas-editor__toolbar button:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.agent-skill-canvas {
  position: relative;
  min-height: 420px;
  height: 100%;
  overflow: hidden;
  background-color: var(--surface-muted);
  background-image:
    linear-gradient(var(--border) 1px, transparent 1px),
    linear-gradient(90deg, var(--border) 1px, transparent 1px);
  background-size: 24px 24px;
  user-select: none;
}

.agent-skill-canvas__edges {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  overflow: visible;
  pointer-events: none;
}

.agent-skill-canvas__edge {
  fill: none;
  stroke: color-mix(in srgb, var(--brand) 70%, var(--text-muted));
  stroke-width: 2;
}

.agent-skill-canvas__edges marker path {
  fill: color-mix(in srgb, var(--brand) 70%, var(--text-muted));
}

.agent-skill-canvas__edge-delete {
  width: 52px;
  height: 22px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--surface);
  color: var(--text-muted);
  font-size: 12px;
  cursor: pointer;
  pointer-events: auto;
}

.agent-skill-canvas-node {
  position: absolute;
  width: 184px;
  height: 104px;
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 10px;
  background: var(--surface);
  box-shadow: 0 8px 20px rgba(15, 23, 42, 0.08);
  cursor: move;
}

.agent-skill-canvas-node--selected {
  border-color: var(--brand);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--brand-soft) 72%, transparent), 0 8px 20px rgba(15, 23, 42, 0.08);
}

.agent-skill-canvas-node--source {
  outline: 2px solid color-mix(in srgb, var(--brand) 78%, transparent);
  outline-offset: 2px;
}

.agent-skill-canvas-node__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}

.agent-skill-canvas-node__head span {
  overflow: hidden;
  color: var(--text-muted);
  font-size: 12px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-skill-canvas-node__head button {
  flex: 0 0 auto;
  border: 1px solid var(--border);
  border-radius: 999px;
  padding: 3px 8px;
  background: var(--surface-muted);
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
}

.agent-skill-canvas-node h3 {
  overflow: hidden;
  margin: 0;
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-skill-canvas-node p {
  display: -webkit-box;
  overflow: hidden;
  margin: 7px 0 0;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.45;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.agent-skill-canvas__empty {
  position: absolute;
  left: 50%;
  top: 50%;
  width: min(320px, calc(100% - 32px));
  transform: translate(-50%, -50%);
  border: 1px dashed var(--border);
  border-radius: 8px;
  padding: 18px;
  background: color-mix(in srgb, var(--surface) 86%, transparent);
  color: var(--text-muted);
  text-align: center;
  font-size: 13px;
}

@media (max-width: 1180px) {
  .agent-skill-canvas-editor {
    grid-template-columns: 200px minmax(420px, 1fr) 280px;
  }
}

@media (max-width: 920px) {
  .agent-skill-canvas-editor {
    grid-template-columns: 1fr;
  }

  .agent-skill-canvas-editor :deep(.agent-skill-node-palette),
  .agent-skill-canvas-editor :deep(.agent-skill-inspector) {
    border: 0;
    border-bottom: 1px solid var(--border);
  }
}
</style>
