<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getAgentRuns, getToolTraces } from '@/api/ai'
import type { AgentRunItem, AgentRunStepItem, ToolTraceItem } from '@/types/ai'

const props = defineProps<{
  sessionId: number | null
  visible: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
}>()

const traces = ref<ToolTraceItem[]>([])
const runs = ref<AgentRunItem[]>([])
const loading = ref(false)
const expandedArgs = ref<Set<number>>(new Set())
const expandedResult = ref<Set<number>>(new Set())
const expandedStepInput = ref<Set<number>>(new Set())
const expandedStepOutput = ref<Set<number>>(new Set())

const loadTraces = async () => {
  if (!props.sessionId) {
    traces.value = []
    runs.value = []
    return
  }
  loading.value = true
  try {
    const [runData, traceData] = await Promise.all([
      getAgentRuns(props.sessionId).catch(() => ({ list: [] as AgentRunItem[] })),
      getToolTraces(props.sessionId),
    ])
    runs.value = runData.list || []
    traces.value = traceData.list || []
  } catch (e) {
    traces.value = []
    ElMessage.error('加载执行轨迹失败，请稍后重试')
  } finally {
    loading.value = false
  }
}

const close = () => {
  emit('update:visible', false)
}

const toggleArgs = (id: number) => {
  const next = new Set(expandedArgs.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedArgs.value = next
}

const toggleResult = (id: number) => {
  const next = new Set(expandedResult.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedResult.value = next
}

const toggleStepInput = (id: number) => {
  const next = new Set(expandedStepInput.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedStepInput.value = next
}

const toggleStepOutput = (id: number) => {
  const next = new Set(expandedStepOutput.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedStepOutput.value = next
}

const formatTime = (iso: string): string => {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return iso
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

const formatJson = (json: string): string => {
  try {
    return JSON.stringify(JSON.parse(json), null, 2)
  } catch {
    return json
  }
}

const statusTagType = (status: string): 'success' | 'warning' | 'danger' | 'info' | 'primary' => {
  if (status === 'succeeded' || status === 'success') return 'success'
  if (status === 'failed' || status === 'error') return 'danger'
  if (status === 'partial' || status === 'fallback') return 'warning'
  if (status === 'canceled') return 'info'
  return 'primary'
}

const stepTitle = (step: AgentRunStepItem): string => {
  if (step.tool_name) return step.tool_name
  if (step.capability_key) return step.capability_key
  return step.step_type
}

const previewJson = (json: string, expanded: boolean): string => {
  const text = formatJson(json)
  return expanded ? text : text.slice(0, 240)
}

// Reload when drawer opens with a valid session
watch(() => props.visible, (val) => {
  if (val) {
    loadTraces()
  }
})

// Reload when session changes (panel already open)
watch(() => props.sessionId, () => {
  if (props.visible && props.sessionId) {
    loadTraces()
  }
})
</script>

<template>
  <el-drawer
    :model-value="visible"
    @update:model-value="(val: boolean) => emit('update:visible', val)"
    title="Agent 执行轨迹"
    size="480px"
    :close-on-click-modal="false"
  >
    <div class="trace-panel" v-loading="loading">
      <template v-if="runs.length === 0 && traces.length === 0 && !loading">
        <el-empty description="本次会话暂无 Agent 执行记录" />
      </template>

      <div v-if="runs.length > 0" class="run-list">
        <section
          v-for="run in runs"
          :key="run.id"
          class="run-item"
        >
          <div class="run-item__header">
            <div>
              <div class="run-item__title">{{ run.agent_name || 'HR Agent' }}</div>
              <div class="run-item__meta">
                {{ run.model_name || '未记录模型' }} · {{ formatTime(run.started_at || run.created_at) }}
              </div>
            </div>
            <el-tag :type="statusTagType(run.status)" size="small">
              {{ run.status }}
            </el-tag>
          </div>

          <div v-if="run.plan_json" class="trace-item__section">
            <div class="trace-item__label">Plan：</div>
            <div class="trace-item__code">
              <pre>{{ formatJson(run.plan_json) }}</pre>
            </div>
          </div>

          <el-alert
            v-if="run.error_message"
            class="run-item__alert"
            :title="run.error_message"
            :type="run.status === 'partial' ? 'warning' : 'error'"
            :closable="false"
            show-icon
          />

          <el-timeline class="run-steps">
            <el-timeline-item
              v-for="step in run.steps"
              :key="step.id"
              :color="step.status === 'failed' ? 'var(--el-color-danger)' : 'var(--el-color-primary)'"
              :timestamp="formatTime(step.started_at || step.created_at)"
            >
              <div class="trace-item">
                <div class="trace-item__header">
                  <span class="trace-item__name" :class="{ 'trace-item__name--error': step.status === 'failed' }">
                    {{ stepTitle(step) }}
                  </span>
                  <el-tag :type="statusTagType(step.status)" size="small" effect="plain">
                    {{ step.step_type }} · {{ step.status }}
                  </el-tag>
                </div>
                <div class="trace-item__duration">
                  耗时：<strong>{{ step.duration_ms || 0 }}</strong> ms
                  <span v-if="step.capability_source || step.capability_key">
                    · {{ step.capability_source }} {{ step.capability_key }}
                  </span>
                </div>

                <div v-if="step.input_json" class="trace-item__section">
                  <div class="trace-item__label">输入：</div>
                  <div class="trace-item__code">
                    <pre>{{ previewJson(step.input_json, expandedStepInput.has(step.id)) }}</pre>
                  </div>
                  <button
                    v-if="formatJson(step.input_json).length > 240"
                    class="trace-item__toggle"
                    @click="toggleStepInput(step.id)"
                  >
                    {{ expandedStepInput.has(step.id) ? '收起' : '展开全部' }}
                  </button>
                </div>

                <div v-if="step.output_json" class="trace-item__section">
                  <div class="trace-item__label">输出：</div>
                  <div class="trace-item__code">
                    <pre>{{ previewJson(step.output_json, expandedStepOutput.has(step.id)) }}</pre>
                  </div>
                  <button
                    v-if="formatJson(step.output_json).length > 240"
                    class="trace-item__toggle"
                    @click="toggleStepOutput(step.id)"
                  >
                    {{ expandedStepOutput.has(step.id) ? '收起' : '展开全部' }}
                  </button>
                </div>

                <el-alert
                  v-if="step.error_message"
                  :title="step.error_message"
                  type="error"
                  :closable="false"
                  show-icon
                />
              </div>
            </el-timeline-item>
          </el-timeline>
        </section>
      </div>

      <div v-if="traces.length > 0" class="legacy-traces">
        <div class="legacy-traces__title">兼容工具调用明细</div>
        <el-timeline>
          <el-timeline-item
            v-for="item in traces"
            :key="item.id"
            :color="item.error_msg ? 'var(--el-color-danger)' : 'var(--el-color-primary)'"
            :timestamp="formatTime(item.created_at)"
          >
            <div class="trace-item">
              <!-- Tool name -->
              <div class="trace-item__header">
                <span class="trace-item__name" :class="{ 'trace-item__name--error': !!item.error_msg }">
                  {{ item.tool_name }}
                </span>
                <el-tag
                  v-if="item.error_msg"
                  type="danger"
                  size="small"
                  effect="dark"
                >失败</el-tag>
                <el-tag
                  v-else
                  type="success"
                  size="small"
                  effect="plain"
                >成功</el-tag>
              </div>

              <!-- Duration -->
              <div class="trace-item__duration">
                耗时：<strong>{{ item.duration_ms }}</strong> ms
              </div>

              <!-- Args -->
              <div v-if="item.args_json" class="trace-item__section">
                <div class="trace-item__label">入参：</div>
                <div
                  class="trace-item__code"
                  :class="{ 'trace-item__code--collapsed': item.args_json.length > 200 && !expandedArgs.has(item.id) }"
                >
                  <pre>{{ expandedArgs.has(item.id) ? formatJson(item.args_json) : formatJson(item.args_json).slice(0, 200) }}</pre>
                </div>
                <button
                  v-if="item.args_json.length > 200"
                  class="trace-item__toggle"
                  @click="toggleArgs(item.id)"
                >
                  {{ expandedArgs.has(item.id) ? '收起' : '展开全部' }}
                </button>
              </div>

              <!-- Result -->
              <div v-if="item.result_content" class="trace-item__section">
                <div class="trace-item__label">结果：</div>
                <div
                  class="trace-item__result"
                  :class="{ 'trace-item__result--collapsed': item.result_content.length > 200 && !expandedResult.has(item.id) }"
                >
                  {{ expandedResult.has(item.id) ? item.result_content : item.result_content.slice(0, 200) }}
                </div>
                <button
                  v-if="item.result_content.length > 200"
                  class="trace-item__toggle"
                  @click="toggleResult(item.id)"
                >
                  {{ expandedResult.has(item.id) ? '收起' : '展开全部' }}
                </button>
              </div>

              <!-- Error message -->
              <div v-if="item.error_msg" class="trace-item__error">
                <el-alert
                  :title="item.error_msg"
                  type="error"
                  :closable="false"
                  show-icon
                />
              </div>
            </div>
          </el-timeline-item>
        </el-timeline>
      </div>
    </div>
  </el-drawer>
</template>

<style scoped>
.trace-panel {
  height: 100%;
  overflow-y: auto;
}

.run-list {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.run-item {
  border-bottom: 1px solid var(--el-border-color-lighter);
  padding-bottom: 16px;
}

.run-item__header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.run-item__title {
  font-size: 15px;
  font-weight: 650;
  color: var(--el-text-color-primary);
}

.run-item__meta {
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.run-item__alert {
  margin-bottom: 10px;
}

.run-steps {
  margin-top: 8px;
}

.legacy-traces {
  margin-top: 18px;
}

.legacy-traces__title {
  margin-bottom: 10px;
  font-size: 13px;
  font-weight: 650;
  color: var(--el-text-color-primary);
}

.trace-item {
  font-size: 13px;
  line-height: 1.5;
}

.trace-item__header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.trace-item__name {
  font-weight: 600;
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.trace-item__name--error {
  color: var(--el-color-danger);
}

.trace-item__duration {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  margin-bottom: 8px;
}

.trace-item__section {
  margin-bottom: 6px;
}

.trace-item__label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 2px;
}

.trace-item__code {
  background: var(--el-fill-color-light);
  border-radius: 4px;
  padding: 6px 8px;
  overflow-x: auto;
}

.trace-item__code pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: 'SF Mono', 'Fira Code', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.4;
  color: var(--el-text-color-regular);
}

.trace-item__code--collapsed pre {
  max-height: 80px;
  overflow: hidden;
}

.trace-item__result {
  background: var(--el-fill-color-light);
  border-radius: 4px;
  padding: 6px 8px;
  font-size: 12px;
  color: var(--el-text-color-regular);
  word-break: break-all;
}

.trace-item__result--collapsed {
  max-height: 80px;
  overflow: hidden;
}

.trace-item__toggle {
  background: none;
  border: none;
  color: var(--el-color-primary);
  cursor: pointer;
  font-size: 12px;
  padding: 2px 0;
  margin-top: 2px;
}

.trace-item__toggle:hover {
  color: var(--el-color-primary-light-3);
}

.trace-item__error {
  margin-top: 6px;
}
</style>
