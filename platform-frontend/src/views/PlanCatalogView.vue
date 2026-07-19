<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listPlans, publishPlanVersion, savePlanVersion } from '@/api/control'
import { PLATFORM_PERMISSIONS } from '@/permissions'
import { useAuthStore } from '@/stores/auth'
import type { PlatformEntitlement, PlatformPlan, PlatformPlanVersion } from '@/types'

const auth = useAuthStore()
const loading = ref(false)
const plans = ref<PlatformPlan[]>([])
const canManage = computed(() => auth.can(PLATFORM_PERMISSIONS.PLAN_MANAGE))
const canPublish = computed(() => auth.can(PLATFORM_PERMISSIONS.PLAN_PUBLISH))
const editorVisible = ref(false)
const publishVisible = ref(false)
const selectedPlan = ref<PlatformPlan | null>(null)
const selectedVersion = ref<PlatformPlanVersion | null>(null)
const form = reactive({ version_id: 0, change_note: '', members: 1, jobs: 1, applications: 1, resumes: 1 })
const publishForm = reactive({ effective_at: '', reason: '' })

const entitlementLabels: Record<string, string> = {
  'members.max': '有效成员上限', 'jobs.published.max': '在线岗位上限',
  'applications.monthly.max': '月投递上限', 'resumes.storage.max': '简历存储上限',
}

const load = async () => {
  loading.value = true
  try { plans.value = (await listPlans()).list || [] } finally { loading.value = false }
}

const valueOf = (version: PlatformPlanVersion | undefined, key: string) => {
  const value = version?.entitlements.find((item) => item.key === key)?.value_json
  return value ? Number(value) : 0
}

const openEditor = (plan: PlatformPlan, version?: PlatformPlanVersion) => {
  selectedPlan.value = plan
  selectedVersion.value = version || null
  form.version_id = version?.id || 0
  form.change_note = version?.change_note || ''
  form.members = valueOf(version, 'members.max') || 10
  form.jobs = valueOf(version, 'jobs.published.max') || 20
  form.applications = valueOf(version, 'applications.monthly.max') || 500
  form.resumes = valueOf(version, 'resumes.storage.max') || 1000
  editorVisible.value = true
}

const entitlements = (): PlatformEntitlement[] => [
  ['members.max', form.members], ['jobs.published.max', form.jobs],
  ['applications.monthly.max', form.applications], ['resumes.storage.max', form.resumes],
].map(([key, value]) => ({ key: String(key), value_type: 'integer', value_json: String(value), enforcement_mode: 'hard' }))

const submitDraft = async () => {
  if (!selectedPlan.value || !form.change_note.trim()) { ElMessage.warning('请填写版本变更说明'); return }
  await savePlanVersion(selectedPlan.value.id, { version_id: form.version_id || undefined, change_note: form.change_note.trim(), entitlements: entitlements() })
  editorVisible.value = false
  ElMessage.success('套餐版本草稿已保存')
  await load()
}

const openPublish = (plan: PlatformPlan, version: PlatformPlanVersion) => {
  selectedPlan.value = plan
  selectedVersion.value = version
  publishForm.effective_at = new Date().toISOString()
  publishForm.reason = ''
  publishVisible.value = true
}

const submitPublish = async () => {
  if (!selectedPlan.value || !selectedVersion.value || !publishForm.reason.trim()) { ElMessage.warning('请填写发布原因'); return }
  await publishPlanVersion(selectedPlan.value.id, selectedVersion.value.id, { effective_at: new Date(publishForm.effective_at).toISOString(), reason: publishForm.reason.trim() })
  publishVisible.value = false
  ElMessage.success('套餐版本已发布')
  await load()
}

const formatTime = (value?: string) => value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
onMounted(load)
</script>

<template>
  <section class="console-page" v-loading="loading">
    <div class="plan-grid">
      <article v-for="plan in plans" :key="plan.id" class="surface-card plan-card">
        <header><div><span class="plan-key">{{ plan.plan_key }}</span><h2>{{ plan.name }}</h2><p>{{ plan.description }}</p></div><el-tag :type="plan.status === 'active' ? 'success' : 'info'">{{ plan.status === 'active' ? '启用' : '已退役' }}</el-tag></header>
        <div class="plan-version-list">
          <section v-for="version in plan.versions" :key="version.id" class="plan-version">
            <div class="plan-version__heading"><div><strong>版本 V{{ version.version }}</strong><small>{{ version.change_note || '暂无版本说明' }}</small></div><el-tag :type="version.status === 'published' ? 'success' : version.status === 'draft' ? 'warning' : 'info'" size="small">{{ version.status }}</el-tag></div>
            <div class="entitlement-grid"><div v-for="item in version.entitlements" :key="item.key"><span>{{ entitlementLabels[item.key] || item.key }}</span><strong>{{ Number(item.value_json).toLocaleString() }}</strong><small>{{ item.enforcement_mode === 'hard' ? '硬限制' : item.enforcement_mode }}</small></div></div>
            <footer><span>{{ version.status === 'published' ? `生效：${formatTime(version.effective_at)}` : `更新：${formatTime(version.updated_at)}` }}</span><div><el-button v-if="canManage && version.status === 'draft'" link type="primary" @click="openEditor(plan, version)">编辑草稿</el-button><el-button v-if="canPublish && version.status === 'draft'" link type="success" @click="openPublish(plan, version)">发布</el-button></div></footer>
          </section>
        </div>
        <el-button v-if="canManage" class="plan-new-version" plain @click="openEditor(plan)">创建新版本草稿</el-button>
      </article>
    </div>

    <el-dialog v-model="editorVisible" :title="`${selectedPlan?.name || ''} · ${form.version_id ? '编辑草稿' : '新建版本'}`" width="680px">
      <el-alert title="已发布版本不可修改；保存新草稿不会立即影响任何租户。" type="info" :closable="false" show-icon />
      <el-form class="dialog-form" label-position="top"><el-form-item label="版本变更说明" required><el-input v-model="form.change_note" maxlength="500" show-word-limit placeholder="说明本版本权益调整背景" /></el-form-item><div class="two-columns"><el-form-item label="有效成员上限"><el-input-number v-model="form.members" :min="1" :max="1000000" controls-position="right" /></el-form-item><el-form-item label="在线岗位上限"><el-input-number v-model="form.jobs" :min="1" :max="1000000" controls-position="right" /></el-form-item><el-form-item label="月投递上限"><el-input-number v-model="form.applications" :min="1" :max="100000000" controls-position="right" /></el-form-item><el-form-item label="简历存储上限"><el-input-number v-model="form.resumes" :min="1" :max="100000000" controls-position="right" /></el-form-item></div></el-form>
      <template #footer><el-button @click="editorVisible = false">取消</el-button><el-button type="primary" @click="submitDraft">保存草稿</el-button></template>
    </el-dialog>

    <el-dialog v-model="publishVisible" title="发布套餐版本" width="560px">
      <el-alert title="发布后版本内容将冻结，并可用于租户订阅。" type="warning" :closable="false" show-icon />
      <el-form class="dialog-form" label-position="top"><el-form-item label="生效时间" required><el-date-picker v-model="publishForm.effective_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" style="width:100%" /></el-form-item><el-form-item label="发布原因" required><el-input v-model="publishForm.reason" type="textarea" :rows="4" maxlength="500" show-word-limit /></el-form-item></el-form>
      <template #footer><el-button @click="publishVisible = false">取消</el-button><el-button type="primary" @click="submitPublish">确认发布</el-button></template>
    </el-dialog>
  </section>
</template>
