<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, VideoCamera, Phone, OfficeBuilding, Clock, Document, User, MapLocation } from '@element-plus/icons-vue'
import { getInterview, getFeedback } from '@/api/interview'
import type { InterviewSchedule, InterviewFeedback } from '@/types/domain'
import {
  INTERVIEW_STATUS_LABEL,
  INTERVIEW_STATUS_TYPE,
  INTERVIEW_MODE_LABEL,
  RECOMMENDATION_LABEL,
  DIMENSION_LABELS,
} from '@/types/domain'
import FeedbackForm from '@/components/FeedbackForm.vue'
import FeedbackDisplay from '@/components/FeedbackDisplay.vue'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const interview = ref<InterviewSchedule | null>(null)
const feedback = ref<InterviewFeedback | null>(null)
const feedbackLoaded = ref(false)

const interviewId = computed(() => Number(route.params.interviewId))

const canSubmitFeedback = computed(() => {
  if (!interview.value) return false
  // Allow feedback as long as interview is not cancelled and feedback not yet submitted.
  // The interview status itself should not block feedback — submitting feedback is what
  // transitions the interview to 'completed', so requiring 'completed' creates a deadlock.
  return interview.value.status !== 'cancelled' && !feedback.value
})

const hasFeedback = computed(() => Boolean(feedback.value))

const modeIcon = computed(() => {
  if (!interview.value) return VideoCamera
  const m = interview.value.mode
  if (m === 'video') return VideoCamera
  if (m === 'phone') return Phone
  return OfficeBuilding
})

const formatDateTime = (iso: string): string => {
  if (!iso) return '-'
  const d = new Date(iso)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

const formatDuration = (mins: number): string => {
  if (mins >= 60) {
    const h = Math.floor(mins / 60)
    const m = mins % 60
    return m > 0 ? `${h} 小时 ${m} 分钟` : `${h} 小时`
  }
  return `${mins} 分钟`
}

const dimensionScores = computed(() => {
  if (!feedback.value?.dimension_scores_json) return []
  try {
    const parsed = JSON.parse(feedback.value.dimension_scores_json)
    return Object.entries(parsed).map(([key, val]) => ({
      label: DIMENSION_LABELS[key] || key,
      score: val as number,
    }))
  } catch {
    return []
  }
})

const openMeeting = () => {
  if (interview.value?.meeting_url) {
    window.open(interview.value.meeting_url, '_blank')
  }
}

const handleFeedbackSubmitted = (fb: InterviewFeedback) => {
  feedback.value = fb
}

const loadData = async () => {
  loading.value = true
  try {
    const [interviewData] = await Promise.all([
      getInterview(interviewId.value),
    ])
    interview.value = interviewData

    // Try to load existing feedback
    try {
      const fbData = await getFeedback(interviewId.value)
      feedback.value = fbData
    } catch {
      // No feedback yet or API error — that's fine
    }
    feedbackLoaded.value = true
  } catch {
    // Handled by request interceptor
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
</script>

<template>
  <div class="interview-detail">
    <!-- Breadcrumb -->
    <div class="detail-breadcrumb">
      <el-button type="primary" link @click="router.push('/interviews')">
        <el-icon><ArrowLeft /></el-icon>
        返回面试列表
      </el-button>
    </div>

    <el-skeleton v-if="loading" :rows="10" animated />

    <template v-else-if="interview">
      <div class="detail-body">
        <!-- Left main column -->
        <div class="detail-main">
          <!-- Interview info -->
          <section class="detail-section">
            <h3 class="section-title">面试信息</h3>
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">面试标题</span>
                <span class="info-value">{{ interview.title }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">面试轮次</span>
                <span class="info-value">第 {{ interview.round_no }} 轮</span>
              </div>
              <div class="info-item">
                <span class="info-label">面试方式</span>
                <span class="info-value">
                  <el-icon><component :is="modeIcon" /></el-icon>
                  {{ INTERVIEW_MODE_LABEL[interview.mode] || interview.mode }}
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">面试时长</span>
                <span class="info-value">{{ formatDuration(interview.duration_minutes) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">状态</span>
                <span class="info-value">
                  <el-tag :type="(INTERVIEW_STATUS_TYPE[interview.status] || 'info') as any" size="small" effect="light">
                    {{ INTERVIEW_STATUS_LABEL[interview.status] || interview.status }}
                  </el-tag>
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">面试时间</span>
                <span class="info-value">{{ formatDateTime(interview.scheduled_at) }}</span>
              </div>
            </div>
            <div v-if="interview.cancel_reason" class="cancel-reason">
              <el-icon color="var(--danger)"><Clock /></el-icon>
              <span>取消原因：{{ interview.cancel_reason }}</span>
            </div>
          </section>

          <!-- Candidate info -->
          <section class="detail-section">
            <h3 class="section-title">候选人信息</h3>
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">
                  <el-icon><User /></el-icon>
                  姓名
                </span>
                <span class="info-value">{{ interview.candidate_name }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">
                  <el-icon><Phone /></el-icon>
                  电话
                </span>
                <span class="info-value">{{ interview.candidate_phone || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">
                  <el-icon><Document /></el-icon>
                  应聘岗位
                </span>
                <span class="info-value">{{ interview.job_title }}</span>
              </div>
            </div>
          </section>

          <!-- Candidate materials -->
          <section class="detail-section">
            <h3 class="section-title">候选人材料</h3>
            <div v-if="interview.resume_url" class="resume-link-wrapper">
              <el-icon><Document /></el-icon>
              <a
                :href="interview.resume_url"
                target="_blank"
                rel="noopener noreferrer"
                class="resume-link"
              >
                {{ interview.candidate_name }}-{{ interview.job_title }}-个人简历
              </a>
            </div>
            <p v-else class="material-hint">暂无简历附件</p>
          </section>

          <!-- Feedback section -->
          <section class="detail-section">
            <h3 class="section-title">面试反馈</h3>
            <FeedbackDisplay
              v-if="hasFeedback && feedback"
              :feedback="feedback"
              :dimension-scores="dimensionScores"
            />
            <FeedbackForm
              v-else-if="canSubmitFeedback && feedbackLoaded"
              :interview="interview"
              @submitted="handleFeedbackSubmitted"
            />
            <el-empty
              v-else-if="!hasFeedback && !canSubmitFeedback && feedbackLoaded"
              description="面试已取消，无需提交反馈"
              :image-size="80"
            />
          </section>
        </div>

        <!-- Right sidebar -->
        <aside class="detail-sidebar">
          <!-- Schedule card -->
          <div class="sidebar-card">
            <h4 class="sidebar-card-title">面试安排</h4>
            <div class="schedule-item">
              <el-icon><Clock /></el-icon>
              <span>{{ formatDateTime(interview.scheduled_at) }}</span>
            </div>
            <div class="schedule-item">
              <el-icon><component :is="modeIcon" /></el-icon>
              <span>{{ INTERVIEW_MODE_LABEL[interview.mode] || interview.mode }}</span>
            </div>
            <div v-if="interview.location" class="schedule-item">
              <el-icon><MapLocation /></el-icon>
              <span>{{ interview.location }}</span>
            </div>
            <div class="schedule-item">
              <el-icon><Clock /></el-icon>
              <span>{{ formatDuration(interview.duration_minutes) }}</span>
            </div>
          </div>

          <!-- Meeting entry -->
          <div v-if="interview.meeting_url && interview.status === 'scheduled'" class="sidebar-card">
            <h4 class="sidebar-card-title">会议入口</h4>
            <el-button type="primary" style="width: 100%" @click="openMeeting">
              <el-icon><VideoCamera /></el-icon>
              进入会议
            </el-button>
          </div>

          <!-- Feedback status -->
          <div class="sidebar-card">
            <h4 class="sidebar-card-title">反馈状态</h4>
            <div v-if="hasFeedback" class="feedback-status submitted">
              <el-tag type="success" size="small" effect="light">已提交</el-tag>
              <span class="feedback-rec">{{ RECOMMENDATION_LABEL[feedback!.recommendation] || feedback!.recommendation }}</span>
            </div>
            <div v-else-if="canSubmitFeedback" class="feedback-status pending">
              <el-tag type="warning" size="small" effect="light">待提交</el-tag>
              <span>请在下方填写反馈</span>
            </div>
            <div v-else class="feedback-status">
              <el-tag type="info" size="small" effect="light">面试已取消</el-tag>
            </div>
          </div>

          <!-- Internal note -->
          <div v-if="interview.internal_note" class="sidebar-card">
            <h4 class="sidebar-card-title">内部备注</h4>
            <p class="internal-note">{{ interview.internal_note }}</p>
          </div>

          <!-- Candidate note -->
          <div v-if="interview.candidate_note" class="sidebar-card">
            <h4 class="sidebar-card-title">候选人备注</h4>
            <p class="internal-note">{{ interview.candidate_note }}</p>
          </div>
        </aside>
      </div>
    </template>
  </div>
</template>

<style scoped>
.detail-breadcrumb {
  margin-bottom: 20px;
}

.detail-body {
  display: flex;
  gap: 28px;
  align-items: flex-start;
}

.detail-main {
  flex: 1;
  min-width: 0;
}

.detail-sidebar {
  width: 320px;
  flex-shrink: 0;
  position: sticky;
  top: 80px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.detail-section {
  background: var(--surface-primary);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 20px 24px;
  margin-bottom: 20px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  line-height: 24px;
  color: var(--text-primary);
  margin-bottom: 16px;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px 24px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-label {
  font-size: 13px;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  gap: 4px;
}

.info-value {
  font-size: 14px;
  color: var(--text-primary);
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 4px;
}

.cancel-reason {
  margin-top: 12px;
  padding: 10px 12px;
  background: var(--danger-soft);
  border-radius: var(--radius-sm);
  font-size: 13px;
  color: var(--danger);
  display: flex;
  align-items: center;
  gap: 6px;
}

.sidebar-card {
  background: var(--surface-primary);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 16px 20px;
}

.sidebar-card-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 12px;
}

.schedule-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-secondary);
  padding: 4px 0;
}

.schedule-item .el-icon {
  color: var(--text-muted);
  font-size: 16px;
}

.feedback-status {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-secondary);
}

.feedback-rec {
  font-weight: 500;
  color: var(--text-primary);
}

.internal-note {
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.6;
  white-space: pre-wrap;
}

.material-hint {
  font-size: 13px;
  color: var(--text-muted);
}

.resume-link-wrapper {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: var(--surface-secondary);
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-subtle);
}

.resume-link-wrapper .el-icon {
  color: var(--brand-primary);
  font-size: 18px;
  flex-shrink: 0;
}

.resume-link {
  font-size: 14px;
  font-weight: 500;
  color: var(--brand-primary);
  text-decoration: none;
  transition: color 120ms ease;
}

.resume-link:hover {
  color: var(--brand-hover);
  text-decoration: underline;
}

@media (max-width: 960px) {
  .detail-body {
    flex-direction: column;
  }
  .detail-sidebar {
    width: 100%;
    position: static;
  }
  .info-grid {
    grid-template-columns: 1fr;
  }
}
</style>
