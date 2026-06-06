<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import { submitFeedback, getFeedback } from '@/api/interview'
import type { InterviewSchedule, InterviewFeedback, FeedbackPayload } from '@/types/domain'
import { RECOMMENDATION_OPTIONS, DIMENSION_LABELS } from '@/types/domain'

const props = defineProps<{
  interview: InterviewSchedule
}>()

const emit = defineEmits<{
  submitted: [feedback: InterviewFeedback]
}>()

const submitting = ref(false)
const formRef = ref()

const DIMENSION_KEYS = Object.keys(DIMENSION_LABELS)

const form = reactive({
  recommendation: '',
  score: 5,
  dimensions: Object.fromEntries(DIMENSION_KEYS.map((k) => [k, 5])) as Record<string, number>,
  comments: '',
})

const rules = {
  recommendation: [{ required: true, message: '请选择推荐结论', trigger: 'change' }],
  score: [{ required: true, message: '请输入综合评分', trigger: 'blur' }],
  comments: [{ required: true, message: '请填写评语', trigger: 'blur' }],
}

const isDirty = computed(() => {
  return form.recommendation !== '' || form.comments !== '' || form.score !== 5 ||
    DIMENSION_KEYS.some((k) => form.dimensions[k] !== 5)
})

const buildPayload = (): FeedbackPayload => ({
  application_id: props.interview.application_id,
  recommendation: form.recommendation,
  score: form.score,
  dimension_scores_json: JSON.stringify(form.dimensions),
  comments: form.comments,
})

const handleSubmit = async () => {
  if (!props.interview.application_id) {
    ElMessage.error('面试数据不完整（缺少 application_id），请刷新页面后重试')
    return
  }
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  try {
    await ElMessageBox.confirm(
      '确认提交后无法修改，是否继续？',
      '提交确认',
      { confirmButtonText: '确认提交', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return
  }

  submitting.value = true
  try {
    await submitFeedback(props.interview.interview_id, buildPayload())
    ElMessage.success('反馈已提交')
    // Refetch feedback data since submitFeedback returns CommonResponse (no entity)
    const fb = await getFeedback(props.interview.interview_id)
    if (fb) {
      emit('submitted', fb)
    }
  } catch {
    // Handled by request interceptor
  } finally {
    submitting.value = false
  }
}

const handleBeforeUnload = (e: BeforeUnloadEvent) => {
  if (isDirty.value) {
    e.preventDefault()
  }
}

onMounted(() => {
  window.addEventListener('beforeunload', handleBeforeUnload)
})

onBeforeUnmount(() => {
  window.removeEventListener('beforeunload', handleBeforeUnload)
})
</script>

<template>
  <el-form
    ref="formRef"
    :model="form"
    :rules="rules"
    label-position="top"
    class="feedback-form"
  >
    <!-- Recommendation -->
    <el-form-item label="推荐结论" prop="recommendation">
      <el-radio-group v-model="form.recommendation">
        <el-radio-button
          v-for="opt in RECOMMENDATION_OPTIONS"
          :key="opt.value"
          :value="opt.value"
        >{{ opt.label }}</el-radio-button>
      </el-radio-group>
    </el-form-item>

    <!-- Overall score -->
    <el-form-item label="综合评分（1-10）" prop="score">
      <div class="score-row">
        <el-slider v-model="form.score" :min="1" :max="10" :step="1" style="flex: 1" />
        <el-input-number v-model="form.score" :min="1" :max="10" :step="1" size="small" style="width: 100px" />
      </div>
    </el-form-item>

    <!-- Dimension scores -->
    <el-form-item label="维度评分（1-10）">
      <div class="dimensions-grid">
        <div v-for="key in DIMENSION_KEYS" :key="key" class="dimension-item">
          <label class="dimension-label">{{ DIMENSION_LABELS[key] }}</label>
          <div class="dimension-input">
            <el-slider v-model="form.dimensions[key]" :min="1" :max="10" :step="1" style="flex: 1" />
            <el-input-number v-model="form.dimensions[key]" :min="1" :max="10" :step="1" size="small" style="width: 100px" />
          </div>
        </div>
      </div>
    </el-form-item>

    <!-- Comments -->
    <el-form-item label="评语" prop="comments">
      <el-input
        v-model="form.comments"
        type="textarea"
        :rows="5"
        placeholder="请描述候选人的表现、优缺点及建议"
        maxlength="2000"
        show-word-limit
      />
    </el-form-item>

    <!-- Submit -->
    <el-form-item>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">
        提交反馈
      </el-button>
    </el-form-item>
  </el-form>
</template>

<style scoped>
.feedback-form {
  max-width: 720px;
}

.score-row {
  display: flex;
  align-items: center;
  gap: 16px;
  width: 100%;
}

.dimensions-grid {
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
}

.dimension-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.dimension-label {
  font-size: 13px;
  color: var(--text-secondary);
  font-weight: 500;
}

.dimension-input {
  display: flex;
  align-items: center;
  gap: 16px;
}

:deep(.el-radio-button__inner) {
  font-size: 13px;
}

@media (max-width: 640px) {
  .feedback-form {
    max-width: 100%;
  }
}
</style>
