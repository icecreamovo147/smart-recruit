<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { scheduleInterview } from '@/api/interview'
import { listJobApplications } from '@/api/application'
import type { Application, StaffUserInfo } from '@/types/domain'
import InterviewerPickerDialog from '@/components/business/InterviewerPickerDialog.vue'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const applications = ref<Application[]>([])
const appLoading = ref(false)
const pickerVisible = ref(false)
const selectedInterviewerName = ref('')

const jobId = Number(route.query.job_id) || 0

const form = reactive({
  application_id: Number(route.query.application_id) || 0,
  interviewer_id: 0,
  round_no: 1,
  title: '',
  mode: 'video',
  meeting_url: '',
  location: '',
  duration_minutes: 60,
  candidate_note: '',
  internal_note: '',
  scheduled_at: '',
})

const loadApplications = async () => {
  if (!jobId) return // Need a job_id to scope the application list
  appLoading.value = true
  try {
    const data = await listJobApplications(jobId, { page: 1, page_size: 100 })
    applications.value = (data.list || []).filter((app) => app.is_current === 1)
  } catch {
    // Silently fail
  } finally {
    appLoading.value = false
  }
}

const interviewerDisplay = computed(() => {
  if (!form.interviewer_id) return ''
  return selectedInterviewerName.value || `面试官 #${form.interviewer_id}`
})

const handleInterviewerSelect = (user: StaffUserInfo) => {
  form.interviewer_id = Number(user.user_id)
  selectedInterviewerName.value = user.username
}

const handleSubmit = async () => {
  if (!form.application_id) {
    ElMessage.warning('请选择投递记录')
    return
  }
  if (!form.interviewer_id) {
    ElMessage.warning('请选择面试官')
    return
  }
  loading.value = true
  try {
    const data = await scheduleInterview({
      application_id: Number(form.application_id),
      interviewer_id: Number(form.interviewer_id),
      round_no: Number(form.round_no),
      title: form.title || undefined,
      mode: form.mode,
      meeting_url: form.meeting_url || undefined,
      location: form.location || undefined,
      duration_minutes: Number(form.duration_minutes),
      candidate_note: form.candidate_note || undefined,
      internal_note: form.internal_note || undefined,
      scheduled_at: form.scheduled_at ? new Date(form.scheduled_at).toISOString() : undefined,
    })
    ElMessage.success('面试安排成功')
    router.push('/hr/interviews')
  } catch (error: unknown) {
    ElMessage.error(error instanceof Error ? error.message : '安排面试失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadApplications()
})
</script>

<template>
  <div class="interview-schedule-view">
    <h2>安排面试</h2>
    <el-card class="mt-4" style="max-width: 720px">
      <el-form :model="form" label-width="120px">
        <el-form-item label="投递记录" required>
          <el-select v-model="form.application_id" placeholder="请选择投递记录" filterable style="width: 100%">
            <el-option
              v-for="app in applications"
              :key="app.application_id"
              :label="(app.real_name || '候选人' + app.user_id) + ' - ' + app.job_title"
              :value="app.application_id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="面试官" required>
          <el-input
            :model-value="interviewerDisplay"
            readonly
            placeholder="请选择面试官"
            style="width: 100%"
            @click="pickerVisible = true"
          >
            <template #append>
              <el-button @click.stop="pickerVisible = true">选择</el-button>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item label="面试轮次">
          <el-input-number v-model="form.round_no" :min="1" :max="10" />
        </el-form-item>

        <el-form-item label="面试标题">
          <el-input v-model="form.title" placeholder="如：初试、复试、终面（留空自动生成）" />
        </el-form-item>

        <el-form-item label="面试模式">
          <el-radio-group v-model="form.mode">
            <el-radio value="video">视频面试</el-radio>
            <el-radio value="phone">电话面试</el-radio>
            <el-radio value="onsite">现场面试</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="会议链接">
          <el-input v-model="form.meeting_url" placeholder="视频会议链接（可选）" />
        </el-form-item>

        <el-form-item label="面试地点">
          <el-input v-model="form.location" placeholder="线下地址（可选）" />
        </el-form-item>

        <el-form-item label="面试时长">
          <el-input-number v-model="form.duration_minutes" :min="15" :max="480" :step="15" />
          <span class="ml-2">分钟</span>
        </el-form-item>

        <el-form-item label="面试时间">
          <el-date-picker
            v-model="form.scheduled_at"
            type="datetime"
            placeholder="选择面试时间"
            value-format="YYYY-MM-DDTHH:mm:ss"
            style="width: 100%"
          />
        </el-form-item>

        <el-form-item label="候选人备注">
          <el-input v-model="form.candidate_note" type="textarea" :rows="3" placeholder="给候选人的注意事项（候选人在面试详情中可见）" />
        </el-form-item>

        <el-form-item label="内部备注">
          <el-input v-model="form.internal_note" type="textarea" :rows="3" placeholder="内部备注（仅 HR 可见）" />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" :loading="loading" @click="handleSubmit">安排面试</el-button>
          <el-button @click="router.back()">取消</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <InterviewerPickerDialog
      v-model:visible="pickerVisible"
      :selected-id="form.interviewer_id"
      @select="handleInterviewerSelect"
    />
  </div>
</template>

<style scoped>
.interview-schedule-view {
  padding: 20px;
  max-width: 800px;
  margin: 0 auto;
}
.mt-4 {
  margin-top: 16px;
}
.ml-2 {
  margin-left: 8px;
}
</style>
