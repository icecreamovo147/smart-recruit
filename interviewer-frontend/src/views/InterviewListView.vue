<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Search } from '@element-plus/icons-vue'
import { listMyInterviews } from '@/api/interview'
import type { InterviewSchedule } from '@/types/domain'
import { INTERVIEW_STATUS_LABEL, INTERVIEW_STATUS_TYPE, INTERVIEW_MODE_LABEL } from '@/types/domain'

const router = useRouter()
const loading = ref(false)
const interviews = ref<InterviewSchedule[]>([])
const activeFilter = ref('all')
const keyword = ref('')

// Pagination
const currentPage = ref(1)
const pageSize = ref(20)

const filters = [
  { key: 'all', label: '全部' },
  { key: 'scheduled', label: '待面试' },
  { key: 'completed', label: '已完成' },
  { key: 'cancelled', label: '已取消' },
]

const filteredInterviews = computed(() => {
  let list = interviews.value
  if (activeFilter.value !== 'all') {
    list = list.filter((i) => i.status === activeFilter.value)
  }
  if (keyword.value.trim()) {
    const kw = keyword.value.trim().toLowerCase()
    list = list.filter((i) =>
      (i.candidate_name || '').toLowerCase().includes(kw) ||
      (i.job_title || '').toLowerCase().includes(kw) ||
      (i.title || '').toLowerCase().includes(kw),
    )
  }
  return list
})

const pagedInterviews = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredInterviews.value.slice(start, start + pageSize.value)
})

// Reset to page 1 when filter or keyword changes
watch([activeFilter, keyword], () => {
  currentPage.value = 1
})

const formatDateTime = (iso: string): string => {
  if (!iso) return '-'
  const d = new Date(iso)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

const loadData = async () => {
  loading.value = true
  try {
    const data = await listMyInterviews()
    interviews.value = data.list || []
  } catch {
    // Handled by request interceptor
  } finally {
    loading.value = false
  }
}

// Dynamic table max-height
const listRef = ref<HTMLElement | null>(null)
const tableMaxHeight = ref<number>(400)
let resizeObserver: ResizeObserver | null = null

const calcTableHeight = () => {
  nextTick(() => {
    const el = listRef.value
    if (!el) return
    const toolbar = el.querySelector('.list-toolbar') as HTMLElement | null
    const pagination = el.querySelector('.pagination-wrapper') as HTMLElement | null
    const toolbarH = toolbar ? toolbar.offsetHeight : 0
    const paginationH = pagination ? pagination.offsetHeight : 0
    // gap between children (2 × 16px) + extra buffer
    const available = el.clientHeight - toolbarH - paginationH - 48
    tableMaxHeight.value = Math.max(available, 120)
  })
}

onMounted(() => {
  loadData()
  if (listRef.value) {
    resizeObserver = new ResizeObserver(calcTableHeight)
    resizeObserver.observe(listRef.value)
  }
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
})
</script>

<template>
  <div ref="listRef" class="interview-list">
    <div class="list-toolbar">
      <div class="filter-tabs">
        <el-radio-group v-model="activeFilter" size="default">
          <el-radio-button v-for="f in filters" :key="f.key" :value="f.key">{{ f.label }}</el-radio-button>
        </el-radio-group>
      </div>
      <el-input
        v-model="keyword"
        placeholder="搜索候选人或岗位"
        clearable
        style="width: 240px"
        :prefix-icon="Search"
      />
    </div>

    <el-table
      v-if="filteredInterviews.length > 0 || loading"
      v-loading="loading"
      :data="pagedInterviews"
      :height="tableMaxHeight"
      style="width: 100%"
      row-class-name="interview-row"
      @row-click="(row: InterviewSchedule) => router.push(`/interviews/${row.interview_id}`)"
    >
      <el-table-column label="候选人" prop="candidate_name" min-width="120" />
      <el-table-column label="岗位" prop="job_title" min-width="140" />
      <el-table-column label="面试标题" prop="title" min-width="160" show-overflow-tooltip />
      <el-table-column label="轮次" width="80" align="center">
        <template #default="{ row }">
          {{ row.round_no }}
        </template>
      </el-table-column>
      <el-table-column label="面试时间" min-width="160">
        <template #default="{ row }">
          {{ formatDateTime(row.scheduled_at) }}
        </template>
      </el-table-column>
      <el-table-column label="方式" width="100">
        <template #default="{ row }">
          {{ INTERVIEW_MODE_LABEL[row.mode] || row.mode }}
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="(INTERVIEW_STATUS_TYPE[row.status] || 'info') as any" size="small" effect="light">
            {{ INTERVIEW_STATUS_LABEL[row.status] || row.status }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" link size="small" @click.stop="router.push(`/interviews/${row.interview_id}`)">
            查看详情
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div v-if="!loading && filteredInterviews.length === 0" class="empty-wrapper">
      <el-empty description="暂无面试记录" />
    </div>

    <div v-if="filteredInterviews.length > 0" class="pagination-wrapper">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :total="filteredInterviews.length"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        background
        small
      />
    </div>
  </div>
</template>

<style scoped>
.interview-list {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.list-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  flex-wrap: wrap;
  gap: 12px;
  flex-shrink: 0;
}

.filter-tabs {
  display: flex;
  gap: 8px;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
  flex-shrink: 0;
}

.empty-wrapper {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200px;
}

:deep(.interview-row) {
  cursor: pointer;
}
:deep(.interview-row:hover td) {
  background: var(--surface-secondary) !important;
}
</style>
