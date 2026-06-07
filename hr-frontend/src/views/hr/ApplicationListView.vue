<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown } from '@element-plus/icons-vue'
import { listJobApplications, updateApplicationStatus } from '@/api/application'
import type { Application, JobQuery } from '@/types/domain'
import { getHRStatusLabel, getStatusType, APP_STATUS_KEY, TERMINAL_STATUS_KEYS, ALLOWED_HR_ACTIONS } from '@/types/domain'
import InterviewScheduleDialog from '@/components/business/InterviewScheduleDialog.vue'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const errorMessage = ref('')
const list = ref<Application[]>([])
const total = ref(0)
const query = reactive<JobQuery>({ page: 1, page_size: 10 })

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
  } else if (statusKey === APP_STATUS_KEY.OFFER_PENDING) {
    try {
      await ElMessageBox.confirm(
        `确认将「${row.real_name || '该候选人'}」推进到待发 Offer 阶段？`,
        '发起 Offer 确认',
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

const handleDropdownCommand = (command: string, row: Application) => {
  switch (command) {
    case 'ai_analyze':
      aiAnalyze(row)
      break
    case 'schedule_interview':
      openScheduleDialog(row)
      break
    case 'interview_passed':
      decide(row, APP_STATUS_KEY.INTERVIEW_PASSED)
      break
    case 'offer_pending':
      decide(row, APP_STATUS_KEY.OFFER_PENDING)
      break
    case 'manage_offer':
      router.push({ path: `/hr/applications/${row.application_id}/offers`, query: { job_id: String(route.params.jobId) } })
      break
    case 'rejected':
      decide(row, APP_STATUS_KEY.REJECTED)
      break
  }
}

onMounted(load)
</script>

<template>
  <section class="application-ledger-page">
    <div class="page-header">
      <h1 class="page-title">候选人台账</h1>
      <el-button @click="$router.push('/hr/jobs')">返回岗位</el-button>
    </div>
    <div class="content-surface application-ledger-surface">
      <el-alert v-if="errorMessage" class="page-error" type="error" :title="errorMessage" show-icon :closable="false">
        <template #default>
          <el-button size="small" type="danger" plain @click="load">重试</el-button>
        </template>
      </el-alert>
      <div class="application-ledger-table-area desktop-only">
        <el-table class="application-ledger-table" height="100%" v-loading="loading" :data="list" empty-text="暂无投递">
          <el-table-column prop="real_name" label="姓名" width="110" />
          <el-table-column prop="phone" label="电话" width="140" />
          <el-table-column prop="education" label="学历" width="100" />
          <el-table-column prop="school" label="学校" min-width="150" />
          <el-table-column label="技能" min-width="220">
            <template #default="{ row }">
              <el-tag v-for="skill in row.skills" :key="skill" style="margin-right: 6px">{{ skill }}</el-tag>
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
          <el-table-column label="操作" width="280" fixed="right">
            <template #default="{ row }">
              <div class="application-actions">
                <el-button size="small" type="primary" plain @click="$router.push('/hr/candidates/' + row.user_id)">候选人详情</el-button>
                <el-button size="small" type="primary" plain @click="viewResume(row)">查看简历</el-button>
                <el-dropdown trigger="click" @command="(cmd: string) => handleDropdownCommand(cmd, row)">
                  <el-button size="small" type="info" plain>
                    更多<el-icon class="el-icon--right"><ArrowDown /></el-icon>
                  </el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="ai_analyze">AI 分析</el-dropdown-item>
                      <el-dropdown-item divided command="schedule_interview" :disabled="!canAction(row, APP_STATUS_KEY.INTERVIEW_PENDING)">安排面试</el-dropdown-item>
                      <el-dropdown-item command="interview_passed" :disabled="!canAction(row, APP_STATUS_KEY.INTERVIEW_PASSED)">面试通过</el-dropdown-item>
                      <el-dropdown-item command="offer_pending" :disabled="!canAction(row, APP_STATUS_KEY.OFFER_PENDING)">推进至Offer阶段</el-dropdown-item>
                      <el-dropdown-item command="manage_offer">创建/发送Offer</el-dropdown-item>
                      <el-dropdown-item command="rejected" :disabled="!canAction(row, APP_STATUS_KEY.REJECTED)">淘汰</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <!-- Mobile candidate cards -->
      <div class="mobile-card-list mobile-only">
        <el-empty v-if="!loading && list.length === 0" description="暂无投递" />
        <div v-for="row in list" :key="row.application_id" class="mobile-application-card">
          <div class="mobile-card__header">
            <h3 class="mobile-card__title">{{ row.real_name || '未知姓名' }}</h3>
            <el-tag :type="statusType(row)" size="small">{{ statusLabel(row) }}</el-tag>
          </div>
          <div class="mobile-card__meta">
            <span>{{ row.phone || '未填写电话' }}</span>
            <span>{{ row.education || '' }}{{ row.school ? ' / ' + row.school : '' }}</span>
          </div>
          <div v-if="row.skills && row.skills.length" class="mobile-card__tags">
            <el-tag v-for="skill in row.skills" :key="skill" size="small">{{ skill }}</el-tag>
          </div>
          <div class="mobile-card__meta">
            <span>{{ row.applied_time_display }}</span>
            <span>第 {{ row.round_no }} 轮</span>
          </div>
          <div class="mobile-card__actions">
            <el-button size="small" type="primary" plain @click="$router.push('/hr/candidates/' + row.user_id)">候选人详情</el-button>
            <el-button size="small" type="primary" plain @click="viewResume(row)">查看简历</el-button>
            <el-dropdown trigger="click" @command="(cmd: string) => handleDropdownCommand(cmd, row)">
              <el-button size="small" type="info" plain>更多操作</el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="ai_analyze">AI 分析</el-dropdown-item>
                  <el-dropdown-item divided command="schedule_interview" :disabled="!canAction(row, APP_STATUS_KEY.INTERVIEW_PENDING)">安排面试</el-dropdown-item>
                  <el-dropdown-item command="interview_passed" :disabled="!canAction(row, APP_STATUS_KEY.INTERVIEW_PASSED)">面试通过</el-dropdown-item>
                  <el-dropdown-item command="offer_pending" :disabled="!canAction(row, APP_STATUS_KEY.OFFER_PENDING)">推进至Offer阶段</el-dropdown-item>
                  <el-dropdown-item command="manage_offer">创建/发送Offer</el-dropdown-item>
                  <el-dropdown-item command="rejected" :disabled="!canAction(row, APP_STATUS_KEY.REJECTED)">淘汰</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </div>
      <el-pagination class="application-ledger-pagination" v-model:current-page="query.page" v-model:page-size="query.page_size" layout="total, prev, pager, next, sizes" :total="total" @current-change="load" @size-change="load" />
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
