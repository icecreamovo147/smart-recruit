<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElNotification } from 'element-plus'
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
import { debugLog } from '@shared/utils/debugLog'
import { formatShanghaiDateTime } from '@shared/utils/format'

type JsonRecord = Record<string, unknown>
type JsonListItem = string | number | JsonRecord
type DimensionScore = {
  key: string
  label?: string
  score: number
  weight?: number
  matched: string[]
  missing: string[]
}

type SnippetRow = { key: string; value: string; hasHtml: boolean }
type EvidenceDisplayItem = CandidateMatchEvidenceInfo & {
  dimensionText: string
  sourceText: string
  typeText: string
  impactText: string
  reason: string
  snippetRows: SnippetRow[]
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
const actionLoadingText = computed(() => {
  if (actionLoading.value === 'parse') return '正在解析简历画像，请稍候...'
  if (actionLoading.value === 'evaluate') return '正在重新评估岗位匹配，请稍候...'
  return ''
})

const canRunAIAction = computed(() => auth.hasPermission(PERM.AI_HR_USE))
const candidateName = computed(() => String(route.query.candidate_name || profile.value?.profile?.full_name || '候选人'))
const jobTitle = computed(() => String(route.query.job_title || '当前投递岗位'))
const jobId = computed(() => Number(route.query.job_id || evaluation.value?.job_id || 0))
const evaluation = computed<CandidateMatchEvaluationInfo | null>(() => evaluationSnapshot.value?.evaluation || null)
const evidence = computed<CandidateMatchEvidenceInfo[]>(() => evaluationSnapshot.value?.evidence || [])

const formatDateTime = (value?: string): string => formatShanghaiDateTime(value, '-', false)

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

const asSignalText = (value: unknown): string => {
  if (value && typeof value === 'object') {
    const record = value as JsonRecord
    return asText(record.message || record.summary || record.name || record.title || record.requirement || record.code || value)
  }
  return asText(value)
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
  if (typeof item === 'number') return { key: fallbackKey, score: item, matched: [], missing: [] }
  if (typeof item === 'string') {
    const score = Number(item)
    return Number.isFinite(score) ? { key: fallbackKey, score, matched: [], missing: [] } : null
  }
  if (!item || typeof item !== 'object') return null

  const record = item as JsonRecord
  const key = asText(record.name || record.key || record.dimension || fallbackKey)
  const label = asText(record.label)
  const scoreValue = record.score ?? record.value ?? record.percentage
  const score = Number(scoreValue)
  const weight = Number(record.weight)
  const matched = Array.isArray(record.matched) ? record.matched.map(asText).filter(Boolean) : []
  const missing = Array.isArray(record.missing) ? record.missing.map(asText).filter(Boolean) : []
  return key || label || Number.isFinite(score)
    ? { key, label, score: Number.isFinite(score) ? score : 0, weight: Number.isFinite(weight) ? weight : undefined, matched, missing }
    : null
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
const strengthTexts = computed(() => strengths.value.map(asSignalText).filter(Boolean))
const riskTexts = computed(() => risks.value.map(asSignalText).filter(Boolean))
const requirementLabelMap = computed(() => {
  const labels = new Map<string, string>()
  const profile = scoreBreakdown.value.requirement_profile
  if (profile && typeof profile === 'object') {
    const requirements = (profile as JsonRecord).requirements
    if (Array.isArray(requirements)) {
      for (const item of requirements) {
        if (!item || typeof item !== 'object') continue
        const record = item as JsonRecord
        const id = asText(record.id)
        const label = asText(record.label || record.description)
        if (id && label) labels.set(id, label)
      }
    }
  }
  return labels
})

const requirementDisplayText = (value: string): string => requirementLabelMap.value.get(value) || dimensionLabels[value] || value

const missingRequirements = computed(() => {
  const fromField = jsonTags(evaluation.value?.missing_requirements_json).map(requirementDisplayText)
  if (fromField.length) return fromField

  const fromBreakdown = scoreBreakdown.value.missing_requirements
  if (!Array.isArray(fromBreakdown)) return []
  return fromBreakdown.map(asText).filter(Boolean).map(requirementDisplayText)
})

const dimensionLabels: Record<string, string> = {
  skills: '技能匹配',
  requirements: '岗位要求',
  term_coverage: '要求覆盖度',
  must_have: '必备要求满足度',
  core_skills: '核心技能匹配',
  growth: '成长潜力与加分项',
  experience: '经验背景',
  education: '教育背景',
  profile: '候选人资料',
  overall: '综合评估',
}

const parseRunStatusLabels: Record<string, string> = {
  running: '解析中',
  succeeded: '解析成功',
  failed: '解析失败',
}

const evidenceTypeLabels: Record<string, string> = {
  skill: '技能证据',
  experience: '经历证据',
  resume_text: '简历文本',
  missing_requirement: '缺失要求',
  candidate_profile: '候选人资料',
  requirement_match: '要求匹配证据',
  no_match: '无匹配证据',
  strength: '优势证据',
  risk: '风险信号',
  evidence: '证据',
}

const sourceTableLabels: Record<string, string> = {
  resume_skills: '技能画像',
  resume_experiences: '工作经历',
  resume_projects: '项目经历',
  resume_educations: '教育经历',
  resumes: '简历原文',
  jobs: '岗位要求',
  candidate_profiles: '候选人资料',
  resume_profiles: '简历画像',
}

const parseRunStatusLabel = (status: string): string => parseRunStatusLabels[status] || status || '-'
const dimensionLabel = (key: string): string => requirementLabelMap.value.get(key) || dimensionLabels[key] || key || '-'
const dimensionDisplayLabel = (item: DimensionScore): string => item.label || dimensionLabel(item.key)
const evidenceTypeLabel = (type: string): string => evidenceTypeLabels[type] || type || '证据'
const sourceTableLabel = (source: string): string => sourceTableLabels[source] || source || '-'
const evidenceTagType = (type: string): string => {
  if (type === 'risk' || type === 'missing_requirement') return 'danger'
  if (type === 'candidate_profile') return 'info'
  return 'success'
}
const parsedSnippet = (snippet: string): SnippetRow[] => {
  if (!snippet) return []
  const htmlTagRe = /<[a-z][\s\S]*>/i
  const parts = snippet.split('|').map(s => s.trim()).filter(Boolean)
  if (parts.length < 2) return [{ key: '', value: snippet, hasHtml: htmlTagRe.test(snippet) }]
  return parts.map(part => {
    const idx = part.indexOf(': ')
    if (idx === -1) return { key: '', value: part, hasHtml: htmlTagRe.test(part) }
    const key = part.slice(0, idx).trim()
    const value = part.slice(idx + 2).trim()
    return { key, value, hasHtml: htmlTagRe.test(value) }
  })
}

const scoreImpactText = (value: number): string => {
  const score = Number(value || 0)
  return `${score > 0 ? '+' : ''}${Math.round(score)}`
}

const evidenceMetadata = (item: CandidateMatchEvidenceInfo): JsonRecord => parseJson<JsonRecord>(item.metadata_json, {})
const evidenceDimensionText = (item: CandidateMatchEvidenceInfo): string => {
  const metadata = evidenceMetadata(item)
  const requirementId = asText(metadata.requirement_id)
  return dimensionLabel(requirementId || item.dimension)
}
const evidenceSourceText = (item: CandidateMatchEvidenceInfo): string => {
  const base = sourceTableLabel(item.source_table)
  return item.source_id ? `${base} #${item.source_id}` : base
}

const evidenceDisplayItems = computed<EvidenceDisplayItem[]>(() => evidence.value.map((item) => {
  const metadata = evidenceMetadata(item)
  return {
    ...item,
    dimensionText: evidenceDimensionText(item),
    sourceText: evidenceSourceText(item),
    typeText: evidenceTypeLabel(item.evidence_type),
    impactText: scoreImpactText(item.score_impact),
    reason: asText(metadata.reason),
    snippetRows: parsedSnippet(item.snippet),
  }
}))

const evidenceByDimension = computed(() => {
  const groups = new Map<string, EvidenceDisplayItem[]>()
  for (const item of evidenceDisplayItems.value) {
    const key = item.dimensionText || '综合评估'
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key)!.push(item)
  }
  return Array.from(groups.entries()).map(([dimension, items]) => {
    const representativeImpact = items
      .map(item => Number(item.score_impact || 0))
      .sort((a, b) => Math.abs(b) - Math.abs(a))[0] || 0
    return { dimension, items, impactText: scoreImpactText(representativeImpact) }
  })
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

const isMissingBusinessError = (error: unknown, messages: string[]): boolean => {
  if (!(error instanceof BusinessError)) return false
  if (error.code === 404) return true
  if (error.code !== 400) return false
  return messages.includes(error.message)
}

const isProfileMissingError = (error: unknown): boolean =>
  isMissingBusinessError(error, ['current resume profile not found'])

const isEvaluationMissingError = (error: unknown): boolean =>
  isMissingBusinessError(error, ['candidate match evaluation not found'])

const loadProfile = async () => {
  if (!applicationId.value) return
  debugLog.ri.info('loadProfile_started', { application_id: applicationId.value })
  profileLoading.value = true
  profileMissing.value = false
  try {
    const data = await getApplicationResumeProfile(applicationId.value, { silentError: true })
    profile.value = data.profile?.profile ? data.profile : null
    profileMissing.value = !profile.value
    debugLog.ri.info('loadProfile_finished', {
      application_id: applicationId.value,
      status: profileMissing.value ? 'empty' : 'succeeded',
      profile_id: profile.value?.profile?.id ?? null,
      version: profile.value?.profile?.version ?? null,
    })
  } catch (error: unknown) {
    if (isProfileMissingError(error)) {
      profile.value = null
      profileMissing.value = true
      debugLog.ri.info('loadProfile_finished', { application_id: applicationId.value, status: 'not_found' })
      return
    }
    debugLog.ri.error('loadProfile_failed', { application_id: applicationId.value, error: (error as Error)?.message })
    throw error
  } finally {
    profileLoading.value = false
  }
}

const loadEvaluation = async () => {
  if (!applicationId.value) return
  debugLog.ri.info('loadEvaluation_started', { application_id: applicationId.value })
  evaluationLoading.value = true
  evaluationMissing.value = false
  try {
    const data = await getCandidateMatchEvaluation(applicationId.value, undefined, { silentError: true })
    evaluationSnapshot.value = data.evaluation?.evaluation ? data.evaluation : null
    evaluationMissing.value = !evaluationSnapshot.value
    debugLog.ri.info('loadEvaluation_finished', {
      application_id: applicationId.value,
      status: evaluationMissing.value ? 'empty' : 'succeeded',
      evaluation_id: evaluationSnapshot.value?.evaluation?.id ?? null,
      overall_score: evaluationSnapshot.value?.evaluation?.overall_score ?? null,
    })
  } catch (error: unknown) {
    if (isEvaluationMissingError(error)) {
      evaluationSnapshot.value = null
      evaluationMissing.value = true
      debugLog.ri.info('loadEvaluation_finished', { application_id: applicationId.value, status: 'not_found' })
      return
    }
    debugLog.ri.error('loadEvaluation_failed', { application_id: applicationId.value, error: (error as Error)?.message })
    throw error
  } finally {
    evaluationLoading.value = false
  }
}

const loadAll = async () => {
  debugLog.ri.info('loadAll_started', { application_id: applicationId.value, job_id: jobId.value })
  try {
    await Promise.all([loadProfile(), loadEvaluation()])
    debugLog.ri.info('loadAll_finished', { application_id: applicationId.value })
  } catch (error: any) {
    debugLog.ri.error('loadAll_failed', { application_id: applicationId.value, error: error?.message })
    ElMessage.error(error?.message || '智能评估信息加载失败')
  }
}

const runParse = async () => {
  if (!canRunAIAction.value) {
    debugLog.ri.warn('runParse_skipped', { application_id: applicationId.value, reason: 'no_permission' })
    return
  }
  if (actionLoading.value) return
  debugLog.ri.info('runParse_started', { application_id: applicationId.value })
  actionLoading.value = 'parse'
  try {
    const data = await parseResumeProfile({ application_id: applicationId.value })
    profile.value = data.profile?.profile ? data.profile : null
    profileMissing.value = !profile.value
    ElNotification({
      title: '简历画像解析完成',
      message: profile.value ? '结构化画像已更新到当前页面。' : '接口已完成，但未返回可展示的画像数据。',
      type: profile.value ? 'success' : 'warning',
      position: 'top-right',
      duration: 4500,
    })
    debugLog.ri.info('runParse_succeeded', { application_id: applicationId.value, has_profile: !!profile.value })
  } catch (error: unknown) {
    debugLog.ri.warn('runParse_fallback', { application_id: applicationId.value })
    await loadProfile().catch(() => undefined)
    ElNotification({
      title: '简历画像解析未完成',
      message: (error as Error)?.message || '接口执行失败，请稍后重试。',
      type: 'error',
      position: 'top-right',
      duration: 6000,
    })
  } finally {
    actionLoading.value = ''
  }
}

const runEvaluation = async () => {
  if (!canRunAIAction.value) {
    debugLog.ri.warn('runEvaluation_skipped', { application_id: applicationId.value, reason: 'no_permission' })
    return
  }
  if (actionLoading.value) return
  debugLog.ri.info('runEvaluation_started', { application_id: applicationId.value })
  actionLoading.value = 'evaluate'
  try {
    const data = await evaluateCandidateMatch(applicationId.value)
    evaluationSnapshot.value = data.evaluation?.evaluation ? data.evaluation : null
    evaluationMissing.value = !evaluationSnapshot.value
    ElNotification({
      title: '岗位匹配评估完成',
      message: evaluation.value
        ? `综合评分 ${Math.round(evaluation.value.overall_score || 0)}，结论：${recommendationLabel.value}`
        : '接口已完成，但未返回可展示的评估数据。',
      type: evaluation.value ? 'success' : 'warning',
      position: 'top-right',
      duration: 5000,
    })
    debugLog.ri.info('runEvaluation_succeeded', {
      application_id: applicationId.value,
      evaluation_id: evaluationSnapshot.value?.evaluation?.id ?? null,
      overall_score: evaluationSnapshot.value?.evaluation?.overall_score ?? null,
      recommendation: evaluationSnapshot.value?.evaluation?.recommendation ?? null,
    })
    if (!profile.value) await loadProfile().catch(() => undefined)
  } catch (error: unknown) {
    debugLog.ri.warn('runEvaluation_fallback', { application_id: applicationId.value })
    await loadEvaluation().catch(() => undefined)
    ElNotification({
      title: '岗位匹配评估未完成',
      message: (error as Error)?.message || '接口执行失败，请稍后重试。',
      type: 'error',
      position: 'top-right',
      duration: 6000,
    })
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
  <section class="console-page console-page--fill">
    <div
      class="workspace-surface"
      v-loading="Boolean(actionLoading)"
      :element-loading-text="actionLoadingText"
      element-loading-background="rgba(255, 255, 255, 0.76)"
    >
      <div class="workspace-surface__header">
        <div class="workspace-surface__header-copy">
          <p class="console-eyebrow">RECRUITING INTELLIGENCE</p>
          <h1 class="console-title">简历画像与匹配评估</h1>
          <p class="console-description">{{ candidateName }} / {{ jobTitle }} / 投递 #{{ applicationId }}</p>
        </div>
        <div class="workspace-surface__header-actions">
          <el-button :icon="ArrowLeft" @click="goBack">返回</el-button>
          <el-button :icon="Refresh" :disabled="Boolean(actionLoading)" @click="loadAll">刷新</el-button>
        </div>
      </div>

      <div class="workspace-surface__divider"></div>

      <div class="workspace-surface__toolbar">
        <div class="workspace-surface__filters" />
        <div class="workspace-surface__actions">
          <el-tooltip :disabled="canRunAIAction" content="需要 AI HR 使用权限">
            <span>
              <el-button :icon="DocumentChecked" :loading="actionLoading === 'parse'" :disabled="!canRunAIAction || Boolean(actionLoading)" @click="runParse">
                {{ profile ? '重新解析画像' : '解析画像' }}
              </el-button>
            </span>
          </el-tooltip>
          <el-tooltip :disabled="canRunAIAction" content="需要 AI HR 使用权限">
            <span>
              <el-button type="primary" :icon="Cpu" :loading="actionLoading === 'evaluate'" :disabled="!canRunAIAction || Boolean(actionLoading)" @click="runEvaluation">
                {{ evaluation ? '重新评估匹配' : '生成匹配评估' }}
              </el-button>
            </span>
          </el-tooltip>
        </div>
      </div>

      <div class="intelligence-grid">
        <section class="intelligence-column" v-loading="profileLoading">
          <el-card shadow="never" class="panel-card">
            <template #header>
              <div class="panel-title">
                <span>结构化简历画像</span>
                <div class="panel-tags">
                  <el-tag v-if="profile?.profile?.is_current" size="small" type="success">当前版本</el-tag>
                  <el-tag v-if="profile?.profile" size="small" type="primary">v{{ profile.profile.version }}</el-tag>
                  <el-tag v-if="profile?.parse_run?.status" size="small" type="info">{{ parseRunStatusLabel(profile.parse_run.status) }}</el-tag>
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
                </div>
              </div>
            </template>

            <template v-if="evaluation">
              <div class="score-panel">
                <div class="dashboard-wrap">
                  <el-progress type="dashboard" :percentage="Math.round(evaluation.overall_score || 0)" :status="scoreStatus as any" :width="120" :stroke-width="10" />
                  <span class="dashboard-score">{{ Math.round(evaluation.overall_score || 0) }}</span>
                </div>
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
                    <div class="dimension-main">
                      <span>{{ dimensionDisplayLabel(item) }}</span>
                      <em v-if="item.weight">权重 {{ Math.round(item.weight * 100) }}%</em>
                    </div>
                    <div class="dimension-meter">
                      <el-progress :percentage="Math.round(item.score)" :show-text="false" />
                      <strong>{{ Math.round(item.score) }}</strong>
                    </div>
                    <div v-if="item.matched.length || item.missing.length" class="dimension-tags">
                      <el-tag v-for="tag in item.matched" :key="`m-${item.key}-${tag}`" size="small" type="success" effect="plain">{{ tag }}</el-tag>
                      <el-tag v-for="tag in item.missing" :key="`x-${item.key}-${tag}`" size="small" type="warning" effect="plain">{{ tag }}</el-tag>
                    </div>
                  </div>
                </div>
              </div>

              <div class="signal-grid">
                <div class="signal-block">
                  <div class="section-label">优势</div>
                  <div v-for="item in strengthTexts" :key="item" class="signal-item signal-item--strength">{{ item }}</div>
                  <div v-if="strengthTexts.length === 0" class="empty-inline">暂无优势信号</div>
                </div>
                <div class="signal-block">
                  <div class="section-label">风险</div>
                  <div v-for="item in riskTexts" :key="item" class="signal-item signal-item--risk">{{ item }}</div>
                  <div v-if="riskTexts.length === 0" class="empty-inline">暂无风险信号</div>
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
                  <div v-for="group in evidenceByDimension" :key="group.dimension" class="evidence-group">
                    <div class="evidence-group-title">
                      <div>
                        <span>匹配维度</span>
                        <strong>{{ group.dimension }}</strong>
                      </div>
                      <div class="evidence-group-stats">
                        <span>{{ group.items.length }} 条证据</span>
                        <strong class="evidence-impact">影响 {{ group.impactText }}</strong>
                      </div>
                    </div>
                    <div v-for="item in group.items" :key="item.id" class="evidence-item">
                      <div class="evidence-head">
                        <div class="evidence-meta">
                          <el-tag size="small" :type="evidenceTagType(item.evidence_type) as any" effect="plain">{{ item.typeText }}</el-tag>
                          <el-tag size="small" type="info" effect="plain">{{ item.sourceText }}</el-tag>
                        </div>
                      </div>
                      <div v-if="item.reason" class="evidence-reason">{{ item.reason }}</div>
                      <div v-if="item.snippetRows.length" class="snippet-list">
                        <div v-for="(kv, i) in item.snippetRows" :key="i" class="snippet-row">
                          <span v-if="kv.key" class="snippet-key">{{ kv.key }}</span>
                          <span v-if="kv.hasHtml" class="snippet-value" v-html="kv.value" />
                          <span v-else class="snippet-value">{{ kv.value }}</span>
                        </div>
                      </div>
                      <span v-else class="snippet-empty">暂无证据片段</span>
                    </div>
                  </div>
                  <div v-if="evidence.length === 0" class="empty-inline">暂无证据明细</div>
                </div>
              </div>
            </template>
            <el-empty v-else :description="evaluationMissing ? '暂无匹配评估，可发起生成' : '评估数据未返回'" />
          </el-card>
        </section>
      </div>
    </div>
  </section>
</template>

<style scoped>
.intelligence-grid {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(0, 1.04fr) minmax(0, 0.96fr);
  gap: 16px;
  align-items: start;
  overflow-y: auto;
  padding: 16px 24px 24px;
}

.panel-card {
  border-radius: 8px;
  border-color: var(--border);
}

.panel-card :deep(.el-card__header) {
  padding: 13px 16px;
  background: var(--surface-muted);
}

.panel-title,
.panel-tags,
.tag-cloud,
.evidence-head {
  display: flex;
  align-items: center;
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
  color: var(--text-primary);
  font-size: 22px;
}

.profile-summary p,
.score-copy p,
.timeline-item p,
.evidence-item p,
.skill-item p {
  color: var(--text-secondary);
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
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
}

.metric-grid span,
.section-label {
  display: block;
  margin-bottom: 6px;
  color: var(--text-muted);
  font-size: 12px;
  font-weight: 800;
}

.metric-grid strong {
  display: block;
  color: var(--text-primary);
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
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  padding: 11px 12px;
}

.evidence-item {
  border-left: 3px solid var(--brand-strong);
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
  color: var(--text-muted);
  font-size: 12px;
  font-style: normal;
}

.timeline-item ul {
  margin: 8px 0 0 18px;
  padding: 0;
  color: var(--text-secondary);
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
  grid-template-columns: 140px minmax(0, 1fr);
  gap: 16px;
  align-items: center;
}

.dashboard-wrap {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.dashboard-wrap :deep(.el-progress) {
  display: flex;
}

.dashboard-wrap :deep(.el-progress__text) {
  display: none;
}

.dashboard-score {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: 26px;
  font-weight: 800;
  color: var(--text-primary);
  line-height: 1;
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
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 11px 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
}

.dimension-main,
.dimension-meter,
.evidence-group-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.dimension-main {
  justify-content: space-between;
}

.dimension-main span {
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 700;
}

.dimension-main em,
.dimension-meter strong,
.evidence-group-title span {
  color: var(--text-secondary);
  font-size: 13px;
  font-style: normal;
}

.dimension-meter :deep(.el-progress) {
  flex: 1;
  min-width: 0;
}

.dimension-meter strong {
  width: 38px;
  text-align: right;
}

.dimension-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.signal-grid {
  margin-top: 18px;
}

.signal-block {
  min-width: 0;
}

.signal-item {
  margin-bottom: 8px;
  color: var(--text-secondary);
  line-height: 1.55;
}

.signal-item--strength {
  border-left: 3px solid var(--el-color-success, #22c55e);
}

.signal-item--risk {
  border-left: 3px solid var(--el-color-danger, #ef4444);
}

.tag-cloud {
  flex-wrap: wrap;
  gap: 7px;
}

.evidence-head {
  flex-wrap: wrap;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 9px;
}

.evidence-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-muted);
}

.evidence-group + .evidence-group {
  margin-top: 4px;
}

.evidence-group-title {
  justify-content: space-between;
  gap: 12px;
  padding: 0 1px 2px;
}

.evidence-group-title div {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.evidence-group-title .evidence-group-stats {
  min-width: max-content;
  align-items: flex-end;
}

.evidence-group-title span,
.evidence-group-title em {
  color: var(--text-muted);
  font-size: 12px;
  font-style: normal;
}

.evidence-group-title strong {
  color: var(--text-primary);
  font-size: 14px;
  overflow-wrap: anywhere;
}

.evidence-group-title .evidence-impact {
  color: var(--brand);
  font-size: 13px;
  font-weight: 800;
}

.evidence-head strong {
  color: var(--brand-strong);
  font-size: 13px;
}

.evidence-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.evidence-reason {
  margin-bottom: 8px;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.55;
  overflow-wrap: anywhere;
}

.snippet-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 6px;
}

.snippet-row {
  display: flex;
  align-items: baseline;
  gap: 6px;
  line-height: 1.55;
}

.snippet-key {
  color: var(--text-muted);
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
  min-width: 3.5em;
}

.snippet-value {
  color: var(--text-secondary);
  font-size: 13px;
  overflow-wrap: anywhere;
}

.snippet-value :deep(p) {
  margin: 0;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.55;
}

.snippet-empty {
  color: var(--text-faint);
  font-size: 13px;
}

.empty-inline {
  color: var(--text-faint);
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
  .intelligence-grid {
    padding: 12px;
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
</style>
