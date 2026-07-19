<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ArrowLeft, Link, Location, Phone, VideoCamera } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { getFeedback, getInterview, submitFeedback } from '@/api/interview'
import { useAuthStore } from '@/stores/auth'
import {
  DIMENSION_LABELS,
  INTERVIEW_MODE_LABEL,
  INTERVIEW_STATUS_LABEL,
  INTERVIEW_STATUS_TYPE,
  RECOMMENDATION_LABEL,
  RECOMMENDATION_TYPE,
  PERM,
  type InterviewFeedback,
  type InterviewSchedule,
} from '@/types/domain'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const submitting = ref(false)
const interview = ref<InterviewSchedule | null>(null)
const feedback = ref<InterviewFeedback | null>(null)
const interviewId = computed(() => Number(route.params.interviewId))
const form = reactive({
  recommendation: 'recommend',
  score: 80,
  comments: '',
  dimensions: { professional: 4, communication: 4, problem_solving: 4, job_fit: 4, potential: 4 } as Record<string, number>,
})

const canSubmit = computed(() => Boolean(
  interview.value
  && interview.value.status !== 'cancelled'
  && !feedback.value
  && auth.hasPermission(PERM.INTERVIEW_FEEDBACK_SUBMIT),
))
const formatDate = (value: string) => value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
const dimensionScores = computed(() => {
  if (!feedback.value?.dimension_scores_json) return []
  try {
    const values = JSON.parse(feedback.value.dimension_scores_json) as Record<string, number>
    return Object.entries(values).map(([key, score]) => ({ key, label: DIMENSION_LABELS[key] || key, score }))
  } catch {
    return []
  }
})

const load = async () => {
  if (!Number.isSafeInteger(interviewId.value) || interviewId.value <= 0) {
    router.replace('/hr/my-interviews')
    return
  }
  loading.value = true
  try {
    const [detail, feedbackResponse] = await Promise.all([
      getInterview(interviewId.value),
      getFeedback(interviewId.value).catch(() => ({ feedback: null })),
    ])
    interview.value = detail.interview
    feedback.value = feedbackResponse.feedback
  } finally {
    loading.value = false
  }
}

const handleSubmit = async () => {
  if (!interview.value || submitting.value) return
  if (!form.comments.trim()) {
    ElMessage.warning('请填写面试评价')
    return
  }
  submitting.value = true
  try {
    await submitFeedback(interviewId.value, {
      application_id: interview.value.application_id,
      recommendation: form.recommendation,
      score: form.score,
      dimension_scores_json: JSON.stringify(form.dimensions),
      comments: form.comments.trim(),
    })
    ElMessage.success('面试反馈已提交')
    await load()
  } finally {
    submitting.value = false
  }
}

onMounted(load)
</script>

<template>
  <section v-loading="loading" class="interview-detail workspace-surface">
    <header class="page-header workspace-surface__header interview-detail__header">
      <div class="workspace-surface__header-copy">
        <el-button link type="primary" :icon="ArrowLeft" @click="router.push('/hr/my-interviews')">返回我的面试</el-button>
        <h1 class="console-title">{{ interview?.title || '面试详情' }}</h1>
        <p v-if="interview" class="console-description">{{ interview.candidate_name }} · {{ interview.job_title }} · 第 {{ interview.round_no }} 轮</p>
      </div>
      <div class="workspace-surface__header-actions">
        <el-tag v-if="interview" :type="INTERVIEW_STATUS_TYPE[interview.status] || 'info'" size="large">
          {{ INTERVIEW_STATUS_LABEL[interview.status] || interview.status }}
        </el-tag>
      </div>
    </header>

    <div class="workspace-surface__divider"></div>

    <div v-if="interview" class="workspace-surface__body interview-detail__body">
      <section class="interview-section" aria-labelledby="interview-schedule-title">
        <div class="interview-section__header">
          <div>
            <h2 id="interview-schedule-title">面试安排</h2>
            <p>本轮面试的时间、方式及联系信息。</p>
          </div>
        </div>
        <el-descriptions :column="2" border>
          <el-descriptions-item label="面试时间">{{ formatDate(interview.scheduled_at) }}</el-descriptions-item>
          <el-descriptions-item label="时长">{{ interview.duration_minutes }} 分钟</el-descriptions-item>
          <el-descriptions-item label="方式">{{ INTERVIEW_MODE_LABEL[interview.mode] || interview.mode }}</el-descriptions-item>
          <el-descriptions-item label="地点">{{ interview.location || '-' }}</el-descriptions-item>
          <el-descriptions-item label="候选人电话">{{ interview.candidate_phone || '-' }}</el-descriptions-item>
          <el-descriptions-item label="内部备注">{{ interview.internal_note || '-' }}</el-descriptions-item>
        </el-descriptions>
        <div v-if="interview.meeting_url" class="meeting-action">
          <el-button type="primary" :icon="interview.mode === 'phone' ? Phone : VideoCamera" tag="a" :href="interview.meeting_url" target="_blank">
            进入面试会议
          </el-button>
          <span><el-icon><Link /></el-icon>{{ interview.meeting_url }}</span>
        </div>
        <div v-else-if="interview.location" class="meeting-action"><el-icon><Location /></el-icon>{{ interview.location }}</div>
      </section>

      <div class="interview-section-divider"></div>

      <section class="interview-section" aria-labelledby="interview-feedback-title">
        <div class="interview-section__header">
          <div>
            <h2 id="interview-feedback-title">面试反馈</h2>
            <p>记录面试结论、维度评分和综合评价。</p>
          </div>
        </div>
        <div v-if="feedback" class="feedback-result">
          <div class="feedback-summary">
            <el-tag :type="RECOMMENDATION_TYPE[feedback.recommendation] || 'info'">{{ RECOMMENDATION_LABEL[feedback.recommendation] || feedback.recommendation }}</el-tag>
            <strong>综合评分 {{ feedback.score }} / 100</strong>
            <span>提交于 {{ formatDate(feedback.submitted_at) }}</span>
          </div>
          <div class="dimension-grid">
            <div v-for="item in dimensionScores" :key="item.key"><span>{{ item.label }}</span><el-rate :model-value="item.score" disabled /></div>
          </div>
          <p class="comments">{{ feedback.comments }}</p>
        </div>

        <el-form v-else-if="canSubmit" label-position="top" @submit.prevent="handleSubmit">
          <div class="feedback-fields">
            <el-form-item label="面试结论" required>
              <el-select v-model="form.recommendation">
                <el-option v-for="(label, value) in RECOMMENDATION_LABEL" :key="value" :label="label" :value="value" />
              </el-select>
            </el-form-item>
            <el-form-item label="综合评分（0-100）" required><el-input-number v-model="form.score" :min="0" :max="100" /></el-form-item>
          </div>
          <div class="dimension-grid">
            <el-form-item v-for="(label, key) in DIMENSION_LABELS" :key="key" :label="label"><el-rate v-model="form.dimensions[key]" /></el-form-item>
          </div>
          <el-form-item label="面试评价" required><el-input v-model="form.comments" type="textarea" :rows="5" maxlength="2000" show-word-limit /></el-form-item>
          <el-button type="primary" native-type="submit" :loading="submitting">提交反馈</el-button>
        </el-form>
        <el-empty v-else :description="interview.status === 'cancelled' ? '面试已取消，无需提交反馈' : '你没有提交此面试反馈的权限'" />
      </section>
    </div>
  </section>
</template>

<style scoped>
.interview-detail__header.page-header {
  align-items: flex-end;
  margin: 0;
  border: 0;
  border-radius: 0;
}
.interview-detail__header h1 { margin-top: 10px; }
.interview-detail__body {
  padding: 24px;
  overflow-x: hidden;
  overflow-y: auto;
}
.interview-section {
  min-width: 0;
}
.interview-section__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}
.interview-section__header h2 {
  margin: 0;
  color: var(--text-primary);
  font-size: 17px;
  line-height: 1.4;
}
.interview-section__header p {
  margin: 4px 0 0;
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.5;
}
.interview-section-divider {
  height: 1px;
  margin: 28px 0;
  background: var(--border);
}
.meeting-action { display: flex; align-items: center; gap: 12px; margin-top: 18px; color: var(--text-secondary); word-break: break-all; }
.feedback-summary { display: flex; align-items: center; gap: 16px; margin-bottom: 20px; }
.feedback-summary span:last-child { color: var(--text-muted); }
.feedback-fields, .dimension-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px 24px; }
.dimension-grid > div { display: flex; align-items: center; justify-content: space-between; }
.comments { padding: 16px; border-radius: 8px; background: var(--surface-secondary); white-space: pre-wrap; }
@media (max-width: 720px) {
  .interview-detail { overflow: visible; }
  .interview-detail__header.page-header { align-items: flex-start; }
  .interview-detail__body { padding: 18px 16px; overflow: visible; }
  .feedback-fields, .dimension-grid { grid-template-columns: 1fr; }
  .feedback-summary { align-items: flex-start; flex-direction: column; }
}
</style>
