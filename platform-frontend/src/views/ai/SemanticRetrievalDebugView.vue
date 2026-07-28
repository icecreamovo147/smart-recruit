<script setup lang="ts">
import { t } from '@shared/i18n'
import { computed, reactive, ref, watch } from 'vue'
import { EditPen, Refresh, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { debugSemanticRetrieval } from '@/api/agentSkill'
import { createPlatformMemory, listPlatformMemories, revokePlatformMemory } from '@/api/memory'
import { PagePanel } from '@/components/admin-console'
import type {
  SemanticMemoryDebugItem,
  SemanticRetrievalDebugResult,
  SemanticSkillDebugItem,
} from '@shared/types/agentSkill'
import {
  MEMORY_OWNER_ROLE_CANDIDATE,
  MEMORY_OWNER_ROLE_HR,
  type MemoryInfo,
} from '@shared/types/memory'

const AGENT_TYPE_OPTIONS = [
  { value: 'hr_recruiting_agent', label: 'HR 招聘助手' },
  { value: 'candidate_assistant', label: '候选人 AI 助手' },
  { value: 'custom', label: '自定义 Agent' },
]

const OWNER_ROLE_OPTIONS = [
  { value: MEMORY_OWNER_ROLE_HR, label: 'HR' },
  { value: MEMORY_OWNER_ROLE_CANDIDATE, label: '候选人' },
]

const MEMORY_SCOPE_OPTIONS = [
  { value: 'hr', label: 'hr（HR 全局）' },
  { value: 'job', label: 'job（岗位）' },
  { value: 'application', label: 'application（投递）' },
  { value: 'candidate', label: 'candidate（候选人）' },
  { value: 'user', label: 'user（用户）' },
]

// TASK-FU-003：pool_confidence 枚举色彩 / 文案映射
const POOL_CONFIDENCE_META: Record<string, { type: 'success' | 'info' | 'warning' | 'danger'; label: string }> = {
  high: { type: 'success', label: '高' },
  medium: { type: 'info', label: '中' },
  low: { type: 'warning', label: '低' },
  none: { type: 'info', label: '无' },
}

const RELEVANCE_MODE_META: Record<string, { type: 'success' | 'warning'; label: string }> = {
  vector_lexical_metadata: { type: 'success', label: '向量+规则' },
  lexical_metadata: { type: 'warning', label: '仅规则' },
}

// TASK-FU-006：breakdown 字段 tooltip 文案
const BREAKDOWN_TOOLTIPS: Record<string, string> = {
  vector: 'embedding cosine ∈ [0, 1]',
  lexical: 'keyword hit ratio ∈ [0, 1]',
  metadata: 'scope / category hit ∈ [0, 1]',
  relevance: 'weighted sum ∈ [0, 1]',
  'business boost': 'relevance × boost ∈ [0, 1.5]',
  'final rank': 'final_rank_score ∈ [0, 1.5]',
}

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
const correctionDrawerVisible = ref(false)
const correctionLoading = ref(false)
const correctionSaving = ref(false)
const correctionRows = ref<MemoryInfo[]>([])
const correctionTotal = ref(0)

const form = reactive({
  query: '',
  agent_type: 'hr_recruiting_agent',
  job_id: undefined as number | undefined,
  application_id: undefined as number | undefined,
  limit: 5,
  owner_role: MEMORY_OWNER_ROLE_HR,
  owner_id: undefined as number | undefined,
})

const correctionForm = reactive({
  content: '',
  scope_type: 'hr',
  scope_id: undefined as number | undefined,
  memory_type: 'preference',
  confirm_high_pii: false,
})

const skillRows = computed<SemanticSkillDebugItem[]>(() => result.value?.skills || [])
const memoryRows = computed<SemanticMemoryDebugItem[]>(() => result.value?.memories || [])
const hasResult = computed(() => Boolean(result.value))

// TASK-FU-003：pool_confidence 渲染辅助
const skillPoolConfidence = computed(() => result.value?.skill_pool_confidence || '')
const memoryPoolConfidence = computed(() => result.value?.memory_pool_confidence || '')
const skillPoolConfidenceMeta = computed(() => POOL_CONFIDENCE_META[skillPoolConfidence.value] || null)
const memoryPoolConfidenceMeta = computed(() => POOL_CONFIDENCE_META[memoryPoolConfidence.value] || null)

function relevanceModeMeta(mode: string | undefined) {
  if (!mode) return null
  return RELEVANCE_MODE_META[mode] || { type: 'info' as const, label: mode }
}

function formatBreakdownScore(value: number | undefined, digits = 2): string {
  return typeof value === 'number' ? value.toFixed(digits) : '-'
}
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

const embeddingProviderText = computed(() => result.value?.embedding_provider || '-')
const embeddingModelText = computed(() => result.value?.embedding_model || '-')
const embeddingDimText = computed(() => result.value?.embedding_dim != null ? `${result.value.embedding_dim}` : '-')
const candidateCountText = computed(() => result.value?.candidate_count != null ? `${result.value.candidate_count}` : '-')
const embeddingLatencyText = computed(() => result.value?.query_embedding_latency_ms != null ? `${result.value.query_embedding_latency_ms}ms` : '-')

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
    .map((item) => item.vector_score)
    .filter((score): score is number => typeof score === 'number')
  if (!scores.length) return undefined
  return Math.max(...scores)
})

const formatScore = (value?: number) => (typeof value === 'number' ? value.toFixed(2) : '-')
const embeddingScore = (item: SemanticSkillDebugItem | SemanticMemoryDebugItem) => item.vector_score
const finalRankScore = (item: SemanticSkillDebugItem | SemanticMemoryDebugItem) => item.final_rank_score ?? item.score
const formatDuration = (value: number | null) => (typeof value === 'number' ? `${value}ms` : '-')
const skillTitle = (item: SemanticSkillDebugItem) => item.display_name || item.name || `Skill #${item.skill_id}`
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

const selectedOwnerLabel = computed(() => OWNER_ROLE_OPTIONS.find((item) => item.value === form.owner_role)?.label || `${form.owner_role}`)

watch(() => form.agent_type, (agentType) => {
  if (agentType === 'candidate_assistant') {
    form.owner_role = MEMORY_OWNER_ROLE_CANDIDATE
  } else if (agentType === 'hr_recruiting_agent') {
    form.owner_role = MEMORY_OWNER_ROLE_HR
  }
})

const ownerReady = computed(() => Boolean(form.owner_id && form.owner_id > 0))

const runDebug = async () => {
  const query = form.query.trim()
  if (!query) {
    ElMessage.warning(t('common.invalid_request'))
    return
  }
  if (!ownerReady.value) {
    ElMessage.warning(t('common.invalid_request'))
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
      owner_role: form.owner_role,
      owner_id: form.owner_id,
    })
    requestDurationMs.value = Math.round(performance.now() - startedAt)
    expandedSkillIds.value = []
    expandedMemoryIds.value = []
  } catch (error) {
    console.error(error)
    requestDurationMs.value = Math.round(performance.now() - startedAt)
    ElMessage.error(t('frontend.operation_failed'))
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
  form.owner_role = MEMORY_OWNER_ROLE_HR
  form.owner_id = undefined
  result.value = null
  requestDurationMs.value = null
  expandedSkillIds.value = []
  expandedMemoryIds.value = []
}

const loadCorrectionMemories = async () => {
  if (!ownerReady.value) {
    ElMessage.warning(t('common.invalid_request'))
    return
  }
  correctionLoading.value = true
  try {
    const res = await listPlatformMemories({
      owner_role: form.owner_role,
      owner_id: form.owner_id,
      page: 1,
      page_size: 50,
    })
    correctionRows.value = res.list || []
    correctionTotal.value = res.total || 0
  } catch (error) {
    console.error(error)
    ElMessage.error(t('frontend.operation_failed'))
  } finally {
    correctionLoading.value = false
  }
}

const openCorrectionDrawer = async () => {
  if (!ownerReady.value) {
    ElMessage.warning(t('common.invalid_request'))
    return
  }
  correctionDrawerVisible.value = true
  await loadCorrectionMemories()
}

const revokeCorrectionMemory = async (row: MemoryInfo) => {
  try {
    await ElMessageBox.confirm(`确认撤销 Memory #${row.id}？`, '撤销 Memory', {
      type: 'warning',
      confirmButtonText: '撤销',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  try {
    await revokePlatformMemory(row.id, {
      owner_role: form.owner_role,
      owner_id: form.owner_id,
      revoke_reason: 'platform_correction',
    })
    ElMessage.success(t('common.success'))
    await loadCorrectionMemories()
  } catch (error) {
    console.error(error)
    ElMessage.error(t('frontend.operation_failed'))
  }
}

const submitCorrectionMemory = async () => {
  const content = correctionForm.content.trim()
  if (!content) {
    ElMessage.warning(t('common.invalid_request'))
    return
  }
  if (!ownerReady.value) {
    ElMessage.warning(t('common.invalid_request'))
    return
  }
  correctionSaving.value = true
  try {
    await createPlatformMemory({
      owner_role: form.owner_role,
      owner_id: form.owner_id,
      scope_type: correctionForm.scope_type,
      scope_id: correctionForm.scope_id,
      memory_type: correctionForm.memory_type,
      content,
      source: 'platform_correction',
      confirm_high_pii: correctionForm.confirm_high_pii,
    })
    ElMessage.success(t('common.success'))
    correctionForm.content = ''
    correctionForm.confirm_high_pii = false
    await loadCorrectionMemories()
  } catch (error) {
    console.error(error)
    ElMessage.error(t('frontend.operation_failed'))
  } finally {
    correctionSaving.value = false
  }
}
</script>

<template>
  <div class="console-page console-page--fill semantic-debug-view">
    <PagePanel>
      <div class="workspace-surface" v-loading="loading">
      <div class="workspace-surface__toolbar">
        <div class="workspace-surface__filters">
          <span class="workspace-surface__context">用真实业务问题验证 Agent Skill 与 AI Memory 的召回效果</span>
        </div>
        <div class="workspace-surface__actions">
          <el-button :icon="EditPen" :disabled="!ownerReady" @click="openCorrectionDrawer">Memory 校正</el-button>
          <el-button :icon="Refresh" @click="reset">重置</el-button>
          <el-button type="primary" :icon="Search" :loading="loading" @click="runDebug">运行测试</el-button>
        </div>
      </div>
      <div class="workspace-surface__body">
        <section class="debug-config" aria-label="测试配置区">
          <div class="console-card debug-query-card">
            <div class="console-card__head">
              <div>
                <h3 class="console-card__title">测试 Query</h3>
                <p class="console-card__desc">输入一次真实 HR 问题，系统将按当前参数执行 Skill 与 Memory 召回。</p>
              </div>
              <span class="debug-card-hint">Ctrl + Enter 运行</span>
            </div>
            <div class="debug-card-body">
              <el-input
                v-model="form.query"
                type="textarea"
                :rows="5"
                maxlength="800"
                show-word-limit
                resize="none"
                placeholder="例如：这位候选人与高级前端工程师岗位是否匹配？请说明证据、风险和面试关注点。"
                @keyup.ctrl.enter="runDebug"
              />
              <div class="preset-chips">
                <span class="preset-chips__label">快捷 Query：</span>
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

          <div class="console-card debug-params-card">
            <div class="console-card__head">
              <div>
                <h3 class="console-card__title">召回参数</h3>
                <p class="console-card__desc">{{ selectedAgentLabel }}</p>
              </div>
            </div>
            <div class="debug-card-body">
              <el-form label-position="top">
                <el-form-item label="助手">
                  <el-select v-model="form.agent_type" placeholder="Agent 类型" filterable>
                    <el-option v-for="item in AGENT_TYPE_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
                  </el-select>
                </el-form-item>
                <div class="debug-params-grid">
                  <el-form-item label="Owner 角色">
                    <el-select v-model="form.owner_role" placeholder="Owner 角色">
                      <el-option v-for="item in OWNER_ROLE_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
                    </el-select>
                  </el-form-item>
                  <el-form-item label="Owner ID">
                    <el-input-number v-model="form.owner_id" :min="1" :controls="false" placeholder="必填" />
                  </el-form-item>
                </div>
                <p class="debug-owner-hint">当前调试 Owner：{{ selectedOwnerLabel }} #{{ form.owner_id || '—' }}</p>
                <div class="debug-params-grid">
                  <el-form-item label="岗位 ID">
                    <el-input-number v-model="form.job_id" :min="0" :controls="false" placeholder="可选" />
                  </el-form-item>
                  <el-form-item label="投递 ID">
                    <el-input-number v-model="form.application_id" :min="0" :controls="false" placeholder="可选" />
                  </el-form-item>
                </div>
                <el-form-item label="Top K">
                  <el-input-number v-model="form.limit" :min="1" :max="20" controls-position="right" />
                </el-form-item>
              </el-form>
            </div>
          </div>
        </section>

        <section class="console-stats debug-stats" aria-label="召回结果概览">
          <div class="console-stat">
            <div class="console-stat__label">测试状态</div>
            <div class="console-stat__value">{{ runStatusText }}</div>
          </div>
          <div class="console-stat">
            <div class="console-stat__label">Skill 命中</div>
            <div class="console-stat__value">{{ skillRows.length }}</div>
          </div>
          <div class="console-stat">
            <div class="console-stat__label">Memory 命中</div>
            <div class="console-stat__value">{{ memoryRows.length }}</div>
          </div>
          <div class="console-stat">
            <div class="console-stat__label">最高向量分数</div>
            <div class="console-stat__value">{{ formatScore(highestScore) }}</div>
          </div>
          <div class="console-stat">
            <div class="console-stat__label">请求耗时</div>
            <div class="console-stat__value">{{ formatDuration(requestDurationMs) }}</div>
          </div>
          <div class="console-stat">
            <div class="console-stat__label">召回模式</div>
            <div class="console-stat__value">
              <el-tag :type="embeddingStatusType" effect="plain">{{ embeddingStatusText }}</el-tag>
            </div>
          </div>
          <!-- TASK-FU-003：pool_confidence 顶部指标 -->
          <div v-if="hasResult" class="console-stat">
            <div class="console-stat__label">Skill 池置信度</div>
            <div class="console-stat__value">
              <el-tag
                v-if="skillPoolConfidenceMeta"
                :type="skillPoolConfidenceMeta.type"
                effect="plain"
              >
                {{ skillPoolConfidenceMeta.label }}（{{ skillPoolConfidence }}）
              </el-tag>
              <span v-else>-</span>
            </div>
          </div>
          <div v-if="hasResult" class="console-stat">
            <div class="console-stat__label">Memory 池置信度</div>
            <div class="console-stat__value">
              <el-tag
                v-if="memoryPoolConfidenceMeta"
                :type="memoryPoolConfidenceMeta.type"
                effect="plain"
              >
                {{ memoryPoolConfidenceMeta.label }}（{{ memoryPoolConfidence }}）
              </el-tag>
              <span v-else>-</span>
            </div>
          </div>
          <template v-if="hasResult">
            <div class="console-stat">
              <div class="console-stat__label">Embedding Provider</div>
              <div class="console-stat__value">{{ embeddingProviderText }}</div>
            </div>
            <div class="console-stat">
              <div class="console-stat__label">Embedding Model</div>
              <div class="console-stat__value">{{ embeddingModelText }}</div>
            </div>
            <div class="console-stat">
              <div class="console-stat__label">向量维度</div>
              <div class="console-stat__value">{{ embeddingDimText }}</div>
            </div>
            <div class="console-stat">
              <div class="console-stat__label">候选数量</div>
              <div class="console-stat__value">{{ candidateCountText }}</div>
            </div>
            <div class="console-stat">
              <div class="console-stat__label">Query 耗时</div>
              <div class="console-stat__value">{{ embeddingLatencyText }}</div>
            </div>
          </template>
        </section>

        <el-alert
          v-if="fallbackMessage"
          class="workspace-surface__error"
          type="warning"
          show-icon
          :closable="false"
          :title="fallbackMessage"
        />

        <div class="debug-results" aria-label="召回结果对比">
          <div class="console-card debug-result-card">
            <div class="console-card__head">
              <div>
                <h3 class="console-card__title">Agent Skill 召回</h3>
                <p class="console-card__desc">展示匹配到的数据库版 SKILL.md、分类、分数和触发原因。</p>
              </div>
              <el-tag size="small" type="info">{{ skillRows.length }} 条</el-tag>
            </div>
            <div class="debug-card-body">
              <div v-if="skillRows.length" class="debug-result-list">
                <article v-for="item in skillRows" :key="item.version_id" class="debug-result-item">
                  <div class="debug-result-item__top">
                    <div>
                      <h4>{{ skillTitle(item) }}</h4>
                      <p>Skill #{{ item.skill_id }} · Version #{{ item.version_id }} · {{ item.version }}</p>
                    </div>
                    <div class="debug-score-pill-group">
                      <span
                        v-if="relevanceModeMeta(item.relevance_mode)"
                        class="debug-mode-tag"
                        :class="`debug-mode-tag--${relevanceModeMeta(item.relevance_mode)?.type}`"
                      >
                        {{ relevanceModeMeta(item.relevance_mode)?.label }}
                      </span>
                      <span class="debug-score-pill debug-score-pill--vector">向量 {{ formatScore(embeddingScore(item)) }}</span>
                      <span class="debug-score-pill">排序 {{ formatScore(finalRankScore(item)) }}</span>
                    </div>
                  </div>
                  <div class="debug-result-item__meta">
                    <span>{{ item.category || 'general' }}</span>
                    <span v-if="item.scenario">{{ item.scenario }}</span>
                    <span>P{{ item.priority ?? 0 }}</span>
                    <span v-if="item.pool_rank != null">排名 {{ item.pool_rank }}</span>
                  </div>
                  <p class="debug-result-item__summary">{{ item.reason || 'metadata and content match' }}</p>
                  <div v-if="item.semantic_tags?.length" class="debug-tag-row">
                    <el-tag v-for="tag in item.semantic_tags" :key="tag" size="small" effect="plain">{{ tag }}</el-tag>
                  </div>
                  <button type="button" class="debug-detail-toggle" @click="toggleSkill(item.version_id)">
                    {{ isSkillExpanded(item.version_id) ? '收起详情' : '查看详情' }}
                  </button>
                  <div v-if="isSkillExpanded(item.version_id)" class="debug-detail-block">
                    <div><span>Skill ID</span><strong>{{ item.skill_id }}</strong></div>
                    <div><span>Version ID</span><strong>{{ item.version_id }}</strong></div>
                    <div><span>Compiled Hash</span><strong>{{ item.compiled_hash || '-' }}</strong></div>
                    <div><span>Embedding 分数</span><strong>{{ formatScore(embeddingScore(item)) }}</strong></div>
                    <div><span>最终排序分</span><strong>{{ formatScore(finalRankScore(item)) }}</strong></div>
                    <div><span>召回原因</span><strong>{{ item.reason || '-' }}</strong></div>
                  </div>
                  <!-- TASK-FU-003：breakdown 展开区 -->
                  <div v-if="isSkillExpanded(item.version_id)" class="debug-breakdown">
                    <h5 class="debug-breakdown__title">混合打分 breakdown</h5>
                    <div class="debug-breakdown__grid">
                      <div>
                        <el-tooltip :content="BREAKDOWN_TOOLTIPS['vector']" placement="top">
                          <span>vector</span>
                        </el-tooltip>
                        <strong>{{ formatBreakdownScore(item.vector_score) }}</strong>
                      </div>
                      <div>
                        <el-tooltip :content="BREAKDOWN_TOOLTIPS['lexical']" placement="top">
                          <span>lexical</span>
                        </el-tooltip>
                        <strong>{{ formatBreakdownScore(item.lexical_score) }}</strong>
                      </div>
                      <div>
                        <el-tooltip :content="BREAKDOWN_TOOLTIPS['metadata']" placement="top">
                          <span>metadata</span>
                        </el-tooltip>
                        <strong>{{ formatBreakdownScore(item.metadata_score) }}</strong>
                      </div>
                      <div>
                        <el-tooltip :content="BREAKDOWN_TOOLTIPS['relevance']" placement="top">
                          <span>relevance</span>
                        </el-tooltip>
                        <strong>{{ formatBreakdownScore(item.relevance_score) }}</strong>
                      </div>
                      <div>
                        <el-tooltip :content="BREAKDOWN_TOOLTIPS['business boost']" placement="top">
                          <span>business boost</span>
                        </el-tooltip>
                        <strong>{{ formatBreakdownScore(item.business_boost) }}</strong>
                      </div>
                      <div>
                        <el-tooltip :content="BREAKDOWN_TOOLTIPS['final rank']" placement="top">
                          <span>final rank</span>
                        </el-tooltip>
                        <strong>{{ formatBreakdownScore(item.final_rank_score) }}</strong>
                      </div>
                    </div>
                  </div>
                </article>
              </div>
              <div v-else class="debug-empty">
                <h4>{{ hasResult ? '未命中 Agent Skill' : '等待运行测试' }}</h4>
                <p>{{ hasResult ? '本次查询没有匹配到可用 Skill。可以检查 Skill 是否启用、语义标签是否覆盖该场景，或调整 Query 表达。' : '运行后这里会展示命中的 Agent Skill、Embedding 分数、最终排序分、分类、触发原因和语义标签。' }}</p>
              </div>
            </div>
          </div>

          <div class="console-card debug-result-card">
            <div class="console-card__head">
              <div>
                <h3 class="console-card__title">AI Memory 召回</h3>
                <p class="console-card__desc">展示按 HR、岗位、投递等 scope 召回的长期记忆和排序依据。</p>
              </div>
              <el-tag size="small" type="info">{{ memoryRows.length }} 条</el-tag>
            </div>
            <div class="debug-card-body">
              <div v-if="memoryRows.length" class="debug-result-list">
                <article v-for="item in memoryRows" :key="item.id" class="debug-result-item">
                  <div class="debug-result-item__top">
                    <div>
                      <h4>{{ memoryScopeText(item) }}</h4>
                      <p>{{ item.memory_type || '-' }} · {{ item.source || '-' }}</p>
                    </div>
                    <div class="debug-score-pill-group">
                      <span
                        v-if="relevanceModeMeta(item.relevance_mode)"
                        class="debug-mode-tag"
                        :class="`debug-mode-tag--${relevanceModeMeta(item.relevance_mode)?.type}`"
                      >
                        {{ relevanceModeMeta(item.relevance_mode)?.label }}
                      </span>
                      <span class="debug-score-pill debug-score-pill--vector">向量 {{ formatScore(embeddingScore(item)) }}</span>
                      <span class="debug-score-pill">排序 {{ formatScore(finalRankScore(item)) }}</span>
                    </div>
                  </div>
                  <div class="debug-result-item__meta">
                    <span>重要度 {{ formatScore(item.importance) }}</span>
                    <span>置信度 {{ formatScore(item.confidence) }}</span>
                    <span v-if="item.pool_rank != null">排名 {{ item.pool_rank }}</span>
                    <span v-if="item.created_at">{{ item.created_at }}</span>
                  </div>
                  <p class="debug-result-item__summary">{{ item.content }}</p>
                  <button type="button" class="debug-detail-toggle" @click="toggleMemory(item.id)">
                    {{ isMemoryExpanded(item.id) ? '收起详情' : '查看详情' }}
                  </button>
                  <div v-if="isMemoryExpanded(item.id)" class="debug-detail-block">
                    <div><span>Memory ID</span><strong>{{ item.id }}</strong></div>
                    <div><span>Scope</span><strong>{{ item.scope_type }} #{{ item.scope_id }}</strong></div>
                    <div><span>Embedding 分数</span><strong>{{ formatScore(embeddingScore(item)) }}</strong></div>
                    <div><span>最终排序分</span><strong>{{ formatScore(finalRankScore(item)) }}</strong></div>
                    <div><span>召回原因</span><strong>{{ item.reason || '-' }}</strong></div>
                  </div>
                  <!-- TASK-FU-003：breakdown 展开区 -->
                  <div v-if="isMemoryExpanded(item.id)" class="debug-breakdown">
                    <h5 class="debug-breakdown__title">混合打分 breakdown</h5>
                    <div class="debug-breakdown__grid">
                      <div>
                        <el-tooltip :content="BREAKDOWN_TOOLTIPS['vector']" placement="top">
                          <span>vector</span>
                        </el-tooltip>
                        <strong>{{ formatBreakdownScore(item.vector_score) }}</strong>
                      </div>
                      <div>
                        <el-tooltip :content="BREAKDOWN_TOOLTIPS['lexical']" placement="top">
                          <span>lexical</span>
                        </el-tooltip>
                        <strong>{{ formatBreakdownScore(item.lexical_score) }}</strong>
                      </div>
                      <div>
                        <el-tooltip :content="BREAKDOWN_TOOLTIPS['metadata']" placement="top">
                          <span>metadata</span>
                        </el-tooltip>
                        <strong>{{ formatBreakdownScore(item.metadata_score) }}</strong>
                      </div>
                      <div>
                        <el-tooltip :content="BREAKDOWN_TOOLTIPS['relevance']" placement="top">
                          <span>relevance</span>
                        </el-tooltip>
                        <strong>{{ formatBreakdownScore(item.relevance_score) }}</strong>
                      </div>
                      <div>
                        <el-tooltip :content="BREAKDOWN_TOOLTIPS['business boost']" placement="top">
                          <span>business boost</span>
                        </el-tooltip>
                        <strong>{{ formatBreakdownScore(item.business_boost) }}</strong>
                      </div>
                      <div>
                        <el-tooltip :content="BREAKDOWN_TOOLTIPS['final rank']" placement="top">
                          <span>final rank</span>
                        </el-tooltip>
                        <strong>{{ formatBreakdownScore(item.final_rank_score) }}</strong>
                      </div>
                    </div>
                  </div>
                </article>
              </div>
              <div v-else class="debug-empty">
                <h4>{{ hasResult ? '未命中 AI Memory' : '等待运行测试' }}</h4>
                <p>{{ hasResult ? '本次查询没有召回长期记忆。可以补充岗位 ID 或投递 ID，确认对应 scope 下是否存在可召回内容。' : '运行后这里会展示命中的 Memory、scope、来源、Embedding 分数、最终排序分、重要度和召回原因。' }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    </PagePanel>

    <el-drawer v-model="correctionDrawerVisible" title="Memory 校正" size="480px" append-to-body>
      <div class="memory-correction" v-loading="correctionLoading">
        <p class="memory-correction__hint">
          Owner：{{ selectedOwnerLabel }} #{{ form.owner_id || '—' }} · 共 {{ correctionTotal }} 条
        </p>
        <div class="memory-correction__list">
          <article v-for="row in correctionRows" :key="row.id" class="memory-correction__item">
            <div class="memory-correction__item-head">
              <strong>#{{ row.id }}</strong>
              <el-tag size="small" effect="plain">{{ row.status || 'active' }}</el-tag>
            </div>
            <p class="memory-correction__item-meta">
              {{ row.scope_type }} #{{ row.scope_id || 0 }} · {{ row.memory_type || '-' }} · {{ row.source || '-' }}
            </p>
            <p class="memory-correction__item-content">{{ row.content }}</p>
            <el-button
              v-if="row.status !== 'revoked'"
              size="small"
              type="danger"
              plain
              @click="revokeCorrectionMemory(row)"
            >
              撤销
            </el-button>
          </article>
          <div v-if="!correctionRows.length" class="memory-correction__empty">暂无 Memory，可在下方创建校正条目。</div>
        </div>
        <div class="memory-correction__create">
          <h4>创建校正 Memory</h4>
          <el-form label-position="top">
            <el-form-item label="内容">
              <el-input v-model="correctionForm.content" type="textarea" :rows="3" maxlength="500" show-word-limit />
            </el-form-item>
            <div class="debug-params-grid">
              <el-form-item label="Scope 类型">
                <el-select v-model="correctionForm.scope_type">
                  <el-option v-for="item in MEMORY_SCOPE_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
              <el-form-item label="Scope ID">
                <el-input-number v-model="correctionForm.scope_id" :min="0" :controls="false" placeholder="可选" />
              </el-form-item>
            </div>
            <el-form-item label="Memory 类型">
              <el-input v-model="correctionForm.memory_type" placeholder="例如 preference / fact" />
            </el-form-item>
            <el-form-item>
              <el-checkbox v-model="correctionForm.confirm_high_pii">
                确认写入高敏感内容（手机号 / 邮箱 / 证件号等）
              </el-checkbox>
            </el-form-item>
            <el-button type="primary" :loading="correctionSaving" @click="submitCorrectionMemory">创建</el-button>
          </el-form>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<style scoped>
.console-page--fill .page-panel {
  flex: 1;
  min-height: 0;
}

.debug-config {
  display: grid;
  grid-template-columns: minmax(0, 1.5fr) minmax(280px, 1fr);
  gap: 16px;
  padding: var(--page-inset);
}

.debug-card-hint {
  color: var(--text-muted);
  font-size: 12px;
  white-space: nowrap;
  align-self: center;
}

.debug-card-body {
  padding: 16px 18px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.debug-card-body :deep(.el-form-item) {
  margin-bottom: 0;
}

.debug-params-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.debug-params-card :deep(.el-input-number),
.debug-params-card :deep(.el-select) {
  width: 100%;
}

.debug-owner-hint {
  margin: 0;
  color: var(--text-muted);
  font-size: 12px;
}

.memory-correction {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 100%;
}

.memory-correction__hint {
  margin: 0;
  color: var(--text-muted);
  font-size: 12px;
}

.memory-correction__list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 320px;
  overflow: auto;
}

.memory-correction__item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 12px;
  border: 1px solid var(--surface-soft-border);
  border-radius: var(--surface-soft-radius, 12px);
  background: var(--surface-muted);
}

.memory-correction__item-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.memory-correction__item-meta {
  margin: 0;
  color: var(--text-muted);
  font-size: 12px;
}

.memory-correction__item-content {
  margin: 0;
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.memory-correction__empty {
  padding: 16px;
  border: 1px dashed var(--surface-soft-border);
  border-radius: var(--surface-soft-radius, 12px);
  color: var(--text-muted);
  font-size: 12px;
  text-align: center;
}

.memory-correction__create {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--surface-soft-border);
}

.memory-correction__create h4 {
  margin: 0;
  color: var(--text-primary);
  font-size: 14px;
}

.preset-chips {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.preset-chips__label {
  color: var(--text-muted);
  font-size: 12px;
  flex: 0 0 auto;
}

.preset-chip {
  max-width: 100%;
  padding: 4px 10px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--surface);
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 12px;
  line-height: 1.4;
  text-align: left;
  transition: border-color 0.15s, color 0.15s, background-color 0.15s;
}

.preset-chip:hover {
  border-color: var(--el-color-primary-light-5);
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.debug-stats {
  padding: 0 var(--page-inset) var(--page-inset);
}

.debug-stats .console-stat__value {
  font-size: 18px;
}

.debug-results {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  padding: 0 var(--page-inset) var(--page-inset);
}

.debug-result-card {
  display: flex;
  flex-direction: column;
}

.debug-result-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.debug-result-item {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--surface-soft-border);
  border-radius: var(--surface-soft-radius, 12px);
  background: var(--surface-muted);
}

.debug-result-item__top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.debug-result-item__top h4 {
  margin: 0;
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 700;
  line-height: 1.4;
}

.debug-result-item__top p {
  margin: 3px 0 0;
  color: var(--text-muted);
  font-size: 12px;
}

.debug-score-pill {
  flex: 0 0 auto;
  padding: 3px 8px;
  border-radius: 999px;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  font-weight: 700;
}

.debug-score-pill--vector {
  background: var(--el-color-success-light-9);
  color: var(--el-color-success);
}

.debug-result-item__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.debug-result-item__meta span {
  padding: 2px 7px;
  border-radius: 999px;
  background: var(--surface);
  color: var(--text-muted);
  font-size: 12px;
}

.debug-result-item__summary {
  margin: 0;
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.6;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
  overflow: hidden;
}

.debug-tag-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.debug-detail-toggle {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--el-color-primary);
  cursor: pointer;
  font-size: 12px;
  align-self: flex-start;
}

.debug-detail-block {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 8px;
  padding: 10px;
  border-radius: var(--surface-soft-radius, 12px);
  background: var(--surface-soft-bg);
}

.debug-detail-block > div {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.debug-detail-block span {
  color: var(--text-muted);
  font-size: 12px;
}

.debug-detail-block strong {
  color: var(--text-primary);
  font-size: 12px;
  font-weight: 600;
}

/* TASK-FU-003：breakdown / mode tag 样式 */
.debug-score-pill-group {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.debug-mode-tag {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  border: 1px solid transparent;
  letter-spacing: 0.02em;
}

.debug-mode-tag--success {
  background: var(--el-color-success-light-9);
  color: var(--el-color-success);
  border-color: var(--el-color-success-light-7);
}

.debug-mode-tag--warning {
  background: var(--el-color-warning-light-9);
  color: var(--el-color-warning);
  border-color: var(--el-color-warning-light-7);
}

.debug-mode-tag--info {
  background: var(--el-color-info-light-9);
  color: var(--el-color-info);
  border-color: var(--el-color-info-light-7);
}

.debug-breakdown {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 12px;
  border: 1px dashed var(--surface-soft-border);
  border-radius: var(--surface-soft-radius, 12px);
  background: var(--surface-muted);
}

.debug-breakdown__title {
  margin: 0;
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.debug-breakdown__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(110px, 1fr));
  gap: 6px 12px;
}

.debug-breakdown__grid > div {
  display: flex;
  flex-direction: column;
  gap: 1px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

.debug-breakdown__grid span {
  color: var(--text-muted);
  font-size: 11px;
  letter-spacing: 0.02em;
}

.debug-breakdown__grid strong {
  color: var(--text-primary);
  font-size: 12px;
  font-weight: 700;
}

.debug-empty {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 24px;
  border: 1px dashed var(--surface-soft-border);
  border-radius: var(--surface-soft-radius, 12px);
  background: var(--surface-soft-bg);
}

.debug-empty h4 {
  margin: 0;
  color: var(--text-primary);
  font-size: 14px;
}

.debug-empty p {
  margin: 0;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.6;
}

@media (max-width: 1366px) {
  .debug-config {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 1080px) {
  .debug-results {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .debug-params-grid {
    grid-template-columns: 1fr;
  }
}
</style>
