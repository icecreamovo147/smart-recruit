<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listMyInterviews, updateInterview, cancelInterview, submitFeedback, getFeedback } from '@/api/interview'
import type { InterviewSchedule } from '@/types/domain'
import { getHRStatusLabel, getStatusType } from '@/types/domain'

const router = useRouter()
const loading = ref(false)
const errorMessage = ref('')
const list = ref<InterviewSchedule[]>([])
const filterStatus = ref('')
const detailVisible = ref(false)
const selectedInterview = ref<InterviewSchedule | null>(null)
const feedbackForm = reactive({
  recommendation: '',
  score: 5,
  comments: '',
  dimension_scores_json: '{}',
})
const feedbackLoading = ref(false)
const existingFeedback = ref<{ recommendation: string; score: number; comments: string; submitted_at: string } | null>(null)

const load = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    const data = await listMyInterviews(filterStatus.value || undefined)
    list.value = data.list || []
  } catch (error: unknown) {
    errorMessage.value = error instanceof Error ? error.message : '面试列表加载失败'
  } finally {
    loading.value = false
  }
}

const formatDateTime = (value: string): string => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (num: number): string => String(num).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

const statusLabel = (row: InterviewSchedule): string => {
  const labels: Record<string, string> = {
    pending: '待安排',
    scheduled: '已安排',
    completed: '已完成',
    cancelled: '已取消',
  }
  return labels[row.status] || row.status
}

const statusType = (row: InterviewSchedule): string => {
  const types: Record<string, string> = {
    pending: 'info',
    scheduled: 'warning',
    completed: 'success',
    cancelled: 'danger',
  }
  return types[row.status] || 'info'
}

const modeLabel = (mode: string): string => {
  const labels: Record<string, string> = {
    video: '视频',
    phone: '电话',
    onsite: '现场',
  }
  return labels[mode] || mode || '视频'
}

const openDetail = async (interview: InterviewSchedule) => {
  selectedInterview.value = interview
  detailVisible.value = true
  existingFeedback.value = null
  feedbackForm.recommendation = ''
  feedbackForm.score = 5
  feedbackForm.comments = ''
  feedbackForm.dimension_scores_json = '{}'

  // Load existing feedback if any
  try {
    const fbData = await getFeedback(interview.interview_id)
    if (fbData.feedback) {
      existingFeedback.value = {
        recommendation: fbData.feedback.recommendation,
        score: fbData.feedback.score,
        comments: fbData.feedback.comments,
        submitted_at: fbData.feedback.submitted_at,
      }
    }
  } catch {
    // No feedback yet
  }
}

const handleCancel = async (interview: InterviewSchedule) => {
  try {
    const { value } = await ElMessageBox.prompt('请输入取消原因', '取消面试', {
      confirmButtonText: '确认取消',
      cancelButtonText: '返回',
      inputPattern: /.{1,}/,
      inputErrorMessage: '请填写取消原因',
    })
    await cancelInterview(interview.interview_id, value || '')
    ElMessage.success('面试已取消')
    await load()
  } catch {
    // User cancelled
  }
}

const handleFeedbackSubmit = async (interviewId: number) => {
  if (!feedbackForm.recommendation) {
    ElMessage.warning('请选择推荐结论')
    return
  }
  feedbackLoading.value = true
  try {
    await submitFeedback(interviewId, {
      application_id: selectedInterview.value!.application_id,
      recommendation: feedbackForm.recommendation,
      score: feedbackForm.score,
      dimension_scores_json: feedbackForm.dimension_scores_json,
      comments: feedbackForm.comments,
    })
    ElMessage.success('面试反馈已提交')
    existingFeedback.value = {
      recommendation: feedbackForm.recommendation,
      score: feedbackForm.score,
      comments: feedbackForm.comments,
      submitted_at: new Date().toISOString(),
    }
    await load()
  } catch (error: unknown) {
    ElMessage.error(error instanceof Error ? error.message : '提交反馈失败')
  } finally {
    feedbackLoading.value = false
  }
}

const goToScheduling = () => {
  // Navigate to jobs list — recruiters pick a job → view applications → schedule interview.
  // The standalone schedule page needs a job_id to scope the application list.
  router.push('/hr/jobs')
}

onMounted(load)
</script>

<template>
  <div class="interview-task-view">
    <!-- Header -->
    <div class="header">
      <h2>我的面试任务</h2>
      <div class="header-actions">
        <el-select v-model="filterStatus" placeholder="全部状态" clearable size="default" style="width: 140px" @change="load">
          <el-option label="全部状态" value="" />
          <el-option label="待安排" value="pending" />
          <el-option label="已安排" value="scheduled" />
          <el-option label="已完成" value="completed" />
          <el-option label="已取消" value="cancelled" />
        </el-select>
        <el-button type="primary" @click="goToScheduling">安排面试</el-button>
      </div>
    </div>

    <el-alert v-if="errorMessage" :title="errorMessage" type="error" show-icon closable class="mb-4" />

    <!-- Interview list -->
    <el-card v-loading="loading" class="mt-4">
      <el-empty v-if="!loading && list.length === 0" description="暂无面试任务" />
      <el-table v-else :data="list" stripe style="width: 100%">
        <el-table-column prop="interview_id" label="ID" width="70" />
        <el-table-column prop="job_title" label="岗位" min-width="160" />
        <el-table-column prop="candidate_name" label="候选人" width="120" />
        <el-table-column prop="round_no" label="轮次" width="70">
          <template #default="{ row }">
            第{{ row.round_no || 1 }}轮
          </template>
        </el-table-column>
        <el-table-column prop="title" label="面试名称" min-width="140" />
        <el-table-column prop="mode" label="模式" width="80">
          <template #default="{ row }">
            <el-tag :type="row.mode === 'onsite' ? 'primary' : row.mode === 'phone' ? 'success' : 'warning'" size="small">
              {{ modeLabel(row.mode) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="scheduled_at" label="面试时间" width="170">
          <template #default="{ row }">
            {{ formatDateTime(row.scheduled_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="duration_minutes" label="时长" width="80">
          <template #default="{ row }">
            {{ row.duration_minutes ? row.duration_minutes + '分钟' : '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusType(row)" size="small">
              {{ statusLabel(row) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openDetail(row)">详情</el-button>
            <el-button v-if="row.status === 'scheduled' || row.status === 'pending'" size="small" type="danger" plain @click="handleCancel(row)">取消</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Detail dialog -->
    <el-dialog v-model="detailVisible" title="面试详情" width="640px" :close-on-click-modal="false">
      <template v-if="selectedInterview">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="岗位" :span="2">{{ selectedInterview.job_title }}</el-descriptions-item>
          <el-descriptions-item label="候选人">{{ selectedInterview.candidate_name }}</el-descriptions-item>
          <el-descriptions-item label="面试官">{{ selectedInterview.interviewer_name }}</el-descriptions-item>
          <el-descriptions-item label="面试名称">{{ selectedInterview.title }}</el-descriptions-item>
          <el-descriptions-item label="轮次">第{{ selectedInterview.round_no || 1 }}轮</el-descriptions-item>
          <el-descriptions-item label="模式">{{ modeLabel(selectedInterview.mode) }}</el-descriptions-item>
          <el-descriptions-item label="面试时间" :span="2">{{ formatDateTime(selectedInterview.scheduled_at) }}</el-descriptions-item>
          <el-descriptions-item label="时长">{{ selectedInterview.duration_minutes ? selectedInterview.duration_minutes + '分钟' : '-' }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="statusType(selectedInterview)" size="small">{{ statusLabel(selectedInterview) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item v-if="selectedInterview.meeting_url" label="会议链接" :span="2">
            <el-link :href="selectedInterview.meeting_url" target="_blank">{{ selectedInterview.meeting_url }}</el-link>
          </el-descriptions-item>
          <el-descriptions-item v-if="selectedInterview.location" label="地点" :span="2">{{ selectedInterview.location }}</el-descriptions-item>
          <el-descriptions-item v-if="selectedInterview.candidate_note" label="注意事项" :span="2">{{ selectedInterview.candidate_note }}</el-descriptions-item>
          <el-descriptions-item v-if="selectedInterview.internal_note" label="内部备注" :span="2">{{ selectedInterview.internal_note }}</el-descriptions-item>
          <el-descriptions-item v-if="selectedInterview.cancel_reason" label="取消原因" :span="2">{{ selectedInterview.cancel_reason }}</el-descriptions-item>
        </el-descriptions>

        <!-- Feedback section -->
        <div v-if="selectedInterview.status === 'scheduled' || selectedInterview.status === 'completed'" class="mt-4">
          <h3 class="mb-2">面试反馈</h3>
          <div v-if="existingFeedback" class="existing-feedback">
            <el-alert :title="'已提交反馈 (' + formatDateTime(existingFeedback.submitted_at) + ')'" type="success" show-icon :closable="false" />
            <p class="mt-2"><strong>推荐结论：</strong>{{ existingFeedback.recommendation === 'positive' ? '推荐通过' : existingFeedback.recommendation === 'negative' ? '不推荐' : '待定' }}</p>
            <p><strong>评分：</strong>{{ existingFeedback.score }}/10</p>
            <p v-if="existingFeedback.comments"><strong>评语：</strong>{{ existingFeedback.comments }}</p>
          </div>
          <div v-else-if="selectedInterview.status === 'scheduled'">
            <el-form :model="feedbackForm" label-width="100px">
              <el-form-item label="推荐结论" required>
                <el-radio-group v-model="feedbackForm.recommendation">
                  <el-radio value="positive">推荐通过</el-radio>
                  <el-radio value="negative">不推荐</el-radio>
                  <el-radio value="pending">待定</el-radio>
                </el-radio-group>
              </el-form-item>
              <el-form-item label="评分">
                <el-rate v-model="feedbackForm.score" :max="10" show-score allow-half />
              </el-form-item>
              <el-form-item label="面试评语">
                <el-input v-model="feedbackForm.comments" type="textarea" :rows="4" placeholder="请填写面试评语" />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" :loading="feedbackLoading" @click="handleFeedbackSubmit(selectedInterview.interview_id)">
                  提交反馈
                </el-button>
              </el-form-item>
            </el-form>
          </div>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.interview-task-view {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.header h2 {
  margin: 0;
}
.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}
.mb-2 {
  margin-bottom: 8px;
}
.mb-4 {
  margin-bottom: 16px;
}
.mt-2 {
  margin-top: 8px;
}
.mt-4 {
  margin-top: 16px;
}
.existing-feedback {
  background: #f0f9f0;
  padding: 16px;
  border-radius: 8px;
}
</style>
