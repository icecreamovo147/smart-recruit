<script setup lang="ts">
import { t } from '@shared/i18n'
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { Edit, Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { applyProfileFill, fillProfileFromResume, getProfile, updateProfile } from '@/api/profile'
import { updateEmail } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import type {
  CandidateEducationInfo,
  CandidateExperienceInfo,
  Profile,
  ProfileFillDraft,
  ProfileFillFieldDiff,
} from '@/types/domain'
import { isRegionComplete, normalizeRegionForProfile } from '@shared/utils/region'
import RichTextEditor from '@/components/RichTextEditor.vue'
import SchoolAutocomplete from '@/components/SchoolAutocomplete.vue'
import RegionCascader from '@/components/RegionCascader.vue'
import ProfileFillDiffDialog from '@/components/ProfileFillDiffDialog.vue'

const DEGREE_OPTIONS = [
  { label: '高中', value: '高中' },
  { label: '大专', value: '大专' },
  { label: '本科', value: '本科' },
  { label: '硕士', value: '硕士' },
  { label: '博士', value: '博士' },
] as const

const JOB_STATUS_OPTIONS = [
  { label: '在职', value: 'employed' },
  { label: '离职', value: 'resigned' },
  { label: '应届生', value: 'fresh_graduate' },
  { label: '在校生', value: 'student' },
] as const

const route = useRoute()
const auth = useAuthStore()
const loading = ref(false)
const saving = ref(false)
const fillLoading = ref(false)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const formRef = ref<any>(null)
const profileComplete = ref(false)
const skillInput = ref('')
const editingEmail = ref(false)
const emailInput = ref('')
const savingEmail = ref(false)
const overwriteOnFill = ref(false)
const fillDialogVisible = ref(false)
const fillReparseLoading = ref(false)
const fillDraft = ref<Profile | null>(null)
const fillDiffs = ref<ProfileFillFieldDiff[]>([])
const fillWarnings = ref<string[]>([])
const fillRefreshed = ref(false)
const fillRefreshReason = ref('')

const form = reactive({
  real_name: '',
  phone: '',
  city: '',
  years_of_experience: 0,
  job_status: '',
  expected_position: '',
  expected_salary_min: undefined as number | undefined,
  expected_salary_max: undefined as number | undefined,
  available_from: '',
  summary: '',
  skills: [] as string[],
  educations: [] as CandidateEducationInfo[],
  experiences: [] as CandidateExperienceInfo[],
})

const rules = computed(() => ({
  real_name: [{ required: true, message: '请输入真实姓名', trigger: 'blur' }],
  phone: [
    { required: true, message: '请输入联系电话', trigger: 'blur' },
    { pattern: /^1\d{10}$/, message: '请输入有效的手机号', trigger: 'blur' },
  ],
  city: [{
    required: true,
    validator: (_: unknown, __: unknown, callback: (error?: Error) => void) => {
      if (!isRegionComplete(form.city)) {
        callback(new Error(t('validation.required_fields')))
        return
      }
      callback()
    },
    trigger: 'change',
  }],
  job_status: [{ required: true, message: '请选择求职状态', trigger: 'change' }],
  expected_position: [{ required: true, message: '请输入期望岗位', trigger: 'blur' }],
  summary: [{ required: true, message: '请填写个人简介', trigger: 'blur' }],
  skills: [{
    validator: (_: unknown, __: unknown, callback: (error?: Error) => void) => {
      if (form.skills.length === 0) callback(new Error(t('validation.required_fields')))
      else callback()
    },
    trigger: 'change',
  }],
}))

const mapProfileToForm = (data: Profile) => {
  form.real_name = data.real_name || data.realName || ''
  form.phone = data.phone || ''
  form.city = normalizeRegionForProfile(data.city || '') || data.city || ''
  form.years_of_experience = data.years_of_experience ?? data.yearsOfExperience ?? 0
  form.job_status = data.job_status || data.jobStatus || ''
  form.expected_position = data.expected_position || data.expectedPosition || ''
  form.expected_salary_min = data.expected_salary_min ?? data.expectedSalaryMin
  form.expected_salary_max = data.expected_salary_max ?? data.expectedSalaryMax
  form.available_from = data.available_from || data.availableFrom || ''
  form.summary = data.summary || ''
  form.skills = Array.isArray(data.skills)
    ? data.skills
    : String(data.skills || '').split(',').map((item) => item.trim()).filter(Boolean)
  form.educations = (data.educations || []).map((item, index) => ({ ...item, sort_order: item.sort_order ?? index }))
  form.experiences = (data.experiences || []).map((item, index) => ({ ...item, sort_order: item.sort_order ?? index }))
  profileComplete.value = Boolean(data.is_complete ?? data.isComplete)
}

const buildPayload = (): Profile => ({
  real_name: form.real_name,
  phone: form.phone,
  city: form.city,
  years_of_experience: form.years_of_experience,
  job_status: form.job_status,
  expected_position: form.expected_position,
  expected_salary_min: form.expected_salary_min,
  expected_salary_max: form.expected_salary_max,
  available_from: form.available_from,
  summary: form.summary,
  skills: form.skills.join(','),
  educations: form.educations.map((item, index) => ({ ...item, sort_order: index })),
  experiences: form.experiences.map((item, index) => ({ ...item, sort_order: index })),
  education: '',
  school: '',
  work_experience: '',
})

const load = async () => {
  loading.value = true
  try {
    const data = await getProfile()
    mapProfileToForm(data)
    emailInput.value = auth.email || ''
  } finally {
    loading.value = false
  }
}

const addEducation = () => {
  form.educations.push({ school: '', degree: '', major: '', start_date: '', end_date: '', description: '' })
}

const removeEducation = (index: number) => {
  form.educations.splice(index, 1)
}

const addExperience = () => {
  form.experiences.push({ company: '', title: '', location: '', start_date: '', end_date: '', is_current: 0, description: '' })
}

const removeExperience = (index: number) => {
  form.experiences.splice(index, 1)
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

const clearCityValidate = () => {
  formRef.value?.clearValidate('city')
}

const save = async () => {
  if (!formRef.value) return
  if (form.educations.length === 0) {
    ElMessage.warning(t('common.invalid_request'))
    return
  }
  try {
    await formRef.value.validate()
  } catch {
    return
  }
  saving.value = true
  try {
    await updateProfile(buildPayload())
    ElMessage.success(t('common.success'))
    await load()
  } finally {
    saving.value = false
  }
}

const applyFillResponse = (payload: ProfileFillDraft) => {
  fillDraft.value = payload.draft || null
  fillDiffs.value = payload.field_diffs || payload.fieldDiffs || []
  fillWarnings.value = payload.warnings || []
  fillRefreshed.value = Boolean(payload.refreshed)
  fillRefreshReason.value = payload.refresh_reason || payload.refreshReason || ''
}

const openFillPreview = async (forceRefresh = false) => {
  fillLoading.value = true
  try {
    const draftWrap = await fillProfileFromResume({
      force_refresh: forceRefresh,
      overwrite_existing: overwriteOnFill.value,
    })
    applyFillResponse(draftWrap)
    if (!fillDraft.value) {
      ElMessage.warning(t('common.invalid_request'))
      return
    }
    fillDialogVisible.value = true
  } finally {
    fillLoading.value = false
  }
}

const reparseFillPreview = async () => {
  fillReparseLoading.value = true
  try {
    const draftWrap = await fillProfileFromResume({
      force_refresh: true,
      overwrite_existing: overwriteOnFill.value,
    })
    applyFillResponse(draftWrap)
    if (!fillDraft.value) {
      ElMessage.warning(t('common.invalid_request'))
      fillDialogVisible.value = false
    }
  } finally {
    fillReparseLoading.value = false
  }
}

const confirmFillApply = async () => {
  if (!fillDraft.value) return
  fillLoading.value = true
  try {
    const merged = await applyProfileFill({
      draft: fillDraft.value,
      overwrite_existing: overwriteOnFill.value,
    })
    mapProfileToForm(merged)
    fillDialogVisible.value = false
    ElMessage.success(t('common.success'))
  } finally {
    fillLoading.value = false
  }
}

const startEditEmail = () => {
  emailInput.value = auth.email || ''
  editingEmail.value = true
}

const saveEmail = async () => {
  savingEmail.value = true
  try {
    await updateEmail(emailInput.value.trim())
    ElMessage.success(t('common.success'))
    editingEmail.value = false
    await auth.restoreSession()
  } finally {
    savingEmail.value = false
  }
}

const cancelEditEmail = () => {
  editingEmail.value = false
}

onMounted(async () => {
  await load()
  if (route.query.fill === '1') {
    await openFillPreview()
  }
})
</script>

<template>
  <section>
    <div class="content-surface" v-loading="loading">
      <div class="email-section">
        <div class="section-head">
          <h2>通知邮箱</h2>
          <el-tag :type="profileComplete ? 'success' : 'warning'" effect="dark">
            {{ profileComplete ? '档案已完善' : '档案未完善，无法投递岗位' }}
          </el-tag>
        </div>
        <p class="email-hint">设置邮箱后，面试安排、Offer 等重要通知将同时发送到您的邮箱。</p>
        <template v-if="editingEmail">
          <div class="email-edit-row">
            <el-input v-model="emailInput" placeholder="请输入邮箱" clearable style="max-width: 320px" />
            <el-button type="primary" :loading="savingEmail" size="small" @click="saveEmail">保存</el-button>
            <el-button type="primary" plain size="small" @click="cancelEditEmail">取消</el-button>
          </div>
        </template>
        <template v-else>
          <div class="email-display">
            <span>{{ auth.email || '未设置' }}</span>
            <el-button :icon="Edit" link type="primary" size="small" @click="startEditEmail">编辑</el-button>
          </div>
        </template>
      </div>

      <el-form ref="formRef" label-position="top" :model="form" :rules="rules" class="profile-form">
        <div class="form-section">
          <div class="section-head">
            <h2>基本信息</h2>
            <div class="section-actions">
              <el-button type="primary" :loading="fillLoading" @click="openFillPreview()">从简历填充</el-button>
            </div>
          </div>
          <el-row :gutter="16">
            <el-col :xs="24" :sm="12"><el-form-item label="真实姓名" prop="real_name"><el-input v-model="form.real_name" /></el-form-item></el-col>
            <el-col :xs="24" :sm="12"><el-form-item label="联系电话" prop="phone"><el-input v-model="form.phone" /></el-form-item></el-col>
            <el-col :xs="24" :sm="12">
              <el-form-item label="所在城市" prop="city">
                <RegionCascader v-model="form.city" @selecting="clearCityValidate" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12">
              <el-form-item label="求职状态" prop="job_status">
                <el-select v-model="form.job_status" placeholder="选择状态" style="width: 100%">
                  <el-option v-for="item in JOB_STATUS_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12">
              <el-form-item label="工作年限（年）">
                <el-input-number v-model="form.years_of_experience" :min="0" :max="50" :step="0.5" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12"><el-form-item label="期望岗位" prop="expected_position"><el-input v-model="form.expected_position" /></el-form-item></el-col>
            <el-col :xs="24" :sm="12"><el-form-item label="期望月薪下限（元）"><el-input-number v-model="form.expected_salary_min" :min="0" :step="1000" style="width: 100%" /></el-form-item></el-col>
            <el-col :xs="24" :sm="12"><el-form-item label="期望月薪上限（元）"><el-input-number v-model="form.expected_salary_max" :min="0" :step="1000" style="width: 100%" /></el-form-item></el-col>
            <el-col :xs="24" :sm="12"><el-form-item label="可到岗日期"><el-date-picker v-model="form.available_from" type="date" value-format="YYYY-MM-DD" placeholder="可选" style="width: 100%" /></el-form-item></el-col>
            <el-col :span="24"><el-form-item label="个人简介" prop="summary"><el-input v-model="form.summary" type="textarea" :rows="3" maxlength="500" show-word-limit /></el-form-item></el-col>
          </el-row>
        </div>

        <div class="form-section">
          <div class="section-head">
            <h2>教育经历</h2>
            <el-button type="primary" :icon="Plus" @click="addEducation">添加教育经历</el-button>
          </div>
          <div v-if="form.educations.length === 0" class="empty-hint">请至少添加一条教育经历</div>
          <div v-for="(edu, index) in form.educations" :key="`edu-${index}`" class="history-card">
            <div class="history-card__head">
              <strong>教育经历 {{ index + 1 }}</strong>
              <el-button link type="primary" @click="removeEducation(index)">删除</el-button>
            </div>
            <el-row :gutter="12">
              <el-col :xs="24" :sm="12"><el-form-item label="学校"><SchoolAutocomplete v-model="edu.school" /></el-form-item></el-col>
              <el-col :xs="24" :sm="12">
                <el-form-item label="学历">
                  <el-select v-model="edu.degree" placeholder="选择学历" clearable style="width: 100%">
                    <el-option v-for="item in DEGREE_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="12"><el-form-item label="专业"><el-input v-model="edu.major" /></el-form-item></el-col>
              <el-col :xs="24" :sm="6"><el-form-item label="开始日期"><el-date-picker v-model="edu.start_date" type="date" value-format="YYYY-MM-DD" style="width: 100%" /></el-form-item></el-col>
              <el-col :xs="24" :sm="6"><el-form-item label="结束日期"><el-date-picker v-model="edu.end_date" type="date" value-format="YYYY-MM-DD" style="width: 100%" /></el-form-item></el-col>
              <el-col :span="24"><el-form-item label="补充说明"><el-input v-model="edu.description" type="textarea" :rows="2" /></el-form-item></el-col>
            </el-row>
          </div>
        </div>

        <div class="form-section">
          <div class="section-head">
            <h2>工作经历</h2>
            <el-button type="primary" :icon="Plus" @click="addExperience">添加工作经历</el-button>
          </div>
          <p class="section-note">工作经历选填；实习生或尚无正式工作经历时可留空</p>
          <div v-if="form.experiences.length === 0" class="empty-hint">暂无工作经历，可点击上方按钮添加</div>
          <div v-for="(exp, index) in form.experiences" :key="`exp-${index}`" class="history-card">
            <div class="history-card__head">
              <strong>工作经历 {{ index + 1 }}</strong>
              <el-button link type="primary" @click="removeExperience(index)">删除</el-button>
            </div>
            <el-row :gutter="12">
              <el-col :xs="24" :sm="12"><el-form-item label="公司"><el-input v-model="exp.company" /></el-form-item></el-col>
              <el-col :xs="24" :sm="12"><el-form-item label="职位"><el-input v-model="exp.title" /></el-form-item></el-col>
              <el-col :xs="24" :sm="12"><el-form-item label="城市"><RegionCascader v-model="exp.location" /></el-form-item></el-col>
              <el-col :xs="24" :sm="6"><el-form-item label="开始日期"><el-date-picker v-model="exp.start_date" type="date" value-format="YYYY-MM-DD" style="width: 100%" /></el-form-item></el-col>
              <el-col :xs="24" :sm="6"><el-form-item label="结束日期"><el-date-picker v-model="exp.end_date" type="date" value-format="YYYY-MM-DD" style="width: 100%" /></el-form-item></el-col>
              <el-col :xs="24" :sm="12"><el-form-item label="在职中"><el-switch v-model="exp.is_current" :active-value="1" :inactive-value="0" /></el-form-item></el-col>
              <el-col :span="24">
                <el-form-item label="工作描述">
                  <RichTextEditor v-model="exp.description" />
                </el-form-item>
              </el-col>
            </el-row>
          </div>
        </div>

        <div class="form-section">
          <div class="section-head">
            <h2>核心技能</h2>
          </div>
          <el-form-item prop="skills">
            <div class="skill-field">
              <div class="skill-input">
                <el-input v-model="skillInput" placeholder="输入技能后回车添加" @keyup.enter="addSkill" />
                <el-button type="primary" @click="addSkill">添加</el-button>
              </div>
              <div class="skill-tags">
                <el-tag v-for="skill in form.skills" :key="skill" closable @close="removeSkill(skill)">{{ skill }}</el-tag>
              </div>
            </div>
          </el-form-item>
          <div class="form-actions">
            <el-button type="primary" :loading="saving" @click="save">保存资料</el-button>
          </div>
        </div>
      </el-form>
    </div>

    <ProfileFillDiffDialog
      v-model="fillDialogVisible"
      v-model:overwrite-existing="overwriteOnFill"
      :loading="fillLoading"
      :reparse-loading="fillReparseLoading"
      :refreshed="fillRefreshed"
      :refresh-reason="fillRefreshReason"
      :warnings="fillWarnings"
      :diffs="fillDiffs"
      @confirm="confirmFillApply"
      @reparse="reparseFillPreview"
    />
  </section>
</template>

<style scoped>
.email-section {
  margin-bottom: 24px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--border-subtle, #eee);
}

.form-section + .form-section {
  margin-top: 8px;
  padding-top: 24px;
  border-top: 1px solid var(--border-subtle, #eee);
}

.email-hint,
.section-note,
.empty-hint {
  font-size: 13px;
  color: var(--text-muted, #888);
  margin: 4px 0 12px;
}

.email-edit-row,
.section-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.email-display {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.history-card {
  border: 1px solid var(--border, #e5e7eb);
  border-radius: 10px;
  padding: 12px 16px 4px;
  margin-bottom: 12px;
  background: var(--surface-muted, #fafafa);
}

.history-card__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.form-actions {
  display: flex;
  justify-content: center;
  margin-top: 8px;
}
</style>
