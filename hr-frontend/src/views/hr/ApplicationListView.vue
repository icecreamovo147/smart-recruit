<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown, Back, Cpu, Refresh, Search } from '@element-plus/icons-vue'
import { listJobApplications, updateApplicationStatus } from '@/api/application'
import { listApplicationInterviews, batchCancelInterviews } from '@/api/interview'
import type { Application, InterviewSchedule, JobQuery } from '@/types/domain'
import { getHRStatusLabel, getStatusType, APP_STATUS_KEY, TERMINAL_STATUS_KEYS, ALLOWED_HR_ACTIONS } from '@/types/domain'
import InterviewScheduleDialog from '@/components/business/InterviewScheduleDialog.vue'
import SkillTagSummary from '@/components/business/SkillTagSummary.vue'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const errorMessage = ref('')
const list = ref<Application[]>([])
const total = ref(0)
const query = reactive<JobQuery>({ page: 1, page_size: 10 })
const keyword = ref('')
const statusFilter = ref('')

const offerStageStatusKeys = new Set<string>([
  APP_STATUS_KEY.OFFER_PENDING,
  APP_STATUS_KEY.OFFER_SENT,
  APP_STATUS_KEY.OFFER_ACCEPTED,
  APP_STATUS_KEY.OFFER_REJECTED,
  APP_STATUS_KEY.HIRED,
])

const formatDateTime = (value: string): string => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (num: number): string => String(num).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

const getStatusKey = (row: Application): string => {
  return row.status_key || ''
}

const canAction = (row: Application, targetKey: string): boolean => {
  const currentKey = getStatusKey(row)
  if (!currentKey) {
    // Fallback for legacy numeric status
    const legacyStatus = Number(row.status)
    if (targetKey === APP_STATUS_KEY.REJECTED) return legacyStatus < 3
    if (targetKey === APP_STATUS_KEY.SCREEN_PASSED) return legacyStatus < 2
    return false
  }
  if (TERMINAL_STATUS_KEYS.has(currentKey)) return false
  // Check against the server-side transition matrix.
  const allowed = ALLOWED_HR_ACTIONS[targetKey]
  if (!allowed) return false
  return allowed.has(currentKey)
}

const load = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    query.page = Number(query.page) || 1
    query.page_size = Number(query.page_size) || 10
    const data = await listJobApplications(Number(route.params.jobId), query)
    list.value = (data.list || []).map((item) => ({
      ...item,
      application_id: item.application_id,
      skills: Array.isArray(item.skills) ? item.skills : (String(item.skills || '')).split(',').map((skill: string) => skill.trim()).filter(Boolean),
      applied_time_display: formatDateTime(item.applied_at),
      status: Number(item.status ?? 0),
      round_no: item.round_no || 1,
      is_current: Number(item.is_current ?? 1),
    }))
    total.value = Number(data.total) || 0
  } catch (error: unknown) {
    errorMessage.value = error instanceof Error ? error.message : '候选人台账加载失败'
  } finally {
    loading.value = false
  }
}

const statusLabel = (row: Application): string => {
  const key = getStatusKey(row)
  if (key) return getHRStatusLabel(key)
  return ['待查看', '已查看', '通过', '淘汰'][row.status] || '待查看'
}

const statusType = (row: Application): string => {
  const key = getStatusKey(row)
  if (key) return getStatusType(key)
  return ['info', 'primary', 'success', 'danger'][row.status] || 'info'
}

const setRowStatus = async (row: Application, statusKey: string, successMessage: string, reason?: string) => {
  await updateApplicationStatus(row.application_id, statusKey, reason)
  row.status_key = statusKey
  ElMessage.success(successMessage)
}

const viewResume = async (row: Application) => {
  if (!row.resume_url) {
    ElMessage.warning('简历链接暂不可用')
    return
  }
  window.open(row.resume_url, '_blank', 'noopener')
  if (row.file_type && row.file_type !== 'pdf') {
    ElMessage.info('系统暂不支持预览 DOCX 格式的文档，请在本地进行查看')
  }
  const currentKey = getStatusKey(row)
  if (!currentKey || currentKey === APP_STATUS_KEY.APPLIED) {
    await setRowStatus(row, APP_STATUS_KEY.VIEWED, '已标记为已查看')
  }
}

const decide = async (row: Application, statusKey: string) => {
  const text = getHRStatusLabel(statusKey, statusKey === APP_STATUS_KEY.REJECTED ? '淘汰' : '通过')
  const currentKey = getStatusKey(row)
  const isRePass = currentKey === APP_STATUS_KEY.REJECTED && statusKey === APP_STATUS_KEY.SCREEN_PASSED

  let reason: string | undefined
  if (statusKey === APP_STATUS_KEY.REJECTED) {
    try {
      const { value } = await ElMessageBox.prompt(`确认将「${row.real_name || '该候选人'}」标记为${text}？请输入淘汰原因。`, '更新投递状态', {
        inputPlaceholder: '请输入淘汰原因',
        type: 'warning',
      })
      reason = value
    } catch {
      return
    }
  } else if (isRePass) {
    try {
      await ElMessageBox.confirm(
        `该候选人之前已被淘汰，重新通过将作为第 ${(row.round_no || 0) + 1} 轮投递处理，确认继续？`,
        '重新通过候选人',
        { type: 'warning', confirmButtonText: '确认重新通过' },
      )
    } catch {
      return
    }
  } else if (statusKey === APP_STATUS_KEY.INTERVIEW_PASSED) {
    try {
      await ElMessageBox.confirm(
        `确认将「${row.real_name || '该候选人'}」标记为面试通过？通过后可安排下一轮面试或发起 Offer。`,
        '面试通过确认',
        { type: 'success' },
      )
    } catch {
      return
    }
  } else {
    try {
      await ElMessageBox.confirm(`确认将「${row.real_name || '该候选人'}」标记为${text}？`, '更新投递状态', { type: 'success' })
    } catch {
      return
    }
  }
  await setRowStatus(row, statusKey, `已标记为${text}`, reason)
}

const aiAnalyze = (row: Application) => {
  router.push({ path: '/hr/ai', query: { application_id: String(row.application_id), candidate_name: row.real_name || '该求职者' } })
}

const openIntelligence = (row: Application) => {
  router.push({
    path: `/hr/applications/${row.application_id}/intelligence`,
    query: {
      job_id: String(route.params.jobId),
      job_title: row.job_title || '',
      candidate_name: row.real_name || '',
    },
  })
}

const shouldOpenOfferCreate = (row: Application): boolean => {
  const statusKey = getStatusKey(row)
  return statusKey === APP_STATUS_KEY.VIEWED || statusKey === APP_STATUS_KEY.INTERVIEW_PASSED
}

const canOpenOfferWorkspace = (row: Application): boolean => {
  return shouldOpenOfferCreate(row) || offerStageStatusKeys.has(getStatusKey(row))
}

const offerActionLabel = (row: Application): string => {
  return shouldOpenOfferCreate(row) ? '创建 Offer' : '管理 Offer'
}

// ── Interview scheduling dialog ──────────────────────────────────────────

const scheduleVisible = ref(false)
const scheduleTarget = ref<Application | null>(null)

const openScheduleDialog = (row: Application) => {
  scheduleTarget.value = row
  scheduleVisible.value = true
}

const onScheduleSuccess = () => {
  ElMessage.success('面试安排成功，候选人状态已更新')
  // Reload list because backend auto-transitions application status to interview_pending
  load()
}

// ── Cancel interview ────────────────────────────────────────────────

const handleCancelInterview = async (row: Application) => {
  try {
    const data = await listApplicationInterviews(row.application_id)
    const activeInterviews = (data.list || []).filter(
      (iv: InterviewSchedule) => iv.status === 'scheduled' || iv.status === 'pending',
    )
    if (activeInterviews.length === 0) {
      ElMessage.warning('该候选人没有可取消的面试')
      return
    }
    const { value: reason } = await ElMessageBox.prompt('请输入取消原因', '取消面试', {
      confirmButtonText: '确认取消',
      cancelButtonText: '返回',
      inputPattern: /.{1,}/,
      inputErrorMessage: '请填写取消原因',
    })
    // Cancel all active interviews for this application via batch API
    await batchCancelInterviews(row.application_id, reason || '')
    ElMessage.success('已取消该候选人的所有面试')
    load()
  } catch {
    // User cancelled or error handled by interceptor
  }
}

const handleDropdownCommand = (command: string, row: Application) => {
  switch (command) {
    case 'ai_analyze':
      aiAnalyze(row)
      break
    case 'intelligence':
      openIntelligence(row)
      break
    case 'schedule_interview':
      openScheduleDialog(row)
      break
    case 'cancel_interview':
      handleCancelInterview(row)
      break
    case 'interview_passed':
      decide(row, APP_STATUS_KEY.INTERVIEW_PASSED)
      break
    case 'manage_offer':
      router.push({
        path: `/hr/applications/${row.application_id}/offers`,
        query: {
          job_id: String(route.params.jobId),
          job_title: row.job_title || '',
          candidate_name: row.real_name || '',
          ...(shouldOpenOfferCreate(row) ? { action: 'create' } : {}),
        },
      })
      break
    case 'rejected':
      decide(row, APP_STATUS_KEY.REJECTED)
      break
  }
}

const filteredList = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return list.value.filter((item) => {
    const matchesKeyword = !q
      || (item.real_name || '').toLowerCase().includes(q)
      || (item.phone || '').toLowerCase().includes(q)
      || (item.school || '').toLowerCase().includes(q)
      || (item.skills || []).some((skill: string) => skill.toLowerCase().includes(q))
    const matchesStatus = !statusFilter.value || getStatusKey(item) === statusFilter.value
    return matchesKeyword && matchesStatus
  })
})

const ledgerStats = computed(() => [
  { label: '投递总数', value: total.value || list.value.length, hint: '当前岗位候选人池' },
  { label: '待查看', value: list.value.filter((item) => getStatusKey(item) === APP_STATUS_KEY.APPLIED || Number(item.status) === 0).length, hint: '需要初筛处理' },
  { label: '面试中', value: list.value.filter((item) => new Set<string>([APP_STATUS_KEY.INTERVIEW_PENDING, APP_STATUS_KEY.INTERVIEW_PASSED]).has(getStatusKey(item))).length, hint: '已进入面试流程' },
  { label: '已终态', value: list.value.filter((item) => TERMINAL_STATUS_KEYS.has(getStatusKey(item))).length, hint: 'Offer 或淘汰完成' },
])

onMounted(load)
</script>

<template>
  <section class="console-page console-page--fill application-ledger-page">
    <div class="workspace-surface">
      <div class="workspace-surface__header">
        <div class="workspace-surface__header-copy">
          <p class="console-eyebrow">CANDIDATE LEDGER</p>
          <h1 class="console-title">候选人台账</h1>
          <p class="console-description">围绕当前岗位跟进候选人投递、简历查看、AI 分析、面试安排和 Offer 推进。</p>
        </div>
        <div class="workspace-surface__header-actions">
          <el-button :icon="Back" @click="$router.push('/hr/jobs')">返回岗位</el-button>
          <el-button :icon="Refresh" @click="load">刷新</el-button>
        </div>
      </div>

      <div class="workspace-surface__divider"></div>

      <div class="workspace-surface__toolbar">
        <div class="workspace-surface__filters">
          <el-input v-model="keyword" :prefix-icon="Search" clearable placeholder="搜索姓名 / 电话 / 学校 / 技能" style="width: 300px" />
          <el-select v-model="statusFilter" clearable placeholder="全部状态" style="width: 160px">
            <el-option label="已投递" :value="APP_STATUS_KEY.APPLIED" />
            <el-option label="已查看" :value="APP_STATUS_KEY.VIEWED" />
            <el-option label="初筛通过" :value="APP_STATUS_KEY.SCREEN_PASSED" />
            <el-option label="待面试" :value="APP_STATUS_KEY.INTERVIEW_PENDING" />
            <el-option label="面试通过" :value="APP_STATUS_KEY.INTERVIEW_PASSED" />
            <el-option label="待发 Offer" :value="APP_STATUS_KEY.OFFER_PENDING" />
            <el-option label="已淘汰" :value="APP_STATUS_KEY.REJECTED" />
          </el-select>
        </div>
      </div>
      <el-alert v-if="errorMessage" class="workspace-surface__error" type="error" :title="errorMessage" show-icon :closable="false">
        <template #default>
          <el-button size="small" type="danger" plain @click="load">重试</el-button>
        </template>
      </el-alert>
      <div class="workspace-surface__body desktop-only">
        <el-table class="console-table" height="100%" v-loading="loading" :data="filteredList" empty-text="暂无投递">
          <el-table-column label="候选人" min-width="220">
            <template #default="{ row }">
              <div class="console-entity">
                <div class="console-entity__name">{{ row.real_name || '未知姓名' }}</div>
                <div class="console-entity__meta">{{ row.phone || '未填写电话' }} / {{ row.education || '-' }} / {{ row.school || '-' }}</div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="技能" min-width="220">
            <template #default="{ row }">
              <SkillTagSummary :skills="row.skills" :limit="3" />
            </template>
          </el-table-column>
          <el-table-column prop="applied_time_display" label="投递时间" width="170" />
          <el-table-column label="轮次" width="90">
            <template #default="{ row }">第 {{ row.round_no }} 轮</template>
          </el-table-column>
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <el-tag :type="statusType(row)">{{ statusLabel(row) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="370" fixed="right">
            <template #default="{ row }">
              <div class="application-actions">
                <el-button size="small" type="primary" plain @click="$router.push('/hr/candidates/' + row.user_id)">候选人详情</el-button>
                <el-button size="small" type="success" plain :icon="Cpu" @click="openIntelligence(row)">智能评估</el-button>
                <el-button size="small" type="primary" plain @click="viewResume(row)">查看简历</el-button>
                <el-dropdown trigger="click" @command="(cmd: string) => handleDropdownCommand(cmd, row)">
                  <el-button size="small" type="info" plain>
                    更多<el-icon class="el-icon--right"><ArrowDown /></el-icon>
                  </el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="ai_analyze">AI 分析</el-dropdown-item>
                      <el-dropdown-item command="intelligence">画像与匹配评估</el-dropdown-item>
                      <el-dropdown-item divided command="schedule_interview" :disabled="!canAction(row, APP_STATUS_KEY.INTERVIEW_PENDING)">安排面试</el-dropdown-item>
                      <el-dropdown-item command="cancel_interview">取消面试</el-dropdown-item>
                      <el-dropdown-item command="interview_passed" :disabled="!canAction(row, APP_STATUS_KEY.INTERVIEW_PASSED)">面试通过</el-dropdown-item>
                      <el-dropdown-item command="manage_offer" :disabled="!canOpenOfferWorkspace(row)">{{ offerActionLabel(row) }}</el-dropdown-item>
                      <el-dropdown-item command="rejected" :disabled="!canAction(row, APP_STATUS_KEY.REJECTED)">淘汰</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="workspace-surface__pagination desktop-only">
        <el-pagination v-model:current-page="query.page" v-model:page-size="query.page_size" layout="total, prev, pager, next, sizes" :total="total" @current-change="load" @size-change="load" />
      </div>

      <div class="mobile-card-list mobile-only">
        <el-empty v-if="!loading && filteredList.length === 0" description="暂无投递" />
        <div v-for="row in filteredList" :key="row.application_id" class="mobile-application-card">
          <div class="mobile-card__header">
            <h3 class="mobile-card__title">{{ row.real_name || '未知姓名' }}</h3>
            <el-tag :type="statusType(row)" size="small">{{ statusLabel(row) }}</el-tag>
          </div>
          <div class="mobile-card__meta">
            <span>{{ row.phone || '未填写电话' }}</span>
            <span>{{ row.education || '' }}{{ row.school ? ' / ' + row.school : '' }}</span>
          </div>
          <div class="mobile-card__tags">
            <SkillTagSummary :skills="row.skills" :limit="2" trigger="click" />
          </div>
          <div class="mobile-card__meta">
            <span>{{ row.applied_time_display }}</span>
            <span>第 {{ row.round_no }} 轮</span>
          </div>
          <div class="mobile-card__actions">
            <el-button size="small" type="primary" plain @click="$router.push('/hr/candidates/' + row.user_id)">候选人详情</el-button>
            <el-button size="small" type="success" plain :icon="Cpu" @click="openIntelligence(row)">智能评估</el-button>
            <el-button size="small" type="primary" plain @click="viewResume(row)">查看简历</el-button>
            <el-dropdown trigger="click" @command="(cmd: string) => handleDropdownCommand(cmd, row)">
              <el-button size="small" type="info" plain>更多操作</el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="ai_analyze">AI 分析</el-dropdown-item>
                  <el-dropdown-item command="intelligence">画像与匹配评估</el-dropdown-item>
                  <el-dropdown-item divided command="schedule_interview" :disabled="!canAction(row, APP_STATUS_KEY.INTERVIEW_PENDING)">安排面试</el-dropdown-item>
                  <el-dropdown-item command="cancel_interview">取消面试</el-dropdown-item>
                  <el-dropdown-item command="interview_passed" :disabled="!canAction(row, APP_STATUS_KEY.INTERVIEW_PASSED)">面试通过</el-dropdown-item>
                  <el-dropdown-item command="manage_offer" :disabled="!canOpenOfferWorkspace(row)">{{ offerActionLabel(row) }}</el-dropdown-item>
                  <el-dropdown-item command="rejected" :disabled="!canAction(row, APP_STATUS_KEY.REJECTED)">淘汰</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </div>
    </div>

    <!-- Interview scheduling dialog -->
    <InterviewScheduleDialog
      v-if="scheduleTarget"
      :visible="scheduleVisible"
      :application-id="scheduleTarget.application_id"
      :job-title="scheduleTarget.job_title"
      :candidate-name="scheduleTarget.real_name || ''"
      @update:visible="scheduleVisible = $event"
      @success="onScheduleSuccess"
    />
  </section>
</template>
