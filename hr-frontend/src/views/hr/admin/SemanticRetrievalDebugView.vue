<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { Refresh, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { debugSemanticRetrieval } from '@/api/agentSkill'
import type {
  SemanticMemoryDebugItem,
  SemanticRetrievalDebugResult,
  SemanticSkillDebugItem,
} from '@/types/agentSkill'

const AGENT_TYPE_OPTIONS = [
  { value: 'hr_recruiting_agent', label: 'HR 招聘助手' },
  { value: 'candidate_assistant', label: '候选人 AI 助手' },
  { value: 'custom', label: '自定义 Agent' },
]

const QUERY_EXAMPLES = [
  '候选人有 5 年 Vue 和 TypeScript 经验，最近负责招聘系统前端架构。请判断他和高级前端工程师岗位是否匹配。',
  '帮我生成这位候选人的面试关注点，重点验证项目复杂度、协作能力和岗位匹配风险。',
  '分析当前岗位的投递情况，找出可能影响招聘转化的原因，并给出下一步建议。',
]

const loading = ref(false)
const result = ref<SemanticRetrievalDebugResult | null>(null)
const requestDurationMs = ref<number | null>(null)
const expandedSkillIds = ref<number[]>([])
const expandedMemoryIds = ref<number[]>([])

const form = reactive({
  query: '',
  agent_type: 'hr_recruiting_agent',
  job_id: undefined as number | undefined,
  application_id: undefined as number | undefined,
  limit: 5,
})

const skillRows = computed<SemanticSkillDebugItem[]>(() => result.value?.skills || [])
const memoryRows = computed<SemanticMemoryDebugItem[]>(() => result.value?.memories || [])
const hasResult = computed(() => Boolean(result.value))
const selectedAgentLabel = computed(() => AGENT_TYPE_OPTIONS.find((item) => item.value === form.agent_type)?.label || form.agent_type)
const runStatusText = computed(() => {
  if (loading.value) return '运行中'
  if (result.value) return '已完成'
  return '未运行'
})
const embeddingStatusText = computed(() => {
  if (!result.value) return '待检测'
  return result.value.embedding_available ? 'Embedding 可用' : '规则回退'
})
const embeddingStatusType = computed(() => {
  if (!result.value) return 'info'
  return result.value.embedding_available ? 'success' : 'warning'
})

const fallbackMessage = computed(() => {
  const reason = result.value?.fallback_reason?.trim()
  if (!reason) return ''
  if (reason.includes('embedding provider unavailable') || reason.includes('embedding retrieval unavailable')) {
    return '当前未启用向量 Embedding Provider，已自动降级为规则匹配与上下文排序。Skill / Memory 结果仍可用于调试，但分数不是向量相似度。'
  }
  return reason
})
const highestScore = computed(() => {
  const scores = [...skillRows.value, ...memoryRows.value]
    .map((item) => item.score)
    .filter((score): score is number => typeof score === 'number')
  if (!scores.length) return undefined
  return Math.max(...scores)
})

const formatScore = (value?: number) => (typeof value === 'number' ? value.toFixed(2) : '-')
const formatDuration = (value: number | null) => (typeof value === 'number' ? `${value}ms` : '-')
const skillTitle = (item: SemanticSkillDebugItem) => item.display_name || item.name || `Skill #${item.id}`
const memoryScopeText = (item: SemanticMemoryDebugItem) => `${item.scope_type || '-'} #${item.scope_id || '-'}`
const isSkillExpanded = (id: number) => expandedSkillIds.value.includes(id)
const isMemoryExpanded = (id: number) => expandedMemoryIds.value.includes(id)

const toggleId = (values: number[], id: number) => (
  values.includes(id) ? values.filter((item) => item !== id) : [...values, id]
)

const toggleSkill = (id: number) => {
  expandedSkillIds.value = toggleId(expandedSkillIds.value, id)
}

const toggleMemory = (id: number) => {
  expandedMemoryIds.value = toggleId(expandedMemoryIds.value, id)
}

const applyExample = (example: string) => {
  form.query = example
}

const runDebug = async () => {
  const query = form.query.trim()
  if (!query) {
    ElMessage.warning('请输入查询内容')
    return
  }
  loading.value = true
  const startedAt = performance.now()
  try {
    result.value = await debugSemanticRetrieval({
      query,
      agent_type: form.agent_type,
      job_id: form.job_id,
      application_id: form.application_id,
      limit: form.limit,
    })
    requestDurationMs.value = Math.round(performance.now() - startedAt)
    expandedSkillIds.value = []
    expandedMemoryIds.value = []
  } catch (error) {
    console.error(error)
    requestDurationMs.value = Math.round(performance.now() - startedAt)
    ElMessage.error('语义召回调试失败')
  } finally {
    loading.value = false
  }
}

const reset = () => {
  form.query = ''
  form.agent_type = 'hr_recruiting_agent'
  form.job_id = undefined
  form.application_id = undefined
  form.limit = 5
  result.value = null
  requestDurationMs.value = null
  expandedSkillIds.value = []
  expandedMemoryIds.value = []
}
</script>

<template>
  <div class="semantic-debug-view">
    <main class="debug-workbench" v-loading="loading">
      <header class="workbench-header">
        <div class="workbench-header__copy">
          <p class="page-kicker">SEMANTIC RETRIEVAL</p>
          <h1 class="page-title">语义召回调试</h1>
          <p class="page-desc">用真实业务问题验证 Agent Skill 与 AI Memory 的召回效果</p>
        </div>
        <div class="workbench-header__actions">
          <el-button :icon="Refresh" @click="reset">重置</el-button>
          <el-button type="primary" :icon="Search" :loading="loading" @click="runDebug">运行测试</el-button>
        </div>
      </header>

      <section class="config-shell" aria-label="测试配置区">
        <div class="query-editor">
          <div class="section-title-row">
            <div>
              <h2>测试 Query</h2>
              <p>输入一次真实 HR 问题，系统将按当前参数执行 Skill 与 Memory 召回。</p>
            </div>
            <span>Ctrl + Enter 运行</span>
          </div>
          <el-input
            v-model="form.query"
            class="query-textarea"
            type="textarea"
            :rows="5"
            maxlength="800"
            show-word-limit
            resize="none"
            placeholder="例如：这位候选人与高级前端工程师岗位是否匹配？请说明证据、风险和面试关注点。"
            @keyup.ctrl.enter="runDebug"
          />
          <div class="preset-block">
            <div class="preset-block__label">快捷 Query</div>
            <div class="preset-chip-row">
              <button
                v-for="example in QUERY_EXAMPLES"
                :key="example"
                type="button"
                class="preset-chip"
                @click="applyExample(example)"
              >
                {{ example }}
              </button>
            </div>
          </div>
        </div>

        <aside class="params-panel" aria-label="召回参数">
          <div class="section-title-row section-title-row--compact">
            <div>
              <h2>召回参数</h2>
              <p>{{ selectedAgentLabel }}</p>
            </div>
          </div>
          <label class="field-group">
            <span>助手</span>
            <el-select v-model="form.agent_type" placeholder="Agent 类型" filterable>
              <el-option v-for="item in AGENT_TYPE_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </label>
          <div class="param-grid">
            <label class="field-group">
              <span>岗位 ID</span>
              <el-input-number v-model="form.job_id" :min="0" :controls="false" placeholder="可选" />
            </label>
            <label class="field-group">
              <span>投递 ID</span>
              <el-input-number v-model="form.application_id" :min="0" :controls="false" placeholder="可选" />
            </label>
            <label class="field-group field-group--full">
              <span>Top K</span>
              <el-input-number v-model="form.limit" :min="1" :max="20" controls-position="right" />
            </label>
          </div>
          <div class="params-status-grid">
            <div>
              <span>运行状态</span>
              <strong>{{ runStatusText }}</strong>
            </div>
            <div>
              <span>Skill</span>
              <strong>{{ skillRows.length }}</strong>
            </div>
            <div>
              <span>Memory</span>
              <strong>{{ memoryRows.length }}</strong>
            </div>
          </div>
        </aside>
      </section>

      <section class="overview-strip" aria-label="召回结果概览">
        <div class="overview-item">
          <span>测试状态</span>
          <strong>{{ runStatusText }}</strong>
        </div>
        <div class="overview-item">
          <span>Skill 命中</span>
          <strong>{{ skillRows.length }}</strong>
        </div>
        <div class="overview-item">
          <span>Memory 命中</span>
          <strong>{{ memoryRows.length }}</strong>
        </div>
        <div class="overview-item">
          <span>最高匹配分数</span>
          <strong>{{ formatScore(highestScore) }}</strong>
        </div>
        <div class="overview-item">
          <span>请求耗时</span>
          <strong>{{ formatDuration(requestDurationMs) }}</strong>
        </div>
        <div class="overview-item overview-item--status">
          <span>召回模式</span>
          <el-tag :type="embeddingStatusType" effect="plain">{{ embeddingStatusText }}</el-tag>
        </div>
      </section>

      <el-alert
        v-if="fallbackMessage"
        type="warning"
        show-icon
        :closable="false"
        :title="fallbackMessage"
      />

      <section class="result-compare" aria-label="召回结果对比">
        <div class="result-panel">
          <div class="result-panel__header">
            <div>
              <h2>Agent Skill 召回</h2>
              <p>展示匹配到的数据库版 SKILL.md、分类、分数和触发原因。</p>
            </div>
            <el-tag size="small" type="info">{{ skillRows.length }} 条</el-tag>
          </div>
          <div v-if="skillRows.length" class="result-list">
            <article v-for="item in skillRows" :key="item.id" class="result-card">
              <div class="result-card__top">
                <div>
                  <h3>{{ skillTitle(item) }}</h3>
                  <p>#{{ item.id }} · {{ item.name }}</p>
                </div>
                <div class="score-pill">{{ formatScore(item.score) }}</div>
              </div>
              <div class="result-card__meta">
                <span>{{ item.category || 'general' }}</span>
                <span v-if="item.scenario">{{ item.scenario }}</span>
                <span>P{{ item.priority ?? 0 }}</span>
              </div>
              <p class="result-card__summary">{{ item.reason || 'metadata and content match' }}</p>
              <div v-if="item.semantic_tags?.length" class="tag-row">
                <el-tag v-for="tag in item.semantic_tags" :key="tag" size="small" effect="plain">{{ tag }}</el-tag>
              </div>
              <button type="button" class="detail-toggle" @click="toggleSkill(item.id)">
                {{ isSkillExpanded(item.id) ? '收起详情' : '查看详情' }}
              </button>
              <div v-if="isSkillExpanded(item.id)" class="detail-block">
                <div><span>Skill ID</span><strong>{{ item.id }}</strong></div>
                <div><span>匹配分数</span><strong>{{ formatScore(item.score) }}</strong></div>
                <div><span>召回原因</span><strong>{{ item.reason || '-' }}</strong></div>
              </div>
            </article>
          </div>
          <div v-else class="empty-state">
            <h3>{{ hasResult ? '未命中 Agent Skill' : '等待运行测试' }}</h3>
            <p>{{ hasResult ? '本次查询没有匹配到可用 Skill。可以检查 Skill 是否启用、语义标签是否覆盖该场景，或调整 Query 表达。' : '运行后这里会展示命中的 Agent Skill、匹配分数、分类、触发原因和语义标签。' }}</p>
          </div>
        </div>

        <div class="result-panel">
          <div class="result-panel__header">
            <div>
              <h2>AI Memory 召回</h2>
              <p>展示按 HR、岗位、投递等 scope 召回的长期记忆和排序依据。</p>
            </div>
            <el-tag size="small" type="info">{{ memoryRows.length }} 条</el-tag>
          </div>
          <div v-if="memoryRows.length" class="result-list">
            <article v-for="item in memoryRows" :key="item.id" class="result-card">
              <div class="result-card__top">
                <div>
                  <h3>{{ memoryScopeText(item) }}</h3>
                  <p>{{ item.memory_type || '-' }} · {{ item.source || '-' }}</p>
                </div>
                <div class="score-pill">{{ formatScore(item.score) }}</div>
              </div>
              <div class="result-card__meta">
                <span>重要度 {{ formatScore(item.importance) }}</span>
                <span>置信度 {{ formatScore(item.confidence) }}</span>
                <span v-if="item.created_at">{{ item.created_at }}</span>
              </div>
              <p class="result-card__summary">{{ item.content }}</p>
              <button type="button" class="detail-toggle" @click="toggleMemory(item.id)">
                {{ isMemoryExpanded(item.id) ? '收起详情' : '查看详情' }}
              </button>
              <div v-if="isMemoryExpanded(item.id)" class="detail-block">
                <div><span>Memory ID</span><strong>{{ item.id }}</strong></div>
                <div><span>Scope</span><strong>{{ item.scope_type }} #{{ item.scope_id }}</strong></div>
                <div><span>召回原因</span><strong>{{ item.reason || '-' }}</strong></div>
              </div>
            </article>
          </div>
          <div v-else class="empty-state">
            <h3>{{ hasResult ? '未命中 AI Memory' : '等待运行测试' }}</h3>
            <p>{{ hasResult ? '本次查询没有召回长期记忆。可以补充岗位 ID 或投递 ID，确认对应 scope 下是否存在可召回内容。' : '运行后这里会展示命中的 Memory、scope、来源、匹配分数、重要度和召回原因。' }}</p>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped>
.semantic-debug-view {
  height: 100%;
  min-height: 0;
  overflow: auto;
  background: var(--el-bg-color-page);
}

.debug-workbench {
  display: grid;
  gap: 16px;
  width: min(100%, 1500px);
  margin: 0 auto;
  padding: 16px 24px 24px;
}

.workbench-header,
.config-shell,
.overview-strip,
.result-compare {
  display: grid;
}

.workbench-header {
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 16px;
  padding: 12px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.workbench-header__actions,
.preset-chip-row,
.result-card__top,
.result-card__meta,
.tag-row,
.detail-block {
  display: flex;
}

.workbench-header__actions {
  gap: 8px;
}

.page-kicker {
  margin: 0 0 4px;
  color: var(--el-color-primary);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0;
}

.page-title {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 22px;
  font-weight: 700;
  line-height: 1.2;
}

.page-desc {
  margin: 4px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.config-shell {
  grid-template-columns: minmax(0, 1fr) clamp(320px, 24vw, 360px);
  gap: 16px;
  align-items: start;
}

.query-editor,
.params-panel,
.result-panel {
  display: grid;
  align-content: start;
  gap: 12px;
  min-width: 0;
  padding: 16px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
}

.section-title-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.section-title-row h2,
.result-panel__header h2 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 15px;
  font-weight: 700;
}

.section-title-row p,
.result-panel__header p {
  margin: 4px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.section-title-row > span {
  flex: 0 0 auto;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.section-title-row--compact {
  align-items: center;
}

.query-textarea :deep(.el-textarea__inner) {
  min-height: 136px !important;
  max-height: 160px;
  line-height: 1.55;
}

.preset-block {
  display: grid;
  gap: 8px;
}

.preset-block__label,
.field-group > span,
.overview-item > span {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.preset-chip-row {
  flex-wrap: wrap;
  gap: 8px;
}

.preset-chip {
  max-width: 100%;
  padding: 6px 10px;
  border: 1px solid var(--el-border-color);
  border-radius: 999px;
  background: var(--el-fill-color-blank);
  color: var(--el-text-color-regular);
  cursor: pointer;
  font-size: 12px;
  line-height: 1.4;
  text-align: left;
}

.preset-chip:hover {
  border-color: var(--el-color-primary-light-5);
  color: var(--el-color-primary);
}

.field-group {
  display: grid;
  gap: 6px;
}

.field-group .el-select,
.field-group .el-input-number {
  width: 100%;
}

.param-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.field-group--full {
  grid-column: 1 / -1;
}

.params-status-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  padding-top: 4px;
}

.params-status-grid > div {
  display: grid;
  gap: 3px;
  padding: 8px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.params-status-grid span {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.params-status-grid strong {
  color: var(--el-text-color-primary);
  font-size: 14px;
}

.overview-strip {
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 1px;
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-border-color-lighter);
}

.overview-item {
  display: grid;
  gap: 4px;
  min-width: 0;
  padding: 12px;
  background: var(--el-bg-color);
}

.overview-item strong {
  overflow: hidden;
  color: var(--el-text-color-primary);
  font-size: 16px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.overview-item--status strong {
  font-size: 14px;
}

.result-compare {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  align-items: start;
}

.result-panel__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.result-list {
  display: grid;
  gap: 12px;
  max-height: 560px;
  overflow: auto;
  padding-right: 2px;
}

.result-card {
  display: grid;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
}

.result-card__top {
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.result-card__top h3 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 14px;
  font-weight: 700;
  line-height: 1.4;
}

.result-card__top p {
  margin: 3px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.score-pill {
  flex: 0 0 auto;
  padding: 3px 8px;
  border-radius: 999px;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  font-family: var(--font-mono, monospace);
  font-size: 12px;
  font-weight: 700;
}

.result-card__meta,
.tag-row,
.detail-block {
  flex-wrap: wrap;
  gap: 6px;
}

.result-card__meta span {
  padding: 2px 7px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.result-card__summary {
  display: -webkit-box;
  margin: 0;
  overflow: hidden;
  color: var(--el-text-color-regular);
  font-size: 12px;
  line-height: 1.6;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

.detail-toggle {
  justify-self: start;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--el-color-primary);
  cursor: pointer;
  font-size: 12px;
}

.detail-block {
  padding: 10px;
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.detail-block div {
  display: grid;
  gap: 2px;
  min-width: 120px;
}

.detail-block span {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.detail-block strong {
  color: var(--el-text-color-primary);
  font-size: 12px;
  font-weight: 600;
}

.empty-state {
  display: grid;
  gap: 6px;
  padding: 24px;
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
}

.empty-state h3 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 14px;
}

.empty-state p {
  margin: 0;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.6;
}

@media (max-width: 1366px) {
  .debug-workbench {
    width: 100%;
    padding-inline: 16px;
  }

  .config-shell {
    grid-template-columns: minmax(0, 1fr) 320px;
  }
}

@media (max-width: 1080px) {
  .config-shell,
  .result-compare {
    grid-template-columns: 1fr;
  }

  .overview-strip {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .debug-workbench {
    padding: 12px;
  }

  .workbench-header {
    grid-template-columns: 1fr;
  }

  .workbench-header__actions {
    justify-content: flex-end;
  }

  .param-grid,
  .params-status-grid,
  .overview-strip {
    grid-template-columns: 1fr;
  }
}
</style>
