<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import AgentSkillInspector from './AgentSkillInspector.vue'
import AgentSkillNodePalette from './AgentSkillNodePalette.vue'
import AgentSkillPreviewPanel from './AgentSkillPreviewPanel.vue'
import type {
  AgentSkillCanvasEdge,
  AgentSkillCanvasFlow,
  AgentSkillCanvasHandle,
  AgentSkillCanvasNode,
  AgentSkillNodeType,
  AgentSkillNodeTypeOption,
} from '@shared/types/agentSkill'

const NODE_WIDTH = 184
const NODE_HEIGHT = 104
const DEFAULT_VIEWPORT = { x: 0, y: 0, zoom: 1 }
const MIN_ZOOM = 0.35
const MAX_ZOOM = 2
const FIT_VIEW_PADDING = 36
const DEFAULT_SOURCE_HANDLE: AgentSkillCanvasHandle = 'right'
const DEFAULT_TARGET_HANDLE: AgentSkillCanvasHandle = 'left'
const DEFAULT_EDGE_CURVATURE = 0.5
const MIN_EDGE_CURVATURE = -1
const MAX_EDGE_CURVATURE = 1
const CONNECTION_HANDLES: AgentSkillCanvasHandle[] = ['top', 'right', 'bottom', 'left']

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

const editorRef = ref<HTMLElement | null>(null)
const canvasRef = ref<HTMLElement | null>(null)
const selectedNodeId = ref<string | null>(null)
const selectedEdgeId = ref<string | null>(null)
const dragState = ref<{ id: string; offsetX: number; offsetY: number } | null>(null)
const panState = ref<{ startX: number; startY: number; viewportX: number; viewportY: number } | null>(null)
const connectionDragState = ref<{ sourceId: string; sourceHandle: AgentSkillCanvasHandle; point: { x: number; y: number } } | null>(null)
const internalFlow = ref<AgentSkillCanvasFlow>(normalizeFlow(props.modelValue))
const isFullscreen = ref(false)

const selectedNode = computed(() => internalFlow.value.nodes.find((node) => node.id === selectedNodeId.value) || null)
const viewport = computed(() => internalFlow.value.viewport || DEFAULT_VIEWPORT)
const contentLayerStyle = computed(() => ({
  transform: `translate(${viewport.value.x}px, ${viewport.value.y}px) scale(${viewport.value.zoom})`,
}))
const isPanning = computed(() => Boolean(panState.value))
const connectingSourceId = computed(() => connectionDragState.value?.sourceId || null)
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

onMounted(() => {
  document.addEventListener('fullscreenchange', handleFullscreenChange)
  document.addEventListener('keydown', handleKeydown)
  handleFullscreenChange()
})

onBeforeUnmount(() => {
  document.removeEventListener('fullscreenchange', handleFullscreenChange)
  document.removeEventListener('keydown', handleKeydown)
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
    edges: normalizeEdges(flow?.edges),
    viewport: normalizeViewport(flow?.viewport),
  }
}

function normalizeEdges(edges?: AgentSkillCanvasEdge[]) {
  const seenPairs = new Set<string>()
  return edges?.filter((edge) => edge.source && edge.target).flatMap((edge) => {
    const pairKey = `${edge.source}->${edge.target}`
    if (seenPairs.has(pairKey)) return []
    seenPairs.add(pairKey)
    return [{
      ...edge,
      sourceHandle: normalizeHandle(edge.sourceHandle, DEFAULT_SOURCE_HANDLE),
      targetHandle: normalizeHandle(edge.targetHandle, DEFAULT_TARGET_HANDLE),
      curvature: normalizeCurvature(edge.curvature),
    }]
  }) || []
}

function normalizeHandle(handle: AgentSkillCanvasEdge['sourceHandle'], fallback: AgentSkillCanvasHandle): AgentSkillCanvasHandle {
  return handle && CONNECTION_HANDLES.includes(handle) ? handle : fallback
}

function normalizeCurvature(curvature?: number) {
  if (!Number.isFinite(curvature)) return DEFAULT_EDGE_CURVATURE
  return clamp(Number(curvature), MIN_EDGE_CURVATURE, MAX_EDGE_CURVATURE)
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value))
}

function normalizeViewport(viewport?: AgentSkillCanvasFlow['viewport']) {
  return {
    x: Number.isFinite(viewport?.x) ? viewport!.x : DEFAULT_VIEWPORT.x,
    y: Number.isFinite(viewport?.y) ? viewport!.y : DEFAULT_VIEWPORT.y,
    zoom: Number.isFinite(viewport?.zoom) && viewport!.zoom > 0 ? viewport!.zoom : DEFAULT_VIEWPORT.zoom,
  }
}

function commit(next: AgentSkillCanvasFlow) {
  const normalized = normalizeFlow(next)
  internalFlow.value = normalized
  emit('update:modelValue', normalized)
  emit('change', normalized)
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
      x: Math.round(x ?? 80 + (index % 3) * 220),
      y: Math.round(y ?? 70 + Math.floor(index / 3) * 150),
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
  if (connectionDragState.value?.sourceId === id) connectionDragState.value = null
}

function selectNode(id: string) {
  selectedNodeId.value = id
  selectedEdgeId.value = null
}

function addEdge(
  source: string,
  target: string,
  sourceHandle: AgentSkillCanvasHandle = DEFAULT_SOURCE_HANDLE,
  targetHandle: AgentSkillCanvasHandle = DEFAULT_TARGET_HANDLE,
) {
  if (source === target) return
  const exists = internalFlow.value.edges.some((edge) => (
    edge.source === source
    && edge.target === target
  ))
  if (exists) return
  const edge: AgentSkillCanvasEdge = {
    id: createId('edge'),
    source,
    target,
    sourceHandle,
    targetHandle,
    curvature: DEFAULT_EDGE_CURVATURE,
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
  if (selectedEdgeId.value === id) selectedEdgeId.value = null
}

function getCanvasPoint(event: MouseEvent | DragEvent) {
  const rect = canvasRef.value?.getBoundingClientRect()
  if (!rect) return { x: 0, y: 0 }
  return {
    x: event.clientX - rect.left,
    y: event.clientY - rect.top,
  }
}

function toFlowPoint(point: { x: number; y: number }) {
  return {
    x: (point.x - viewport.value.x) / viewport.value.zoom,
    y: (point.y - viewport.value.y) / viewport.value.zoom,
  }
}

function handleDrop(event: DragEvent) {
  const type = event.dataTransfer?.getData('application/x-agent-skill-node-type') || event.dataTransfer?.getData('text/plain')
  if (!resolvedNodeTypes.value.some((item) => item.type === type)) return
  const point = toFlowPoint(getCanvasPoint(event))
  addNode(type as AgentSkillNodeType, {
    x: point.x - NODE_WIDTH / 2,
    y: point.y - 28,
  })
}

function handleNodePointerDown(event: PointerEvent, node: AgentSkillCanvasNode) {
  if (event.button !== 0) return
  const target = event.target as HTMLElement
  if (target.closest('button') || target.closest('.agent-skill-canvas-node__connector')) return
  selectNode(node.id)
  const point = toFlowPoint(getCanvasPoint(event))
  dragState.value = {
    id: node.id,
    offsetX: point.x - node.position.x,
    offsetY: point.y - node.position.y,
  }
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
}

function handlePointerMove(event: PointerEvent) {
  if (connectionDragState.value) {
    connectionDragState.value = {
      ...connectionDragState.value,
      point: toFlowPoint(getCanvasPoint(event)),
    }
    return
  }

  if (panState.value) {
    commit({
      ...internalFlow.value,
      viewport: {
        ...viewport.value,
        x: Math.round(panState.value.viewportX + event.clientX - panState.value.startX),
        y: Math.round(panState.value.viewportY + event.clientY - panState.value.startY),
      },
    })
    return
  }

  if (dragState.value) {
    const point = toFlowPoint(getCanvasPoint(event))
    const nextX = Math.round(point.x - dragState.value.offsetX)
    const nextY = Math.round(point.y - dragState.value.offsetY)
    commit({
      ...internalFlow.value,
      nodes: internalFlow.value.nodes.map((node) => (
        node.id === dragState.value?.id
          ? { ...node, position: { x: nextX, y: nextY } }
          : node
      )),
    })
  }
}

function handlePointerUp(event: PointerEvent) {
  if (connectionDragState.value) {
    const target = getConnectionTarget(event, connectionDragState.value.sourceId)
    if (target) addEdge(connectionDragState.value.sourceId, target.nodeId, connectionDragState.value.sourceHandle, target.handle)
    connectionDragState.value = null
  }
  dragState.value = null
  panState.value = null
}

function handleCanvasPointerDown(event: PointerEvent) {
  const target = event.target as HTMLElement
  const isBlankCanvas = event.target === canvasRef.value
    || target.classList.contains('agent-skill-canvas__content')
    || target.classList.contains('agent-skill-canvas__edges')
  if (event.button !== 0 || !isBlankCanvas) return
  selectedNodeId.value = null
  selectedEdgeId.value = null
  panState.value = {
    startX: event.clientX,
    startY: event.clientY,
    viewportX: viewport.value.x,
    viewportY: viewport.value.y,
  }
  canvasRef.value?.setPointerCapture(event.pointerId)
}

function handleConnectorPointerDown(event: PointerEvent, node: AgentSkillCanvasNode, handle: AgentSkillCanvasHandle) {
  if (event.button !== 0) return
  event.stopPropagation()
  selectedNodeId.value = node.id
  selectedEdgeId.value = null
  connectionDragState.value = {
    sourceId: node.id,
    sourceHandle: handle,
    point: toFlowPoint(getCanvasPoint(event)),
  }
}

function selectEdge(id: string) {
  selectedEdgeId.value = id
  selectedNodeId.value = null
}

function getConnectionTarget(
  event: PointerEvent,
  sourceId: string,
): { nodeId: string; handle: AgentSkillCanvasHandle } | null {
  const target = event.target as HTMLElement
  const connector = target.closest<HTMLElement>('.agent-skill-canvas-node__connector')
  const targetNodeId = connector?.dataset.nodeId
  const targetHandle = connector?.dataset.handle as AgentSkillCanvasHandle | undefined
  if (!targetNodeId || targetNodeId === sourceId) return null
  if (!targetHandle || !CONNECTION_HANDLES.includes(targetHandle)) return null
  const exists = internalFlow.value.edges.some((edge) => (
    edge.source === sourceId
    && edge.target === targetNodeId
  ))
  return exists ? null : { nodeId: targetNodeId, handle: targetHandle }
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    connectionDragState.value = null
    return
  }

  if (event.key !== 'Delete' && event.key !== 'Backspace') return
  const target = event.target as HTMLElement | null
  const isEditing = target?.matches('input, textarea, select, [contenteditable="true"]')
  if (isEditing) return

  if (selectedEdgeId.value) {
    event.preventDefault()
    deleteEdge(selectedEdgeId.value)
    return
  }

  if (selectedNodeId.value) {
    event.preventDefault()
    deleteNode(selectedNodeId.value)
  }
}

function fitView() {
  if (internalFlow.value.nodes.length === 0) {
    commit({
      ...internalFlow.value,
      viewport: { ...DEFAULT_VIEWPORT },
    })
    return
  }

  const rect = canvasRef.value?.getBoundingClientRect()
  if (!rect || rect.width <= 0 || rect.height <= 0) return

  const bounds = internalFlow.value.nodes.reduce((acc, node) => ({
    minX: Math.min(acc.minX, node.position.x),
    minY: Math.min(acc.minY, node.position.y),
    maxX: Math.max(acc.maxX, node.position.x + NODE_WIDTH),
    maxY: Math.max(acc.maxY, node.position.y + NODE_HEIGHT),
  }), {
    minX: Number.POSITIVE_INFINITY,
    minY: Number.POSITIVE_INFINITY,
    maxX: Number.NEGATIVE_INFINITY,
    maxY: Number.NEGATIVE_INFINITY,
  })

  const boundsWidth = Math.max(1, bounds.maxX - bounds.minX)
  const boundsHeight = Math.max(1, bounds.maxY - bounds.minY)
  const availableWidth = Math.max(1, rect.width - FIT_VIEW_PADDING * 2)
  const availableHeight = Math.max(1, rect.height - FIT_VIEW_PADDING * 2)
  const zoom = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, Math.min(availableWidth / boundsWidth, availableHeight / boundsHeight)))
  const centerX = bounds.minX + boundsWidth / 2
  const centerY = bounds.minY + boundsHeight / 2

  commit({
    ...internalFlow.value,
    viewport: {
      x: Math.round(rect.width / 2 - centerX * zoom),
      y: Math.round(rect.height / 2 - centerY * zoom),
      zoom: Number(zoom.toFixed(3)),
    },
  })
}

async function toggleFullscreen() {
  try {
    if (document.fullscreenElement === editorRef.value) {
      await document.exitFullscreen()
      return
    }
    await editorRef.value?.requestFullscreen()
  } catch {
    handleFullscreenChange()
  }
}

function handleFullscreenChange() {
  isFullscreen.value = document.fullscreenElement === editorRef.value
}

function handleCanvasClick(event: MouseEvent) {
  const target = event.target as HTMLElement
  if (event.target === canvasRef.value
    || target.classList.contains('agent-skill-canvas__content')
    || target.classList.contains('agent-skill-canvas__edges')) {
    selectedNodeId.value = null
    selectedEdgeId.value = null
  }
}

function getHandlePoint(node: AgentSkillCanvasNode, handle: AgentSkillCanvasHandle) {
  const x = node.position.x
  const y = node.position.y
  if (handle === 'top') return { x: x + NODE_WIDTH / 2, y }
  if (handle === 'right') return { x: x + NODE_WIDTH, y: y + NODE_HEIGHT / 2 }
  if (handle === 'bottom') return { x: x + NODE_WIDTH / 2, y: y + NODE_HEIGHT }
  return { x, y: y + NODE_HEIGHT / 2 }
}

function buildLinePath(sourcePoint: { x: number; y: number }, targetPoint: { x: number; y: number }) {
  return {
    path: `M ${sourcePoint.x} ${sourcePoint.y} L ${targetPoint.x} ${targetPoint.y}`,
    midX: (sourcePoint.x + targetPoint.x) / 2,
    midY: (sourcePoint.y + targetPoint.y) / 2,
  }
}

function edgePoints(edge: AgentSkillCanvasEdge) {
  const source = internalFlow.value.nodes.find((node) => node.id === edge.source)
  const target = internalFlow.value.nodes.find((node) => node.id === edge.target)
  if (!source || !target) return null
  const sourceHandle = normalizeHandle(edge.sourceHandle, DEFAULT_SOURCE_HANDLE)
  const targetHandle = normalizeHandle(edge.targetHandle, DEFAULT_TARGET_HANDLE)
  const sourcePoint = getHandlePoint(source, sourceHandle)
  const targetPoint = getHandlePoint(target, targetHandle)
  return {
    x1: sourcePoint.x,
    y1: sourcePoint.y,
    x2: targetPoint.x,
    y2: targetPoint.y,
    ...buildLinePath(sourcePoint, targetPoint),
  }
}

function edgePath(edge: AgentSkillCanvasEdge) {
  return edgePoints(edge)?.path || ''
}

function temporaryEdgePath() {
  if (!connectionDragState.value) return ''
  const source = internalFlow.value.nodes.find((node) => node.id === connectionDragState.value?.sourceId)
  if (!source) return ''
  const sourcePoint = getHandlePoint(source, connectionDragState.value.sourceHandle)
  const targetPoint = connectionDragState.value.point
  return buildLinePath(sourcePoint, targetPoint).path
}
</script>

<template>
  <section ref="editorRef" class="agent-skill-canvas-editor">
    <AgentSkillNodePalette :node-types="resolvedNodeTypes" @add="addNode" />

    <main class="agent-skill-canvas-editor__main">
      <div class="agent-skill-canvas-editor__toolbar">
        <div>
          <h2>画布编排</h2>
          <span v-if="connectingSourceId">拖拽连线到任意节点的任一连接点</span>
          <span v-else>拖入节点，移动位置，选择节点后编辑属性</span>
        </div>
        <div class="agent-skill-canvas-editor__actions">
          <button type="button" @click="fitView">适应画布</button>
          <button type="button" @click="toggleFullscreen">{{ isFullscreen ? '退出全屏' : '全屏编排' }}</button>
        </div>
      </div>

      <div
        ref="canvasRef"
        class="agent-skill-canvas"
        :class="{ 'agent-skill-canvas--panning': isPanning }"
        @dragover.prevent
        @drop.prevent="handleDrop"
        @pointerdown="handleCanvasPointerDown"
        @pointermove="handlePointerMove"
        @pointerup="handlePointerUp"
        @pointerleave="handlePointerUp"
        @click="handleCanvasClick"
      >
        <div class="agent-skill-canvas__content" :style="contentLayerStyle">
          <svg class="agent-skill-canvas__edges" :class="{ 'agent-skill-canvas__edges--connecting': connectionDragState }">
            <defs>
              <marker id="agent-skill-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
                <path d="M 0 0 L 10 5 L 0 10 z" />
              </marker>
            </defs>
            <g v-for="edge in internalFlow.edges" :key="edge.id">
              <path class="agent-skill-canvas__edge-hit" :d="edgePath(edge)" @click.stop="selectEdge(edge.id)" />
              <path
                class="agent-skill-canvas__edge"
                :class="{ 'agent-skill-canvas__edge--selected': selectedEdgeId === edge.id }"
                :d="edgePath(edge)"
                marker-end="url(#agent-skill-arrow)"
                @click.stop="selectEdge(edge.id)"
              />
              <template v-if="selectedEdgeId === edge.id && edgePoints(edge)">
                <g
                  class="agent-skill-canvas__edge-delete"
                  :transform="`translate(${edgePoints(edge)!.midX}, ${edgePoints(edge)!.midY})`"
                  @click.stop="deleteEdge(edge.id)"
                >
                  <circle r="10" />
                  <path d="M -4 -4 L 4 4 M 4 -4 L -4 4" />
                </g>
              </template>
            </g>
            <path
              v-if="connectionDragState"
              class="agent-skill-canvas__edge agent-skill-canvas__edge--temporary"
              :d="temporaryEdgePath()"
              marker-end="url(#agent-skill-arrow)"
            />
          </svg>

          <article
            v-for="node in internalFlow.nodes"
            :key="node.id"
            class="agent-skill-canvas-node"
            :class="{
              'agent-skill-canvas-node--selected': selectedNodeId === node.id,
              'agent-skill-canvas-node--source': connectingSourceId === node.id,
              'agent-skill-canvas-node--connecting': Boolean(connectionDragState),
            }"
            :style="{ left: `${node.position.x}px`, top: `${node.position.y}px` }"
            @pointerdown="handleNodePointerDown($event, node)"
          >
            <span
              v-for="handle in CONNECTION_HANDLES"
              :key="handle"
              class="agent-skill-canvas-node__connector agent-skill-canvas-node__connector--input"
              :class="[
                `agent-skill-canvas-node__connector--${handle}`,
                { 'agent-skill-canvas-node__connector--active': connectingSourceId === node.id && connectionDragState?.sourceHandle === handle },
              ]"
              :data-node-id="node.id"
              :data-handle="handle"
              :title="`${handle} 连接点`"
              @pointerdown="handleConnectorPointerDown($event, node, handle)"
            />
            <div class="agent-skill-canvas-node__head">
              <span>{{ nodeTypeMap[node.type]?.label || node.type }}</span>
            </div>
            <h3>{{ node.data.title || nodeTypeMap[node.type]?.label || node.type }}</h3>
            <p>{{ node.data.content || '未填写内容' }}</p>
          </article>
        </div>

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

.agent-skill-canvas-editor:fullscreen {
  width: 100vw;
  height: 100vh;
  min-height: 0;
  border: 0;
  border-radius: 0;
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

.agent-skill-canvas-editor__actions {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
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
  cursor: grab;
  user-select: none;
}

.agent-skill-canvas--panning {
  cursor: grabbing;
}

.agent-skill-canvas__content {
  position: absolute;
  inset: 0;
  transform-origin: 0 0;
}

.agent-skill-canvas__edges {
  position: absolute;
  inset: 0;
  z-index: 1;
  width: 100%;
  height: 100%;
  overflow: visible;
}

.agent-skill-canvas__edges--connecting {
  pointer-events: none;
}

.agent-skill-canvas__edge {
  fill: none;
  stroke: color-mix(in srgb, var(--brand) 70%, var(--text-muted));
  stroke-width: 2;
  pointer-events: stroke;
  cursor: pointer;
}

.agent-skill-canvas__edge-hit {
  fill: none;
  stroke: transparent;
  stroke-width: 14;
  pointer-events: stroke;
  cursor: pointer;
}

.agent-skill-canvas__edge--selected {
  stroke: var(--brand);
  stroke-width: 3;
}

.agent-skill-canvas__edge--temporary {
  stroke-dasharray: 8 7;
  pointer-events: none;
}

.agent-skill-canvas__edges marker path {
  fill: color-mix(in srgb, var(--brand) 70%, var(--text-muted));
}

.agent-skill-canvas__edge-delete {
  cursor: pointer;
  pointer-events: all;
}

.agent-skill-canvas__edge-delete circle {
  fill: var(--surface);
  stroke: color-mix(in srgb, var(--brand) 72%, var(--text-muted));
  stroke-width: 1.5;
  filter: drop-shadow(0 4px 8px rgba(15, 23, 42, 0.14));
}

.agent-skill-canvas__edge-delete path {
  fill: none;
  stroke: var(--text-muted);
  stroke-linecap: round;
  stroke-width: 1.7;
}

.agent-skill-canvas__edge-delete:hover circle {
  stroke: var(--danger, #dc2626);
}

.agent-skill-canvas__edge-delete:hover path {
  stroke: var(--danger, #dc2626);
}

.agent-skill-canvas-node {
  position: absolute;
  z-index: 2;
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

.agent-skill-canvas-node__connector {
  position: absolute;
  z-index: 2;
  width: 14px;
  height: 14px;
  border: 2px solid var(--surface);
  border-radius: 50%;
  background: color-mix(in srgb, var(--brand) 78%, var(--text-muted));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--brand) 70%, transparent);
  cursor: crosshair;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.15s ease, box-shadow 0.15s ease, background-color 0.15s ease;
}

.agent-skill-canvas-node__connector--top {
  left: 50%;
  top: -8px;
  transform: translateX(-50%);
}

.agent-skill-canvas-node__connector--right {
  right: -8px;
  top: 50%;
  transform: translateY(-50%);
}

.agent-skill-canvas-node__connector--bottom {
  left: 50%;
  bottom: -8px;
  transform: translateX(-50%);
}

.agent-skill-canvas-node__connector--left {
  left: -8px;
  top: 50%;
  transform: translateY(-50%);
}

.agent-skill-canvas-node__connector:hover,
.agent-skill-canvas-node__connector--active {
  background: var(--brand);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--brand-soft) 76%, transparent);
}

.agent-skill-canvas-node:hover .agent-skill-canvas-node__connector,
.agent-skill-canvas-node--selected .agent-skill-canvas-node__connector,
.agent-skill-canvas-node--connecting .agent-skill-canvas-node__connector {
  opacity: 1;
  pointer-events: auto;
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
