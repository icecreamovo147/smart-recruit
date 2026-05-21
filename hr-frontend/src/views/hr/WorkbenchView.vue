<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  Briefcase, User, Bell, TrendCharts, DataAnalysis, Clock,
} from '@element-plus/icons-vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, PieChart, BarChart } from 'echarts/charts'
import {
  TitleComponent, TooltipComponent, LegendComponent, GridComponent,
} from 'echarts/components'
import { getDashboardSummary } from '@/api/dashboard'
import type { DashboardSummary } from '@/types/dashboard'

use([CanvasRenderer, LineChart, PieChart, BarChart, TitleComponent, TooltipComponent, LegendComponent, GridComponent])

// Count-up animation: animates from 0 to target over ~600ms.
function useCountUp(getTarget: () => number) {
  const display = ref(0)
  let raf = 0

  const animate = (target: number) => {
    cancelAnimationFrame(raf)
    const start = display.value
    const duration = 600
    const startedAt = performance.now()

    const tick = () => {
      const elapsed = performance.now() - startedAt
      const progress = Math.min(elapsed / duration, 1)
      // ease-out cubic
      const eased = 1 - Math.pow(1 - progress, 3)
      display.value = Math.round(start + (target - start) * eased)
      if (progress < 1) {
        raf = requestAnimationFrame(tick)
      }
    }
    tick()
  }

  watch(getTarget, (val) => animate(val), { immediate: true })

  return display
}

const router = useRouter()
const loading = ref(true)
const error = ref('')
const data = ref<DashboardSummary | null>(null)

const fetchData = async () => {
  loading.value = true
  error.value = ''
  try {
    data.value = await getDashboardSummary()
  } catch (e: unknown) {
    error.value = (e as { message?: string })?.message || '加载工作台数据失败'
  } finally {
    loading.value = false
  }
}

onMounted(fetchData)

// Count-up animation wrappers for each KPI value.
const animatedOnlineJobs = useCountUp(() => data.value?.kpi?.online_jobs ?? 0)
const animatedOfflineJobs = useCountUp(() => data.value?.kpi?.offline_jobs ?? 0)
const animatedTotalApps = useCountUp(() => data.value?.kpi?.total_applications ?? 0)
const animatedTodayApps = useCountUp(() => data.value?.kpi?.today_applications ?? 0)
const animatedPending = useCountUp(() => data.value?.kpi?.pending_actions ?? 0)
const animatedUnread = useCountUp(() => data.value?.kpi?.unread_notifications ?? 0)

const kpiCards = computed(() => {
  const kpi = data.value?.kpi
  if (!kpi) return []
  return [
    { label: '在招岗位', value: animatedOnlineJobs.value, icon: Briefcase, color: '#409EFF' },
    { label: '已下线岗位', value: animatedOfflineJobs.value, icon: Briefcase, color: '#909399' },
    { label: '候选人总数', value: animatedTotalApps.value, icon: User, color: '#67C23A' },
    { label: '今日新增投递', value: animatedTodayApps.value, icon: TrendCharts, color: '#E6A23C' },
    { label: '待处理事项', value: animatedPending.value, icon: Clock, color: '#F56C6C' },
    { label: '未读通知', value: animatedUnread.value, icon: Bell, color: '#909399' },
  ]
})

const emptyDeptData = computed(() => {
  const d = data.value?.job_distribution
  return !d || d.labels.length === 0
})

const emptyStageData = computed(() => {
  const d = data.value?.stage_distribution
  return !d || d.values.reduce((a, b) => a + b, 0) === 0
})

const jobDistOption = computed(() => {
  const d = data.value?.job_distribution
  return {
    title: { text: '部门岗位分布', left: 'center' },
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: { type: 'category', data: d?.labels || [] },
    yAxis: { type: 'value', minInterval: 1 },
    series: [{ type: 'bar', data: d?.values || [], barMaxWidth: 60, itemStyle: { color: '#409EFF' } }],
  }
})

const stageDistOption = computed(() => {
  const d = data.value?.stage_distribution
  const values = d?.values || []
  const total = values.reduce((a, b) => a + b, 0)
  // Filter out zero-value labels.
  const pieData = (d?.labels || [])
    .map((label, i) => ({ name: label, value: values[i] || 0 }))
    .filter((item) => item.value > 0)
  return {
    title: { text: '候选人阶段分布', left: 'center', top: 4 },
    tooltip: { trigger: 'item' },
    legend: { bottom: '0%' },
    series: [{
      type: 'pie',
      center: ['50%', '55%'],
      radius: total > 0 ? ['40%', '65%'] : ['40%', '65%'],
      avoidLabelOverlap: false,
      label: { show: true, formatter: '{b}: {c}' },
      data: pieData,
      itemStyle: {
        color: (params: any) => ['#409EFF', '#67C23A', '#E6A23C', '#F56C6C'][params.dataIndex] || '#409EFF',
      },
    }],
  }
})

const quickLinks = [
  { label: '岗位管理', path: '/hr/jobs', icon: Briefcase },
  { label: 'AI 数据助手', path: '/hr/ai', icon: DataAnalysis },
]

const goTo = (path: string) => router.push(path)

</script>

<template>
  <div class="workbench" v-loading="loading">
    <div v-if="error" class="workbench-error">
      <el-result icon="error" title="数据加载失败" :sub-title="error">
        <template #extra>
          <el-button type="primary" @click="fetchData">重新加载</el-button>
        </template>
      </el-result>
    </div>

    <template v-else>
      <!-- Quick Links (top) -->
      <el-card shadow="hover" class="quick-links-card">
        <template #header>快捷入口</template>
        <div class="quick-links">
          <el-button
            v-for="link in quickLinks"
            :key="link.path"
            :icon="link.icon"
            @click="goTo(link.path)"
          >
            {{ link.label }}
          </el-button>
        </div>
      </el-card>

      <!-- KPI Cards -->
      <el-row :gutter="16" class="kpi-row">
        <el-col v-for="card in kpiCards" :key="card.label" :xs="12" :sm="8" :md="4" class="kpi-col">
          <el-card shadow="hover" class="kpi-card">
            <div class="kpi-inner">
              <div class="kpi-icon" :style="{ backgroundColor: card.color }">
                <el-icon :size="20"><component :is="card.icon" /></el-icon>
              </div>
              <div class="kpi-body">
                <div class="kpi-value">{{ card.value }}</div>
                <div class="kpi-label">{{ card.label }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <!-- Charts Row -->
      <el-row :gutter="16" class="chart-row">
        <el-col :xs="24" :md="12" class="chart-col">
          <el-card shadow="hover">
            <template v-if="emptyDeptData">
              <el-empty description="暂无岗位数据" />
            </template>
            <v-chart v-else :option="jobDistOption" style="height: 320px" autoresize />
          </el-card>
        </el-col>
        <el-col :xs="24" :md="12" class="chart-col">
          <el-card shadow="hover">
            <template v-if="emptyStageData">
              <el-empty description="暂无候选人数据" />
            </template>
            <v-chart v-else :option="stageDistOption" style="height: 320px" autoresize />
          </el-card>
        </el-col>
      </el-row>
    </template>
  </div>
</template>

<style scoped>
.workbench { height: 100%; overflow-y: auto; padding: 16px; }
.workbench-error { padding: 40px 0; }

.quick-links-card { margin-bottom: 16px; }
.quick-links { display: flex; flex-wrap: wrap; gap: 8px; }

.kpi-row { margin-bottom: 16px; }
.kpi-col { margin-bottom: 12px; }

.kpi-card :deep(.el-card__body) { padding: 16px; }
.kpi-inner { display: flex; align-items: center; gap: 12px; }
.kpi-icon {
  width: 44px; height: 44px; border-radius: 10px;
  display: flex; align-items: center; justify-content: center;
  color: #fff; flex-shrink: 0;
}
.kpi-value { font-size: 24px; font-weight: 700; line-height: 1.2; }
.kpi-label { font-size: 12px; color: #909399; margin-top: 2px; }

.chart-row { margin-bottom: 16px; }
.chart-col { margin-bottom: 12px; }

@media (max-width: 768px) {
  .kpi-value { font-size: 20px; }
  .kpi-icon { width: 36px; height: 36px; }
}
</style>
