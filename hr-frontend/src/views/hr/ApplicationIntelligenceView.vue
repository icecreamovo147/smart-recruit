<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Cpu, DocumentChecked, Refresh, TrendCharts } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import { PERM } from '@/types/domain'
import { BusinessError } from '@/types/api'
import {
  evaluateCandidateMatch,
  getApplicationResumeProfile,
  getCandidateMatchEvaluation,
  parseResumeProfile,
} from '@/api/recruitingIntelligence'
import type {
  CandidateMatchEvaluationInfo,
  CandidateMatchEvidenceInfo,
  CandidateMatchEvaluationSnapshotInfo,
  ResumeExperienceInfo,
  ResumeProfileSnapshotInfo,
} from '@/types/recruitingIntelligence'

type JsonRecord = Record<string, unknown>
type JsonListItem = string | number | JsonRecord
type DimensionScore = {
  key: string
  score: number
}

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const applicationId = computed(() => Number(route.params.applicationId || 0))
const profileLoading = ref(false)
const evaluationLoading = ref(false)
const actionLoading = ref<'parse' | 'evaluate' | ''>('')
const profileMissing = ref(false)
const evaluationMissing = ref(false)
const profile = ref<ResumeProfileSnapshotInfo | null>(null)
const evaluationSnapshot = ref<CandidateMatchEvaluationSnapshotInfo | null>(null)

const canRunAIAction = computed(() => auth.hasPermission(PERM.AI_HR_USE))
const candidateName = computed(() => String(route.query.candidate_name || profile.value?.profile?.full_name || '候选人'))
const jobTitle = computed(() => String(route.query.job_title || '当前投递岗位'))
const jobId = computed(() => Number(route.query.job_id || evaluation.value?.job_id || 0))
const evaluation = computed<CandidateMatchEvaluationInfo | null>(() => evaluationSnapshot.value?.evaluation || null)
const evidence = computed<CandidateMatchEvidenceInfo[]>(() => evaluationSnapshot.value?.evidence || [])

const formatDateTime = (value?: string): string => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (num: number): string => String(num).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

const formatRange = (start?: string, end?: string, isCurrent?: number): string => {
  const left = start || '未知'
  const right = isCurrent ? '至今' : (end || '未知')
  return `${left} - ${right}`
}

const parseJson = <T>(value: string | undefined, fallback: T): T => {
  if (!value) return fallback
  try {
    return JSON.parse(value) as T
  } catch {
    return fallback
  }
}

const asText = (value: unknown): string => {
  if (value === null || value === undefined) return ''
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') return String(value)
  if (typeof value === 'object') {
    const record = value as JsonRecord
    return String(record.message || record.name || record.title || record.requirement || record.code || JSON.stringify(record))
  }
  return ''
}

const jsonList = (value?: string): JsonListItem[] => {
  const parsed = parseJson<unknown>(value, [])
  if (Array.isArray(parsed)) return parsed as JsonListItem[]
  if (parsed && typeof parsed === 'object') return Object.entries(parsed).map(([key, item]) => ({ key, value: item }))
  return []
}

const jsonTags = (value?: string): string[] => jsonList(value).map(asText).filter(Boolean)

const scoreBreakdown = computed(() => parseJson<JsonRecord>(evaluation.value?.score_breakdown_json, {}))

const toDimensionScore = (item: unknown, fallbackKey = ''): DimensionScore | null => {
  if (typeof item === 'number') return { key: fallbackKey, score: item }
  if (typeof item === 'string') {
    const score = Number(item)
    return Number.isFinite(score) ? { key: fallbackKey, score } : null
  }
  if (!item || typeof item !== 'object') return null

  const record = item as JsonRecord
  const key = asText(record.name || record.key || record.dimension || fallbackKey)
  const scoreValue = record.score ?? record.value ?? record.percentage
  const score = Number(scoreValue)
  return key || Number.isFinite(score) ? { key, score: Number.isFinite(score) ? score : 0 } : null
}

const dimensionScoresFromParsed = (value: unknown): DimensionScore[] => {
  if (Array.isArray(value)) {
    return value.map((item) => toDimensionScore(item)).filter((item): item is DimensionScore => Boolean(item))
  }
  if (value && typeof value === 'object') {
    const record = value as JsonRecord
    const nestedDimensions = record.dimensions
    if (Array.isArray(nestedDimensions)) return dimensionScoresFromParsed(nestedDimensions)
    return Object.entries(record)
      .map(([key, item]) => toDimensionScore(item, key))
      .filter((item): item is DimensionScore => Boolean(item))
  }
  return []
}

const dimensions = computed(() => {
  const merged = new Map<string, DimensionScore>()
  for (const item of dimensionScoresFromParsed(scoreBreakdown.value)) {
    if (item.key) merged.set(item.key, item)
  }
  for (const item of dimensionScoresFromParsed(parseJson<unknown>(evaluation.value?.dimensions_json, []))) {
    if (item.key) merged.set(item.key, item)
  }
  return Array.from(merged.values()).filter((item) => item.score || item.key)
})

const strengths = computed(() => jsonList(evaluation.value?.strengths_json))
const risks = computed(() => jsonList(evaluation.value?.risks_json))
const missingRequirements = computed(() => {
  const fromField = jsonTags(evaluation.value?.missing_requirements_json)
  if (fromField.length) return fromField

  const fromBreakdown = scoreBreakdown.value.missing_requirements
  if (!Array.isArray(fromBreakdown)) return []
  return fromBreakdown.map(asText).filter(Boolean)
})

const recommendationLabel = computed(() => {
  const value = evaluation.value?.recommendation || ''
  const labels: Record<string, string> = {
    strong_match: '强匹配',
    possible_match: '可推进',
    weak_match: '弱匹配',
    needs_review: '需要复核',
    not_recommended: '不建议推进',
    review: '需要复核',
    strong_recommend: '强烈推荐',
    recommend: '推荐推进',
    neutral: '谨慎评估',
    not_recommend: '不建议推进',
    strong_not_recommend: '强烈不建议',
  }
  return labels[value] || value || '-'
})

const recommendationType = computed(() => {
  const value = evaluation.value?.recommendation || ''
  const types: Record<string, string> = {
    strong_match: 'success',
    possible_match: 'primary',
    weak_match: 'warning',
    needs_review: 'info',
    not_recommended: 'danger',
    review: 'info',
    strong_recommend: 'success',
    recommend: 'success',
    neutral: 'warning',
    not_recommend: 'danger',
    strong_not_recommend: 'danger',
  }
  if (types[value]) return types[value]
  if (value.includes('recommend') && !value.includes('not')) return 'success'
  if (value.includes('not')) return 'danger'
  return 'warning'
})

const scoreStatus = computed(() => {
  const score = Number(evaluation.value?.overall_score || 0)
  if (score >= 80) return 'success'
  if (score >= 60) return 'warning'
  return 'exception'
})

const isNotFound = (error: unknown): boolean => error instanceof BusinessError && error.code === 404

const loadProfile = async () => {
  if (!applicationId.value) return
  profileLoading.value = true
  profileMissing.value = false
  try {
    const data = await getApplicationResumeProfile(applicationId.value)
    profile.value = data.profile?.profile ? data.profile : null
    profileMissing.value = !profile.value
  } catch (error: unknown) {
    if (isNotFound(error)) {
      profile.value = null
      profileMissing.value = true
      return
    }
    throw error
  } finally {
    profileLoading.value = false
  }
}

const loadEvaluation = async () => {
  if (!applicationId.value) return
  evaluationLoading.value = true
  evaluationMissing.value = false
  try {
    const data = await getCandidateMatchEvaluation(applicationId.value)
    evaluationSnapshot.value = data.evaluation?.evaluation ? data.evaluation : null
    evaluationMissing.value = !evaluationSnapshot.value
  } catch (error: unknown) {
    if (isNotFound(error)) {
      evaluationSnapshot.value = null
      evaluationMissing.value = true
      return
    }
    throw error
  } finally {
    evaluationLoading.value = false
  }
}

const loadAll = async () => {
  try {
    await Promise.all([loadProfile(), loadEvaluation()])
  } catch (error: any) {
    ElMessage.error(error?.message || '智能评估信息加载失败')
  }
}

const runParse = async () => {
  if (!canRunAIAction.value) return
  actionLoading.value = 'parse'
  try {
    const data = await parseResumeProfile({ application_id: applicationId.value })
    profile.value = data.profile?.profile ? data.profile : null
    profileMissing.value = !profile.value
    ElMessage.success('简历画像解析已完成')
  } catch {
    await loadProfile().catch(() => undefined)
  } finally {
    actionLoading.value = ''
  }
}

const runEvaluation = async () => {
  if (!canRunAIAction.value) return
  actionLoading.value = 'evaluate'
  try {
    const data = await evaluateCandidateMatch(applicationId.value)
    evaluationSnapshot.value = data.evaluation?.evaluation ? data.evaluation : null
    evaluationMissing.value = !evaluationSnapshot.value
    ElMessage.success('匹配评估已完成')
    if (!profile.value) await loadProfile().catch(() => undefined)
  } catch {
    await loadEvaluation().catch(() => undefined)
  } finally {
    actionLoading.value = ''
  }
}

const goBack = () => {
  if (jobId.value) {
    router.push(`/hr/jobs/${jobId.value}/applications`)
    return
  }
  router.back()
}

const experienceAchievements = (item: ResumeExperienceInfo): string[] => jsonTags(item.achievements_json)

onMounted(loadAll)
</script>

<template>
  <section class="intelligence-page">
    <header class="page-header intelligence-header">
      <div>
        <p class="console-eyebrow">RECRUITING INTELLIGENCE</p>
        <h2>简历画像与匹配评估</h2>
        <p>{{ candidateName }} / {{ jobTitle }} / 投递 #{{ applicationId }}</p>
      </div>
      <div class="header-actions">
        <el-button :icon="ArrowLeft" @click="goBack">返回</el-button>
        <el-button :icon="Refresh" @click="loadAll">刷新</el-button>
        <el-tooltip :disabled="canRunAIAction" content="需要 AI HR 使用权限">
          <span>
            <el-button :icon="DocumentChecked" :loading="actionLoading === 'parse'" :disabled="!canRunAIAction" @click="runParse">
              {{ profile ? '重新解析画像' : '解析画像' }}
            </el-button>
          </span>
        </el-tooltip>
        <el-tooltip :disabled="canRunAIAction" content="需要 AI HR 使用权限">
          <span>
            <el-button type="primary" :icon="Cpu" :loading="actionLoading === 'evaluate'" :disabled="!canRunAIAction" @click="runEvaluation">
              {{ evaluation ? '重新评估匹配' : '生成匹配评估' }}
            </el-button>
          </span>
        </el-tooltip>
      </div>
    </header>

    <div class="intelligence-grid">
      <section class="intelligence-column" v-loading="profileLoading">
        <el-card shadow="never" class="panel-card">
          <template #header>
            <div class="panel-title">
              <span>结构化简历画像</span>
              <div class="panel-tags">
                <el-tag v-if="profile?.profile" size="small" type="primary">v{{ profile.profile.version }}</el-tag>
                <el-tag v-if="profile?.profile?.is_current" size="small" type="success">当前版本</el-tag>
                <el-tag v-if="profile?.parse_run?.status" size="small" type="info">{{ profile.parse_run.status }}</el-tag>
              </div>
            </div>
          </template>

          <template v-if="profile?.profile">
            <div class="profile-summary">
              <h3>{{ profile.profile.full_name || candidateName }}</h3>
              <p>{{ profile.profile.headline || profile.profile.summary || '暂无画像摘要' }}</p>
              <div class="metric-grid">
                <div><span>经验年限</span><strong>{{ profile.profile.total_experience_years || 0 }} 年</strong></div>
                <div><span>最高学历</span><strong>{{ profile.profile.highest_degree || '-' }}</strong></div>
                <div><span>所在地</span><strong>{{ profile.profile.location || '-' }}</strong></div>
                <div><span>更新时间</span><strong>{{ formatDateTime(profile.profile.updated_at) }}</strong></div>
              </div>
            </div>

            <div class="section-block">
              <div class="section-label">技能证据</div>
              <div class="skill-list">
                <div v-for="skill in profile.skills" :key="skill.id" class="skill-item">
                  <div>
                    <strong>{{ skill.name }}</strong>
                    <span>{{ [skill.category, skill.level, skill.years ? `${skill.years}年` : ''].filter(Boolean).join(' / ') || '未标注' }}</span>
                  </div>
                  <p v-if="skill.evidence">{{ skill.evidence }}</p>
                </div>
                <div v-if="profile.skills.length === 0" class="empty-inline">暂无技能画像</div>
              </div>
            </div>

            <div class="section-block">
              <div class="section-label">工作经历</div>
              <div class="timeline-list">
                <div v-for="item in profile.experiences" :key="item.id" class="timeline-item">
                  <div class="timeline-head">
                    <strong>{{ item.company || '-' }}</strong>
                    <span>{{ formatRange(item.start_date, item.end_date, item.is_current) }}</span>
                  </div>
                  <div class="timeline-role">{{ item.title || '-' }}<span v-if="item.location"> / {{ item.location }}</span></div>
                  <p v-if="item.description">{{ item.description }}</p>
                  <ul v-if="experienceAchievements(item).length">
                    <li v-for="achievement in experienceAchievements(item)" :key="achievement">{{ achievement }}</li>
                  </ul>
                </div>
                <div v-if="profile.experiences.length === 0" class="empty-inline">暂无工作经历</div>
              </div>
            </div>

            <div class="two-column-block">
              <div class="section-block">
                <div class="section-label">教育经历</div>
                <div v-for="item in profile.educations" :key="item.id" class="compact-item">
                  <strong>{{ item.school || '-' }}</strong>
                  <span>{{ [item.degree, item.major].filter(Boolean).join(' / ') || '-' }}</span>
                  <em>{{ formatRange(item.start_date, item.end_date) }}</em>
                </div>
                <div v-if="profile.educations.length === 0" class="empty-inline">暂无教育经历</div>
              </div>
              <div class="section-block">
                <div class="section-label">项目经历</div>
                <div v-for="item in profile.projects" :key="item.id" class="compact-item">
                  <strong>{{ item.name || '-' }}</strong>
                  <span>{{ item.role || '-' }}</span>
                  <em>{{ jsonTags(item.technologies_json).join(' / ') || formatRange(item.start_date, item.end_date) }}</em>
                </div>
                <div v-if="profile.projects.length === 0" class="empty-inline">暂无项目经历</div>
              </div>
            </div>
          </template>
          <el-empty v-else :description="profileMissing ? '暂无结构化简历画像，可发起解析' : '画像数据未返回'" />
        </el-card>
      </section>

      <section class="intelligence-column" v-loading="evaluationLoading">
        <el-card shadow="never" class="panel-card">
          <template #header>
            <div class="panel-title">
              <span>岗位匹配评估</span>
              <div class="panel-tags">
                <el-tag v-if="evaluation" size="small" type="primary">v{{ evaluation.evaluation_version }}</el-tag>
                <el-tag v-if="evaluation?.is_latest" size="small" type="success">最新</el-tag>
                <el-tag v-if="evaluation?.model_name" size="small" type="info">{{ evaluation.model_name }}</el-tag>
              </div>
            </div>
          </template>

          <template v-if="evaluation">
            <div class="score-panel">
              <el-progress type="dashboard" :percentage="Math.round(evaluation.overall_score || 0)" :status="scoreStatus as any" />
              <div class="score-copy">
                <el-tag :type="recommendationType as any" size="large">{{ recommendationLabel }}</el-tag>
                <p>{{ evaluation.summary || '暂无评估摘要' }}</p>
                <span>评估时间：{{ formatDateTime(evaluation.evaluated_at || evaluation.updated_at) }}</span>
              </div>
            </div>

            <div v-if="dimensions.length" class="section-block">
              <div class="section-label">评分拆解</div>
              <div class="dimension-list">
                <div v-for="item in dimensions" :key="item.key" class="dimension-row">
                  <span>{{ item.key }}</span>
                  <el-progress :percentage="Math.round(item.score)" :show-text="false" />
                  <strong>{{ Math.round(item.score) }}</strong>
                </div>
              </div>
            </div>

            <div class="signal-grid">
              <div class="signal-block">
                <div class="section-label">优势</div>
                <div v-for="item in strengths" :key="asText(item)" class="signal-item signal-item--strength">{{ asText(item) }}</div>
                <div v-if="strengths.length === 0" class="empty-inline">暂无优势信号</div>
              </div>
              <div class="signal-block">
                <div class="section-label">风险</div>
                <div v-for="item in risks" :key="asText(item)" class="signal-item signal-item--risk">{{ asText(item) }}</div>
                <div v-if="risks.length === 0" class="empty-inline">暂无风险信号</div>
              </div>
            </div>

            <div class="section-block">
              <div class="section-label">缺失要求</div>
              <div class="tag-cloud">
                <el-tag v-for="item in missingRequirements" :key="item" type="warning" effect="plain">{{ item }}</el-tag>
                <span v-if="missingRequirements.length === 0" class="empty-inline">暂无缺失要求</span>
              </div>
            </div>

            <div class="section-block">
              <div class="section-label">证据明细</div>
              <div class="evidence-list">
                <div v-for="item in evidence" :key="item.id" class="evidence-item">
                  <div class="evidence-head">
                    <el-tag size="small" :type="item.evidence_type === 'risk' ? 'danger' : 'success'">{{ item.evidence_type || 'evidence' }}</el-tag>
                    <span>{{ item.dimension || '-' }} / {{ item.source_table || '-' }}</span>
                    <strong>{{ item.score_impact > 0 ? '+' : '' }}{{ item.score_impact || 0 }}</strong>
                  </div>
                  <p>{{ item.snippet || '暂无证据片段' }}</p>
                </div>
                <div v-if="evidence.length === 0" class="empty-inline">暂无证据明细</div>
              </div>
            </div>
          </template>
          <el-empty v-else :description="evaluationMissing ? '暂无匹配评估，可发起生成' : '评估数据未返回'" />
        </el-card>
      </section>
    </div>
  </section>
</template>

<style scoped>
.intelligence-page {
  width: 100%;
  max-width: 1440px;
  margin: 0 auto;
  padding: 18px 20px 28px;
  box-sizing: border-box;
}

.intelligence-header {
  align-items: flex-start;
  margin-bottom: 16px;
}

.intelligence-header h2 {
  margin: 4px 0 6px;
}

.intelligence-header p {
  margin: 0;
  color: #64748b;
}

.console-eyebrow {
  margin: 0;
  color: #2563eb;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0;
}

.header-actions,
.panel-title,
.panel-tags,
.tag-cloud,
.evidence-head {
  display: flex;
  align-items: center;
}

.header-actions {
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.intelligence-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.04fr) minmax(0, 0.96fr);
  gap: 16px;
  align-items: start;
}

.panel-card {
  border-radius: 8px;
  border-color: #e2e8f0;
}

.panel-card :deep(.el-card__header) {
  padding: 13px 16px;
  background: #f8fafc;
}

.panel-title {
  justify-content: space-between;
  gap: 12px;
  font-weight: 700;
}

.panel-tags {
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px;
}

.profile-summary h3 {
  margin: 0 0 8px;
  color: #111827;
  font-size: 22px;
}

.profile-summary p,
.score-copy p,
.timeline-item p,
.evidence-item p,
.skill-item p {
  color: #475569;
  line-height: 1.65;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-top: 14px;
}

.metric-grid div {
  min-width: 0;
  padding: 11px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #fff;
}

.metric-grid span,
.section-label {
  display: block;
  margin-bottom: 6px;
  color: #64748b;
  font-size: 12px;
  font-weight: 800;
}

.metric-grid strong {
  display: block;
  color: #111827;
  overflow-wrap: anywhere;
}

.section-block {
  margin-top: 18px;
}

.skill-list,
.timeline-list,
.evidence-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.skill-item,
.timeline-item,
.evidence-item,
.compact-item,
.signal-item {
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #fff;
  padding: 11px 12px;
}

.skill-item div,
.timeline-head {
  display: flex;
  justify-content: space-between;
  gap: 10px;
}

.skill-item span,
.timeline-head span,
.timeline-role,
.compact-item span,
.compact-item em,
.evidence-head span,
.score-copy span {
  color: #64748b;
  font-size: 12px;
  font-style: normal;
}

.timeline-item ul {
  margin: 8px 0 0 18px;
  padding: 0;
  color: #475569;
}

.two-column-block,
.signal-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.compact-item {
  display: flex;
  flex-direction: column;
  gap: 5px;
  margin-bottom: 8px;
}

.score-panel {
  display: grid;
  grid-template-columns: 150px minmax(0, 1fr);
  gap: 16px;
  align-items: center;
}

.score-copy {
  min-width: 0;
}

.dimension-list {
  display: flex;
  flex-direction: column;
  gap: 9px;
}

.dimension-row {
  display: grid;
  grid-template-columns: minmax(96px, 0.8fr) minmax(0, 1.5fr) 42px;
  gap: 10px;
  align-items: center;
}

.dimension-row span,
.dimension-row strong {
  color: #334155;
  font-size: 13px;
}

.signal-grid {
  margin-top: 18px;
}

.signal-block {
  min-width: 0;
}

.signal-item {
  margin-bottom: 8px;
  color: #334155;
  line-height: 1.55;
}

.signal-item--strength {
  border-left: 3px solid #22c55e;
}

.signal-item--risk {
  border-left: 3px solid #ef4444;
}

.tag-cloud {
  flex-wrap: wrap;
  gap: 7px;
}

.evidence-head {
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 7px;
}

.evidence-head strong {
  margin-left: auto;
  color: #1d4ed8;
}

.empty-inline {
  color: #94a3b8;
  font-size: 13px;
}

@media (max-width: 1120px) {
  .intelligence-grid,
  .two-column-block,
  .signal-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .intelligence-page {
    padding: 12px;
  }

  .intelligence-header {
    flex-direction: column;
  }

  .header-actions {
    justify-content: flex-start;
  }

  .metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .score-panel {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 520px) {
  .metric-grid,
  .dimension-row {
    grid-template-columns: 1fr;
  }
}

:global(:root[data-theme='dark']) .intelligence-header p,
:global(:root[data-theme='dark']) .profile-summary p,
:global(:root[data-theme='dark']) .score-copy p,
:global(:root[data-theme='dark']) .timeline-item p,
:global(:root[data-theme='dark']) .evidence-item p,
:global(:root[data-theme='dark']) .skill-item p {
  color: var(--text-secondary);
}

:global(:root[data-theme='dark']) .panel-card,
:global(:root[data-theme='dark']) .metric-grid div,
:global(:root[data-theme='dark']) .skill-item,
:global(:root[data-theme='dark']) .timeline-item,
:global(:root[data-theme='dark']) .evidence-item,
:global(:root[data-theme='dark']) .compact-item,
:global(:root[data-theme='dark']) .signal-item {
  border-color: var(--border);
  background: var(--surface);
}

:global(:root[data-theme='dark']) .panel-card :deep(.el-card__header) {
  background: var(--surface-muted);
}

:global(:root[data-theme='dark']) .profile-summary h3,
:global(:root[data-theme='dark']) .metric-grid strong,
:global(:root[data-theme='dark']) .dimension-row span,
:global(:root[data-theme='dark']) .dimension-row strong,
:global(:root[data-theme='dark']) .signal-item {
  color: var(--text-primary);
}

:global(:root[data-theme='dark']) .metric-grid span,
:global(:root[data-theme='dark']) .section-label,
:global(:root[data-theme='dark']) .skill-item span,
:global(:root[data-theme='dark']) .timeline-head span,
:global(:root[data-theme='dark']) .timeline-role,
:global(:root[data-theme='dark']) .compact-item span,
:global(:root[data-theme='dark']) .compact-item em,
:global(:root[data-theme='dark']) .evidence-head span,
:global(:root[data-theme='dark']) .score-copy span,
:global(:root[data-theme='dark']) .empty-inline {
  color: var(--text-muted);
}
</style>
