<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listJobApplications, updateApplicationStatus } from '@/api/application'
import type { Application, JobQuery } from '@/types/domain'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const errorMessage = ref('')
const list = ref<Application[]>([])
const total = ref(0)
const query = reactive<JobQuery>({ page: 1, page_size: 10 })
const statusText = ['待查看', '已查看', '通过', '淘汰']
const statusType = ['info', 'primary', 'success', 'danger']

const normalizeStatus = (value: unknown): number => {
  const status = Number(value)
  return Number.isInteger(status) && status >= 0 && status < statusText.length ? status : 0
}

const formatDateTime = (value: string): string => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (num: number): string => String(num).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
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
      status: normalizeStatus(item.status),
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

const setRowStatus = async (row: Application, status: number, successMessage: string) => {
  await updateApplicationStatus(row.application_id, status)
  row.status = status
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
  if (row.status === 0) {
    await setRowStatus(row, 1, '已标记为已查看')
  }
}

const decide = async (row: Application, status: number) => {
  const text = status === 2 ? '通过' : '淘汰'
  try {
    await ElMessageBox.confirm(`确认将「${row.real_name || '该候选人'}」标记为${text}？`, '更新投递状态', { type: status === 2 ? 'success' : 'warning' })
  } catch {
    return
  }
  await setRowStatus(row, status, `已标记为${text}`)
}

const aiAnalyze = (row: Application) => {
  router.push({ path: '/hr/ai', query: { application_id: String(row.application_id), candidate_name: row.real_name || '该求职者' } })
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
              <el-tag :type="statusType[row.status] || 'info'">{{ statusText[row.status] || '待查看' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="330" fixed="right">
            <template #default="{ row }">
              <div class="application-actions">
                <el-button size="small" type="primary" plain @click="viewResume(row)">查看简历</el-button>
                <el-button size="small" type="primary" plain @click="aiAnalyze(row)">AI 分析</el-button>
                <el-button size="small" type="success" plain :disabled="row.status === 2" @click="decide(row, 2)">通过</el-button>
                <el-button size="small" type="danger" plain :disabled="row.status === 3" @click="decide(row, 3)">淘汰</el-button>
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
            <el-tag :type="statusType[row.status] || 'info'" size="small">{{ statusText[row.status] || '待查看' }}</el-tag>
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
            <el-button size="small" type="primary" plain @click="viewResume(row)">查看简历</el-button>
            <el-button size="small" type="primary" plain @click="aiAnalyze(row)">AI 分析</el-button>
            <el-button size="small" type="success" plain :disabled="row.status === 2" @click="decide(row, 2)">通过</el-button>
            <el-button size="small" type="danger" plain :disabled="row.status === 3" @click="decide(row, 3)">淘汰</el-button>
          </div>
        </div>
      </div>
      <el-pagination class="application-ledger-pagination" v-model:current-page="query.page" v-model:page-size="query.page_size" layout="total, prev, pager, next, sizes" :total="total" @current-change="load" @size-change="load" />
    </div>
  </section>
</template>
