<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { createOffer } from '@/api/offer'

const props = defineProps<{
  visible: boolean
  applicationId: number
  jobTitle?: string
  salaryRange?: string
  workLocation?: string
  candidateName?: string
}>()

const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
  (e: 'success'): void
}>()

const loading = ref(false)
const form = reactive({
  title: '',
  salary_range: '',
  level: '',
  work_location: '',
  start_date: '',
  expires_at: '',
  terms_json: '',
})

const resetForm = () => {
  form.title = props.jobTitle || ''
  form.salary_range = props.salaryRange || ''
  form.level = ''
  form.work_location = props.workLocation || ''
  form.start_date = ''
  form.expires_at = ''
  form.terms_json = ''
}

const handleSubmit = async () => {
  if (!form.title) {
    ElMessage.warning('请输入Offer职位名称')
    return
  }
  loading.value = true
  try {
    const data: Record<string, unknown> = {
      application_id: Number(props.applicationId),
      title: form.title,
    }
    if (form.salary_range) data.salary_range = form.salary_range
    if (form.level) data.level = form.level
    if (form.work_location) data.work_location = form.work_location
    if (form.start_date) data.start_date = form.start_date
    if (form.expires_at) {
      data.expires_at = new Date(form.expires_at).toISOString()
    }
    if (form.terms_json) data.terms_json = form.terms_json

    await createOffer(data as any)
    ElMessage.success('Offer创建成功')
    resetForm()
    emit('success')
    emit('update:visible', false)
  } catch (error: unknown) {
    const msg = error instanceof Error ? error.message : '创建Offer失败'
    ElMessage.error(msg)
  } finally {
    loading.value = false
  }
}

const dialogVisible = ref(false)
</script>

<template>
  <el-dialog
    :model-value="props.visible"
    @update:model-value="(val: boolean) => emit('update:visible', val)"
    title="创建Offer"
    width="600px"
    :close-on-click-modal="false"
    @open="resetForm"
  >
    <el-form label-width="120px" label-position="top" size="default">
      <el-form-item label="Offer职位名称" required>
        <el-input v-model="form.title" disabled placeholder="从岗位信息自动带入" />
      </el-form-item>
      <el-row :gutter="16">
        <el-col :span="12">
          <el-form-item label="薪资范围">
            <el-input v-model="form.salary_range" disabled placeholder="从岗位信息自动带入" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="职级">
            <el-input v-model="form.level" placeholder="例如：P6" />
          </el-form-item>
        </el-col>
      </el-row>
      <el-row :gutter="16">
        <el-col :span="12">
          <el-form-item label="工作地点">
            <el-input v-model="form.work_location" disabled placeholder="从岗位信息自动带入" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="预计入职日期">
            <el-date-picker
              v-model="form.start_date"
              type="date"
              placeholder="选择入职日期"
              format="YYYY-MM-DD"
              value-format="YYYY-MM-DD"
              style="width: 100%"
            />
          </el-form-item>
        </el-col>
      </el-row>
      <el-row :gutter="16">
        <el-col :span="12">
          <el-form-item label="Offer过期时间">
            <el-date-picker
              v-model="form.expires_at"
              type="datetime"
              placeholder="选择过期时间"
              format="YYYY-MM-DD HH:mm"
              value-format="YYYY-MM-DDTHH:mm:ssZ"
              style="width: 100%"
            />
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item label="Offer条款（JSON）">
        <el-input
          v-model="form.terms_json"
          type="textarea"
          :rows="4"
          placeholder='例如：{"bonus":"2个月年终奖","stock":"1000期权","benefits":"五险一金、补充医疗保险"}'
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="emit('update:visible', false)">取消</el-button>
      <el-button type="primary" :loading="loading" @click="handleSubmit">创建Offer</el-button>
    </template>
  </el-dialog>
</template>
