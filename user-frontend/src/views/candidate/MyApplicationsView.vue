<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { listMyApplications } from '@/api/application'
import type { Application, JobQuery } from '@/types/domain'

const router = useRouter()
const loading = ref(false)
const errorMessage = ref('')
const list = ref<Application[]>([])
const total = ref(0)
const query = reactive<JobQuery>({ page: 1, page_size: 10 })
const statusText = ['待查看', '已查看', '通过', '已进入公司公共人才库']
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
    const data = await listMyApplications(query)
    const applicationList = data.list || []
    list.value = applicationList.map((item) => ({
      ...item,
      applied_time_display: formatDateTime(item.applied_at),
      status: normalizeStatus(item.status),
      round_no: item.round_no || 1,
      is_current: Number(item.is_current ?? 0),
      application_id: item.application_id,
      job_id: item.job_id,
      job_title: item.job_title,
      applied_at: item.applied_at,
    }))
    total.value = Number(data.total) || 0
  } catch (error: unknown) {
    errorMessage.value = error instanceof Error ? error.message : '投递记录加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="applications-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">我的投递</h1>
        <p class="page-subtitle">跟踪每一次投递状态。</p>
      </div>
    </div>
    <div class="content-surface applications-surface">
      <el-alert v-if="errorMessage" class="page-error" type="error" :title="errorMessage" show-icon :closable="false">
        <template #default>
          <el-button size="small" type="danger" plain @click="load">重试</el-button>
        </template>
      </el-alert>
      <div class="applications-table-area desktop-only">
        <el-table class="applications-table" height="100%" v-loading="loading" :data="list" empty-text="暂无投递记录">
          <el-table-column label="岗位" min-width="200">
            <template #default="{ row }">
              <el-link type="primary" @click="router.push(`/jobs/${row.job_id}`)">{{ row.job_title }}</el-link>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="180">
            <template #default="{ row }">
              <el-tag :type="statusType[row.status] || 'info'">{{ statusText[row.status] || '未知' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="applied_time_display" label="投递时间" width="170" />
          <el-table-column label="轮次" width="90">
            <template #default="{ row }">第 {{ row.round_no }} 轮</template>
          </el-table-column>
          <el-table-column label="流程" width="110">
            <template #default="{ row }">
              <el-tag :type="row.is_current === 1 ? 'success' : 'info'">{{ row.is_current === 1 ? '当前投递' : '历史投递' }}</el-tag>
            </template>
          </el-table-column>
        </el-table>
        <div v-if="!loading && !errorMessage && list.length === 0" class="empty-actions applications-empty-actions">
          <el-button type="primary" @click="router.push('/jobs')">去看看岗位</el-button>
        </div>
      </div>
      <!-- Mobile application cards -->
      <div class="mobile-card-list mobile-only">
        <el-empty v-if="!loading && list.length === 0" description="暂无投递记录">
          <el-button type="primary" @click="router.push('/jobs')">去看看岗位</el-button>
        </el-empty>
        <div v-for="row in list" :key="row.application_id" class="mobile-application-card">
          <div class="mobile-card__header">
            <h3 class="mobile-card__title">
              <el-link type="primary" @click="router.push(`/jobs/${row.job_id}`)">{{ row.job_title }}</el-link>
            </h3>
            <el-tag :type="statusType[row.status] || 'info'" size="small">{{ statusText[row.status] || '未知' }}</el-tag>
          </div>
          <div class="mobile-card__meta">
            <span>{{ row.applied_time_display }}</span>
            <span>第 {{ row.round_no }} 轮</span>
            <el-tag :type="row.is_current === 1 ? 'success' : 'info'" size="small">{{ row.is_current === 1 ? '当前投递' : '历史投递' }}</el-tag>
          </div>
        </div>
      </div>
      <el-pagination class="applications-pagination" v-model:current-page="query.page" v-model:page-size="query.page_size" layout="total, prev, pager, next" :total="total" @current-change="load" />
    </div>
  </section>
</template>
