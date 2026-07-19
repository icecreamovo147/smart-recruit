<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'
import { listMyInterviews } from '@/api/interview'
import {
  INTERVIEW_MODE_LABEL,
  INTERVIEW_STATUS_LABEL,
  INTERVIEW_STATUS_TYPE,
  type InterviewSchedule,
} from '@/types/domain'

const router = useRouter()
const loading = ref(false)
const status = ref('')
const keyword = ref('')
const interviews = ref<InterviewSchedule[]>([])

const filtered = computed(() => {
  const normalized = keyword.value.trim().toLowerCase()
  return interviews.value.filter((item) => {
    if (status.value && item.status !== status.value) return false
    if (!normalized) return true
    return [item.candidate_name, item.job_title, item.title]
      .some((value) => (value || '').toLowerCase().includes(normalized))
  })
})

const formatDate = (value: string) => value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'

const load = async () => {
  loading.value = true
  try {
    const response = await listMyInterviews()
    interviews.value = response.list || []
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="my-interviews workspace-surface">
    <header class="page-header workspace-surface__header my-interviews__header">
      <div class="workspace-surface__header-copy">
        <h1 class="console-title">我的面试</h1>
        <p class="console-description">查看当前企业分配给你的面试，并在面试结束后提交评价。</p>
      </div>
    </header>

    <div class="workspace-surface__divider"></div>

    <div class="workspace-surface__toolbar my-interviews__toolbar">
      <div class="workspace-surface__filters">
        <el-radio-group v-model="status">
          <el-radio-button value="">全部</el-radio-button>
          <el-radio-button value="scheduled">待面试</el-radio-button>
          <el-radio-button value="completed">已完成</el-radio-button>
          <el-radio-button value="cancelled">已取消</el-radio-button>
        </el-radio-group>
      </div>
      <div class="workspace-surface__actions">
        <el-input v-model="keyword" :prefix-icon="Search" clearable placeholder="搜索候选人、岗位或面试标题" />
      </div>
    </div>

    <div class="workspace-surface__body my-interviews__body">
      <el-table v-loading="loading" :data="filtered" empty-text="当前企业暂无面试任务" @row-click="(row: InterviewSchedule) => router.push(`/hr/my-interviews/${row.interview_id}`)">
        <el-table-column prop="candidate_name" label="候选人" min-width="120" />
        <el-table-column prop="job_title" label="岗位" min-width="150" />
        <el-table-column prop="title" label="面试" min-width="160" show-overflow-tooltip />
        <el-table-column label="时间" min-width="180">
          <template #default="{ row }">{{ formatDate(row.scheduled_at) }}</template>
        </el-table-column>
        <el-table-column label="方式" width="110">
          <template #default="{ row }">{{ INTERVIEW_MODE_LABEL[row.mode] || row.mode }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="INTERVIEW_STATUS_TYPE[row.status] || 'info'">{{ INTERVIEW_STATUS_LABEL[row.status] || row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click.stop="router.push(`/hr/my-interviews/${row.interview_id}`)">查看</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </section>
</template>

<style scoped>
.my-interviews__header.page-header {
  margin: 0;
  border: 0;
  border-radius: 0;
}
.my-interviews__toolbar .el-input { width: min(360px, 45vw); }
.my-interviews__body { padding-top: 0; }
:deep(.el-table__row) { cursor: pointer; }
@media (max-width: 720px) {
  .my-interviews { overflow: visible; }
  .my-interviews__toolbar { align-items: stretch; flex-direction: column; }
  .my-interviews__toolbar .workspace-surface__actions { width: 100%; }
  .my-interviews__toolbar .el-input { width: 100%; }
  .my-interviews__body { overflow: visible; }
}
</style>
