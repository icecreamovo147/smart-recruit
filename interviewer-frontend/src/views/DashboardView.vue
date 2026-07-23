<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Clock, VideoCamera, Phone, OfficeBuilding } from '@element-plus/icons-vue'
import { listMyInterviews } from '@/api/interview'
import type { InterviewSchedule } from '@/types/domain'
import { INTERVIEW_MODE_LABEL, INTERVIEW_STATUS_LABEL } from '@/types/domain'
import { formatShanghaiDateTime, formatShanghaiLongDate } from '@shared/utils/format'

const router = useRouter()
const loading = ref(false)
const interviews = ref<InterviewSchedule[]>([])

const today = new Date()
const todayKey = formatShanghaiDateTime(today, '-', false).slice(0, 10)
const todayStr = formatShanghaiLongDate(today)

const todayInterviews = computed(() => {
  return interviews.value.filter((i) => {
    return formatShanghaiDateTime(i.scheduled_at, '-', false).startsWith(todayKey)
  }).sort((a, b) => new Date(a.scheduled_at).getTime() - new Date(b.scheduled_at).getTime())
})

const nextInterview = computed(() => {
  const now = Date.now()
  return todayInterviews.value.find((i) => new Date(i.scheduled_at).getTime() > now) || null
})

const pendingFeedback = computed(() =>
  interviews.value.filter((i) => i.status === 'completed' && !i.has_feedback),
)

const summaryText = computed(() => {
  const todayCount = todayInterviews.value.length
  const feedbackCount = pendingFeedback.value.length
  const parts = [`今日 ${todayCount} 场`]
  if (feedbackCount > 0) parts.push(`待反馈 ${feedbackCount} 项`)
  return parts.join(' · ')
})

const formatTime = (iso: string): string => {
  return formatShanghaiDateTime(iso, '-', false).slice(11)
}

const formatDateTime = (iso: string): string => {
  const value = formatShanghaiDateTime(iso, '-', false)
  return `${Number(value.slice(5, 7))}/${Number(value.slice(8, 10))} ${value.slice(11)}`
}

const modeIcon = (mode: string) => {
  if (mode === 'video') return VideoCamera
  if (mode === 'phone') return Phone
  return OfficeBuilding
}

const openMeeting = (url: string) => {
  window.open(url, '_blank')
}

const getCountdown = (iso: string): string => {
  const diff = new Date(iso).getTime() - Date.now()
  if (diff <= 0) return '已开始'
  const hours = Math.floor(diff / 3600000)
  const mins = Math.floor((diff % 3600000) / 60000)
  if (hours > 0) return `${hours} 小时 ${mins} 分钟后`
  return `${mins} 分钟后`
}

const loadData = async () => {
  loading.value = true
  try {
    const data = await listMyInterviews()
    interviews.value = data.list || []
  } catch {
    // Handled by request interceptor
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
</script>

<template>
  <div class="dashboard">
    <div class="dashboard-header">
      <h2 class="dashboard-title">今天的面试</h2>
      <p class="dashboard-date">{{ todayStr }}</p>
    </div>

    <p class="dashboard-summary">{{ summaryText }}</p>

    <div class="dashboard-body">
      <div class="dashboard-main">
        <!-- Next interview highlight -->
        <div v-if="nextInterview" class="next-interview">
          <div class="next-label">下一场面试</div>
          <div class="next-time">{{ formatTime(nextInterview.scheduled_at) }}</div>
          <div class="next-countdown">{{ getCountdown(nextInterview.scheduled_at) }}</div>
          <div class="next-details">
            <p class="next-candidate">{{ nextInterview.candidate_name }}</p>
            <p class="next-job">{{ nextInterview.job_title }}</p>
            <p class="next-meta">
              <el-icon><component :is="modeIcon(nextInterview.mode)" /></el-icon>
              {{ INTERVIEW_MODE_LABEL[nextInterview.mode] || nextInterview.mode }}
              · {{ nextInterview.duration_minutes }} 分钟
              · 第 {{ nextInterview.round_no }} 轮
            </p>
          </div>
          <div class="next-actions">
            <el-button type="primary" @click="router.push(`/interviews/${nextInterview.interview_id}`)">查看详情</el-button>
            <el-button
              v-if="nextInterview.meeting_url"
              type="primary"
              plain
              @click="openMeeting(nextInterview.meeting_url)"
            >进入会议</el-button>
          </div>
        </div>

        <el-empty v-else-if="!loading && todayInterviews.length === 0" description="今天暂无面试安排" />

        <!-- Today timeline -->
        <div v-if="todayInterviews.length > 0" class="today-timeline">
          <h3 class="section-title">今日时间线</h3>
          <div class="timeline-list">
            <div
              v-for="item in todayInterviews"
              :key="item.interview_id"
              class="timeline-row"
              @click="router.push(`/interviews/${item.interview_id}`)"
            >
              <div class="timeline-time">{{ formatTime(item.scheduled_at) }}</div>
              <div class="timeline-info">
                <span class="timeline-candidate">{{ item.candidate_name }}</span>
                <span class="timeline-job">{{ item.job_title }}</span>
              </div>
              <div class="timeline-status">
                <el-tag
                  :type="item.status === 'scheduled' ? 'primary' : item.status === 'completed' ? 'success' : 'info'"
                  size="small"
                  effect="light"
                >
                  {{ INTERVIEW_STATUS_LABEL[item.status] || item.status }}
                </el-tag>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Pending feedback sidebar -->
      <aside class="feedback-panel">
        <h3 class="section-title">待反馈</h3>
        <div v-if="pendingFeedback.length > 0" class="feedback-list">
          <div
            v-for="item in pendingFeedback"
            :key="item.interview_id"
            class="feedback-row"
          >
            <div class="feedback-info">
              <span class="feedback-candidate">{{ item.candidate_name }}</span>
              <span class="feedback-date">{{ formatDateTime(item.scheduled_at) }}</span>
            </div>
            <el-button
              type="primary"
              size="small"
              @click="router.push(`/interviews/${item.interview_id}`)"
            >填写反馈</el-button>
          </div>
        </div>
        <p v-else class="feedback-empty">所有反馈已提交</p>
      </aside>
    </div>

    <el-skeleton v-if="loading" :rows="6" animated />
  </div>
</template>

<style scoped>
.dashboard-header {
  margin-bottom: 8px;
}

.dashboard-title {
  font-size: 24px;
  font-weight: 600;
  line-height: 32px;
  color: var(--text-primary);
}

.dashboard-date {
  font-size: 14px;
  color: var(--text-muted);
  margin-top: 4px;
}

.dashboard-summary {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 24px;
}

.dashboard-body {
  display: flex;
  gap: 28px;
}

.dashboard-main {
  flex: 1;
  min-width: 0;
}

.next-interview {
  background: var(--surface-primary);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 24px;
  margin-bottom: 28px;
}

.next-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-muted);
  margin-bottom: 8px;
}

.next-time {
  font-size: 28px;
  font-weight: 600;
  line-height: 36px;
  color: var(--brand-primary);
}

.next-countdown {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 16px;
}

.next-candidate {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.next-job {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 4px;
}

.next-meta {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: var(--text-muted);
}

.next-actions {
  margin-top: 16px;
  display: flex;
  gap: 8px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  line-height: 24px;
  color: var(--text-primary);
  margin-bottom: 12px;
}

.timeline-list {
  display: flex;
  flex-direction: column;
}

.timeline-row {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 0;
  border-bottom: 1px solid var(--border-subtle);
  cursor: pointer;
  transition: background-color 120ms ease;
}
.timeline-row:hover {
  background: var(--surface-secondary);
}

.timeline-time {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  width: 52px;
  flex-shrink: 0;
}

.timeline-info {
  flex: 1;
  min-width: 0;
}

.timeline-candidate {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin-right: 8px;
}

.timeline-job {
  font-size: 13px;
  color: var(--text-muted);
}

.feedback-panel {
  width: 320px;
  flex-shrink: 0;
}

.feedback-list {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.feedback-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 0;
  border-bottom: 1px solid var(--border-subtle);
}

.feedback-info {
  display: flex;
  flex-direction: column;
}

.feedback-candidate {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

.feedback-date {
  font-size: 13px;
  color: var(--text-muted);
}

.feedback-empty {
  font-size: 14px;
  color: var(--text-muted);
}

@media (max-width: 1180px) {
  .dashboard-body {
    flex-direction: column;
  }
  .feedback-panel {
    width: 100%;
  }
}
</style>
