<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { listJobs } from '@/api/job'
import type { Job, JobQuery } from '@/types/domain'

const router = useRouter()
const loading = ref(false)
const errorMessage = ref('')
const jobs = ref<Job[]>([])
const total = ref(0)
const query = reactive<JobQuery>({ page: 1, page_size: 10, keyword: '' })
let searchTimer: ReturnType<typeof setTimeout> | null = null

const load = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    query.page = Number(query.page) || 1
    query.page_size = Number(query.page_size) || 10
    const data = await listJobs(query)
    jobs.value = data.list || []
    total.value = Number(data.total) || 0
  } catch (error: unknown) {
    errorMessage.value = error instanceof Error ? error.message : '岗位列表加载失败'
  } finally {
    loading.value = false
  }
}

watch(
  () => query.keyword,
  () => {
    query.page = 1
    if (searchTimer) clearTimeout(searchTimer)
    searchTimer = setTimeout(load, 500)
  }
)

onMounted(load)
</script>

<template>
  <section class="job-list-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">正在招聘</h1>
        <p class="page-subtitle">发现最适合你的岗位与团队。</p>
      </div>
      <div class="search-bar">
        <el-input v-model="query.keyword" clearable placeholder="搜索岗位、部门、地点" />
        <el-button type="primary" :loading="loading" @click="load">搜索</el-button>
      </div>
    </div>
    <div class="content-surface job-list-surface" v-loading="loading">
      <el-alert v-if="errorMessage" class="page-error" type="error" :title="errorMessage" show-icon :closable="false">
        <template #default>
          <el-button size="small" type="danger" plain @click="load">重试</el-button>
        </template>
      </el-alert>
      <div class="job-list-scroll">
        <el-empty v-if="!loading && !errorMessage && jobs.length === 0" description="暂无符合条件的岗位" />
        <div v-else class="job-grid">
          <article v-for="job in jobs" :key="job.job_id" class="job-card">
            <div class="job-card__header">
              <div class="job-card__title-group">
                <h2 class="job-card__title">{{ job.title }}</h2>
                <span class="job-card__salary">{{ job.salary_range ? job.salary_range + ' 元/月' : '薪资面议' }}</span>
              </div>
              <div class="job-card__meta">{{ job.department || '未填写部门' }} · {{ job.location || '地点待定' }}</div>
            </div>
            <div class="job-card__footer">
              <el-tag :type="job.status === 1 ? 'success' : 'info'" size="small">{{ job.status === 1 ? '招募中' : '已下架' }}</el-tag>
              <el-button type="primary" size="small" @click="router.push(`/jobs/${job.job_id}`)">查看详情</el-button>
            </div>
          </article>
        </div>
      </div>
      <el-pagination class="job-list-pagination" v-model:current-page="query.page" v-model:page-size="query.page_size" layout="total, prev, pager, next" :total="total" @current-change="load" />
    </div>
  </section>
</template>
