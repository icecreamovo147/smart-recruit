<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Document, UploadFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getProfile, updateProfile } from '@/api/profile'
import { getResume } from '@/api/resume'
import RichTextEditor from '@/components/RichTextEditor.vue'
import type { Profile, ResumeInfo } from '@/types/domain'

interface ProfileForm {
  real_name: string
  phone: string
  education: string
  school: string
  work_experience: string
  skills: string[]
}

const router = useRouter()
const loading = ref(false)
const saving = ref(false)
const resumeLoading = ref(false)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const formRef = ref<any>(null)
const currentResume = ref<ResumeInfo | null>(null)
const form = reactive<ProfileForm>({
  real_name: '',
  phone: '',
  education: '',
  school: '',
  work_experience: '',
  skills: [],
})
const profileComplete = ref(false)
const skillInput = ref('')
const rules = {
  real_name: [{ required: true, message: '请输入真实姓名', trigger: 'blur' }],
  phone: [
    { required: true, message: '请输入联系电话', trigger: 'blur' },
    { pattern: /^1\d{10}$/, message: '请输入有效的手机号', trigger: 'blur' },
  ],
  education: [{ required: true, message: '请选择学历', trigger: 'change' }],
  school: [{ required: true, message: '请输入毕业院校', trigger: 'blur' }],
  work_experience: [
    { required: true, message: '请填写经历', trigger: 'blur' },
    { min: 20, message: '至少填写 20 个字符', trigger: 'blur' },
  ],
}

const load = async () => {
  loading.value = true
  try {
    const data: Profile = await getProfile()
    form.real_name = data.real_name || data.realName || ''
    form.phone = data.phone || ''
    form.education = data.education || ''
    form.school = data.school || ''
    form.work_experience = data.work_experience || data.workExperience || ''
    form.skills = Array.isArray(data.skills)
      ? data.skills
      : (String(data.skills || '')).split(',').map((item: string) => item.trim()).filter(Boolean)
    profileComplete.value = Boolean(data.is_complete)
  } finally {
    loading.value = false
  }
}

const loadResume = async () => {
  resumeLoading.value = true
  try {
    const data = await getResume()
    currentResume.value = data.resume || null
  } finally {
    resumeLoading.value = false
  }
}

const addSkill = () => {
  const value = skillInput.value.trim()
  if (!value || form.skills.includes(value)) return
  form.skills.push(value)
  skillInput.value = ''
}

const removeSkill = (skill: string) => {
  form.skills = form.skills.filter((item) => item !== skill)
}

const save = async () => {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch {
    return
  }
  saving.value = true
  try {
    const payload: Profile = {
      ...form,
      skills: form.skills.join(','),
    }
    await updateProfile(payload)
    ElMessage.success('保存成功')
    await load()
  } finally {
    saving.value = false
  }
}

const formatFileSize = (value: number): string => {
  if (!value) return '0 KB'
  if (value >= 1024 * 1024) return `${(value / 1024 / 1024).toFixed(1)} MB`
  return `${Math.ceil(value / 1024)} KB`
}

const formatUploadedAt = (value: string): string => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { hour12: false })
}

onMounted(() => {
  load()
  loadResume()
})
</script>

<template>
  <section>
    <div class="page-header">
      <div>
        <h1 class="page-title">个人档案</h1>
        <p class="page-subtitle">完善档案后才能投递岗位。</p>
      </div>
      <el-tag :type="profileComplete ? 'success' : 'warning'" effect="dark">
        {{ profileComplete ? '档案已完善' : '档案未完善，无法投递岗位' }}
      </el-tag>
    </div>
    <div class="content-surface" v-loading="loading">
      <el-form ref="formRef" label-position="top" :model="form" :rules="rules">
        <el-row :gutter="16">
          <el-col :xs="24" :sm="12"><el-form-item label="真实姓名" prop="real_name"><el-input v-model="form.real_name" /></el-form-item></el-col>
          <el-col :xs="24" :sm="12"><el-form-item label="联系电话" prop="phone"><el-input v-model="form.phone" /></el-form-item></el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item label="最高学历" prop="education">
              <el-select v-model="form.education" placeholder="选择学历" style="width: 100%">
                <el-option label="高中" value="高中" />
                <el-option label="大专" value="大专" />
                <el-option label="本科" value="本科" />
                <el-option label="硕士" value="硕士" />
                <el-option label="博士" value="博士" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12"><el-form-item label="毕业院校" prop="school"><el-input v-model="form.school" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="核心技能">
          <div class="skill-field">
            <div class="skill-input">
              <el-input v-model="skillInput" placeholder="输入技能后回车添加" @keyup.enter="addSkill" />
              <el-button @click="addSkill">添加</el-button>
            </div>
            <div class="skill-tags">
              <el-tag v-for="skill in form.skills" :key="skill" closable @close="removeSkill(skill)">{{ skill }}</el-tag>
            </div>
          </div>
        </el-form-item>
        <el-form-item label="工作/项目经历" prop="work_experience">
          <RichTextEditor v-model="form.work_experience" />
        </el-form-item>
        <el-button type="primary" :loading="saving" @click="save">保存资料</el-button>
      </el-form>
    </div>
    <div class="content-surface profile-resume-surface" v-loading="resumeLoading">
      <div class="section-head">
        <h2>我的简历</h2>
        <el-button type="primary" :icon="UploadFilled" @click="router.push('/resume')">
          {{ currentResume ? '更新简历' : '上传简历' }}
        </el-button>
      </div>
      <div v-if="currentResume" class="resume-current">
        <div class="resume-current__icon">
          <el-icon><Document /></el-icon>
        </div>
        <div class="resume-current__body">
          <div class="resume-current__top">
            <h2>{{ currentResume.file_name }}</h2>
            <el-tag type="success">已上传</el-tag>
          </div>
          <div class="resume-meta">
            <span>{{ (currentResume.file_type || 'pdf').toUpperCase() }}</span>
            <span>{{ formatFileSize(currentResume.file_size) }}</span>
            <span>{{ formatUploadedAt(currentResume.uploaded_at) }}</span>
          </div>
        </div>
      </div>
      <div v-else class="resume-empty">
        <el-icon><Document /></el-icon>
        <h2>未上传简历</h2>
      </div>
    </div>
  </section>
</template>
