<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getToolTraces } from '@/api/ai'
import type { ToolTraceItem } from '@/types/ai'

const props = defineProps<{
  sessionId: number | null
  visible: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
}>()

const traces = ref<ToolTraceItem[]>([])
const loading = ref(false)
const expandedArgs = ref<Set<number>>(new Set())
const expandedResult = ref<Set<number>>(new Set())

const loadTraces = async () => {
  if (!props.sessionId) {
    traces.value = []
    return
  }
  loading.value = true
  try {
    const data = await getToolTraces(props.sessionId)
    traces.value = data.list || []
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
      <template v-if="traces.length === 0 && !loading">
        <el-empty description="本次会话无工具调用" />
      </template>

      <el-timeline v-else>
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
  </el-drawer>
</template>

<style scoped>
.trace-panel {
  height: 100%;
  overflow-y: auto;
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
