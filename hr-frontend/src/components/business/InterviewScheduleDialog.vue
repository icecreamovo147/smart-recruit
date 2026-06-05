<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { scheduleInterview } from '@/api/interview'
import type { StaffUserInfo } from '@/types/domain'
import InterviewerPickerDialog from '@/components/business/InterviewerPickerDialog.vue'

const props = defineProps<{
  visible: boolean
  applicationId: number
  jobTitle?: string
  candidateName?: string
}>()

const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
  (e: 'success'): void
}>()

const loading = ref(false)
const pickerVisible = ref(false)
const selectedInterviewerName = ref('')
const form = reactive({
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

const resetForm = () => {
  form.interviewer_id = 0
  form.round_no = 1
  form.title = ''
  form.mode = 'video'
  form.meeting_url = ''
  form.location = ''
  form.duration_minutes = 60
  form.candidate_note = ''
  form.internal_note = ''
  form.scheduled_at = ''
  selectedInterviewerName.value = ''
}

const candidateSummary = computed(() => {
  return [props.candidateName, props.jobTitle].filter((item) => item && item.trim()).join(' - ')
})

const interviewerDisplay = computed(() => {
  if (!form.interviewer_id) return ''
  return selectedInterviewerName.value || `面试官 #${form.interviewer_id}`
})

const handleInterviewerSelect = (user: StaffUserInfo) => {
  form.interviewer_id = Number(user.user_id)
  selectedInterviewerName.value = user.username
}

const handleSubmit = async () => {
  if (!form.interviewer_id) {
    ElMessage.warning('请选择面试官')
    return
  }
  loading.value = true
  try {
    await scheduleInterview({
      application_id: Number(props.applicationId),
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
    emit('success')
    emit('update:visible', false)
    resetForm()
  } catch (error: unknown) {
    ElMessage.error(error instanceof Error ? error.message : '安排面试失败')
  } finally {
    loading.value = false
  }
}

const handleClose = () => {
  emit('update:visible', false)
}
</script>

<template>
  <el-dialog
    :model-value="props.visible"
    title="安排面试"
    width="600px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form :model="form" label-width="110px">
      <el-alert v-if="candidateSummary" :title="candidateSummary" type="info" :closable="false" class="mb-4" />

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
        <el-input v-model="form.title" placeholder="如：初试、复试（留空自动生成）" />
      </el-form-item>

      <el-form-item label="面试模式">
        <el-radio-group v-model="form.mode">
          <el-radio value="video">视频</el-radio>
          <el-radio value="phone">电话</el-radio>
          <el-radio value="onsite">现场</el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="会议链接">
        <el-input v-model="form.meeting_url" placeholder="可选" />
      </el-form-item>

      <el-form-item label="面试地点">
        <el-input v-model="form.location" placeholder="可选" />
      </el-form-item>

      <el-form-item label="时长（分钟）">
        <el-input-number v-model="form.duration_minutes" :min="15" :max="480" :step="15" />
      </el-form-item>

      <el-form-item label="面试时间">
        <el-date-picker
          v-model="form.scheduled_at"
          type="datetime"
          placeholder="选择时间"
          value-format="YYYY-MM-DDTHH:mm:ss"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item label="候选人备注">
        <el-input v-model="form.candidate_note" type="textarea" :rows="2" placeholder="注意事项（候选人可见）" />
      </el-form-item>

      <el-form-item label="内部备注">
        <el-input v-model="form.internal_note" type="textarea" :rows="2" placeholder="仅 HR 可见" />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" :loading="loading" @click="handleSubmit">安排面试</el-button>
    </template>
  </el-dialog>

  <InterviewerPickerDialog
    v-model:visible="pickerVisible"
    :selected-id="form.interviewer_id"
    @select="handleInterviewerSelect"
  />
</template>

<style scoped>
.mb-4 {
  margin-bottom: 16px;
}
</style>
