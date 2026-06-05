<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, BarChart, PieChart } from 'echarts/charts'
import {
  TitleComponent, TooltipComponent, LegendComponent, GridComponent,
} from 'echarts/components'
import { getDashboardReport, getFunnelReport, getTimeInStageReport, getInterviewOfferMetrics } from '@/api/analytics'
import type { DashboardReport, FunnelReport, TimeInStageReport, InterviewOfferMetrics } from '@/types/analytics'

use([CanvasRenderer, LineChart, BarChart, PieChart, TitleComponent, TooltipComponent, LegendComponent, GridComponent])

const loading = ref(true)
const error = ref('')
const activeTab = ref('overview')

const dashboard = ref<DashboardReport | null>(null)
const funnel = ref<FunnelReport | null>(null)
const timeInStage = ref<TimeInStageReport | null>(null)
const metrics = ref<InterviewOfferMetrics | null>(null)

const jobIdFilter = ref('')
const dateRange = ref<[string, string] | null>(null)

const fetchAll = async () => {
  loading.value = true
  error.value = ''
  try {
    const jobId = jobIdFilter.value ? Number(jobIdFilter.value) : undefined
    const params: any = {}
    if (jobId && jobId > 0) params.job_id = jobId
    if (dateRange.value) {
      params.start_date = dateRange.value[0]
      params.end_date = dateRange.value[1]
    }

    const [d, f, t, m] = await Promise.all([
      getDashboardReport(),
      getFunnelReport(params),
      getTimeInStageReport(params),
      getInterviewOfferMetrics(params),
    ])
    dashboard.value = d
    funnel.value = f
    timeInStage.value = t
    metrics.value = m
  } catch (e: unknown) {
    error.value = (e as { message?: string })?.message || '加载分析数据失败'
  } finally {
    loading.value = false
  }
}

onMounted(fetchAll)

// Dashboard trend chart
const trendOption = computed(() => {
  const trend = dashboard.value?.trend
  if (!trend || trend.length === 0) return null
  return {
    title: { text: '近7日投递趋势', left: 'center' },
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: { type: 'category', data: trend.map(t => t.date.slice(5)) },
    yAxis: { type: 'value', minInterval: 1 },
    series: [{ type: 'line', data: trend.map(t => t.applications), smooth: true, areaStyle: { opacity: 0.3 } }],
  }
})

// Funnel chart
const funnelOption = computed(() => {
  const stages = funnel.value?.stages
  if (!stages || stages.length === 0) return null
  return {
    title: { text: '招聘漏斗', left: 'center' },
    tooltip: {
      trigger: 'item',
      formatter: (params: any) => `${params.name}: ${params.value} (${params.percent}%)`,
    },
    series: [{
      type: 'funnel',
      left: '10%',
      right: '10%',
      top: 60,
      bottom: 20,
      minSize: '10%',
      maxSize: '100%',
      sort: 'descending',
      gap: 2,
      label: { show: true, formatter: '{b}: {c}' },
      itemStyle: { borderColor: '#fff', borderWidth: 1 },
      data: stages.map(s => ({ name: s.stage_label, value: s.count })),
    }],
  }
})

// Time-in-stage bar chart
const stageDurationOption = computed(() => {
  const durations = timeInStage.value?.durations
  if (!durations || durations.length === 0) return null
  return {
    title: { text: '各阶段平均耗时(小时)', left: 'center' },
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '15%', bottom: '3%', containLabel: true },
    xAxis: { type: 'value', name: '小时' },
    yAxis: { type: 'category', data: durations.map(d => d.stage_label).reverse() },
    series: [{
      type: 'bar',
      data: durations.map(d => Math.round(d.avg_hours * 10) / 10).reverse(),
      itemStyle: { color: '#409EFF' },
    }],
  }
})

// KPI cards from dashboard (flat response — fields at top level)
const kpiCards = computed(() => {
  const d = dashboard.value
  if (!d) return []
  return [
    { label: '在招岗位', value: d.online_jobs, color: '#409EFF' },
    { label: '已下线岗位', value: d.offline_jobs, color: '#909399' },
    { label: '候选人数', value: d.total_applications, color: '#67C23A' },
    { label: '今日新增', value: d.today_applications, color: '#E6A23C' },
    { label: '待处理', value: d.pending_actions, color: '#F56C6C' },
  ]
})

const interviewPassRate = computed(() => {
  const m = metrics.value
  if (!m) return 0
  return m.completed_interviews > 0 ? Math.round(m.pass_rate * 10) / 10 : 0
})

const offerAcceptRate = computed(() => {
  const m = metrics.value
  if (!m) return 0
  return m.total_offers > 0 ? Math.round(m.acceptance_rate * 10) / 10 : 0
})

const handleRefresh = () => fetchAll()
</script>

<template>
  <div class="analytics-view" v-loading="loading">
    <div v-if="error" class="analytics-error">
      <el-result icon="error" title="数据加载失败" :sub-title="error">
        <template #extra>
          <el-button type="primary" @click="handleRefresh">重新加载</el-button>
        </template>
      </el-result>
    </div>

    <template v-else>
      <el-card shadow="hover" class="filter-bar">
        <el-form :inline="true">
          <el-form-item label="岗位ID">
            <el-input v-model="jobIdFilter" placeholder="筛选岗位" style="width: 140px" clearable />
          </el-form-item>
          <el-form-item label="时间范围">
            <el-date-picker
              v-model="dateRange"
              type="datetimerange"
              range-separator="至"
              start-placeholder="开始"
              end-placeholder="结束"
              value-format="YYYY-MM-DDTHH:mm:ssZ"
              style="width: 360px"
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleRefresh">查询</el-button>
          </el-form-item>
        </el-form>
      </el-card>

      <!-- KPIs -->
      <el-row :gutter="16" class="kpi-row">
        <el-col v-for="card in kpiCards" :key="card.label" :xs="12" :sm="8" :md="4" class="kpi-col">
          <el-card shadow="hover" class="kpi-card">
            <div class="kpi-value" :style="{ color: card.color }">{{ card.value }}</div>
            <div class="kpi-label">{{ card.label }}</div>
          </el-card>
        </el-col>
      </el-row>

      <!-- Tab Navigation -->
      <el-card shadow="hover">
        <template #header>
          <el-radio-group v-model="activeTab" size="small">
            <el-radio-button value="overview">概览</el-radio-button>
            <el-radio-button value="funnel">招聘漏斗</el-radio-button>
            <el-radio-button value="timing">阶段耗时</el-radio-button>
            <el-radio-button value="metrics">面试与Offer</el-radio-button>
          </el-radio-group>
        </template>

        <!-- Overview Tab -->
        <div v-if="activeTab === 'overview'">
          <el-row :gutter="16">
            <el-col :span="24">
              <div v-if="trendOption" style="height: 320px">
                <v-chart :option="trendOption" style="height: 100%" autoresize />
              </div>
              <el-empty v-else description="暂无趋势数据" />
            </el-col>
          </el-row>
          <el-row :gutter="16" style="margin-top: 16px">
            <el-col :span="12">
              <div v-if="funnelOption" style="height: 360px">
                <v-chart :option="funnelOption" style="height: 100%" autoresize />
              </div>
              <el-empty v-else description="暂无漏斗数据" />
            </el-col>
            <el-col :span="12">
              <div v-if="metrics" class="metrics-summary">
                <h4 style="margin: 0 0 12px; text-align: center">关键指标</h4>
                <el-descriptions :column="1" border>
                  <el-descriptions-item label="面试总数">{{ metrics.total_interviews }}</el-descriptions-item>
                  <el-descriptions-item label="面试通过率">{{ interviewPassRate }}%</el-descriptions-item>
                  <el-descriptions-item label="Offer总数">{{ metrics.total_offers }}</el-descriptions-item>
                  <el-descriptions-item label="Offer接受率">{{ offerAcceptRate }}%</el-descriptions-item>
                </el-descriptions>
              </div>
            </el-col>
          </el-row>
        </div>

        <!-- Funnel Tab -->
        <div v-if="activeTab === 'funnel'">
          <el-row :gutter="16">
            <el-col :span="14">
              <div v-if="funnelOption" style="height: 400px">
                <v-chart :option="funnelOption" style="height: 100%" autoresize />
              </div>
              <el-empty v-else description="暂无漏斗数据" />
            </el-col>
            <el-col :span="10">
              <el-table :data="funnel?.stages || []" stripe size="small" max-height="400">
                <el-table-column prop="stage_label" label="阶段" min-width="100" />
                <el-table-column prop="count" label="数量" width="70" align="center" />
                <el-table-column prop="conversion_rate" label="转化率" width="90" align="center">
                  <template #default="{ row }">
                    {{ row.conversion_rate.toFixed(1) }}%
                  </template>
                </el-table-column>
              </el-table>
            </el-col>
          </el-row>
        </div>

        <!-- Timing Tab -->
        <div v-if="activeTab === 'timing'">
          <div v-if="stageDurationOption" style="height: 400px">
            <v-chart :option="stageDurationOption" style="height: 100%" autoresize />
          </div>
          <el-empty v-else description="暂无阶段耗时数据" />
          <el-table :data="timeInStage?.durations || []" stripe size="small" style="margin-top: 16px">
            <el-table-column prop="stage_label" label="阶段" min-width="120" />
            <el-table-column label="平均耗时" width="120" align="center">
              <template #default="{ row }">
                {{ row.avg_hours ? Math.round(row.avg_hours * 10) / 10 + ' 小时' : '-' }}
              </template>
            </el-table-column>
            <el-table-column prop="transition_count" label="统计数量" width="100" align="center" />
          </el-table>
        </div>

        <!-- Metrics Tab -->
        <div v-if="activeTab === 'metrics'">
          <el-row :gutter="24">
            <el-col :span="12">
              <el-card shadow="never">
                <template #header>面试指标</template>
                <el-descriptions :column="1" border>
                  <el-descriptions-item label="面试总次数">{{ metrics?.total_interviews || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="已完成面试">{{ metrics?.completed_interviews || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="正面反馈">{{ metrics?.positive_feedbacks || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="面试通过率">
                    <el-tag :type="interviewPassRate >= 50 ? 'success' : 'warning'">
                      {{ interviewPassRate }}%
                    </el-tag>
                  </el-descriptions-item>
                </el-descriptions>
              </el-card>
            </el-col>
            <el-col :span="12">
              <el-card shadow="never">
                <template #header>Offer指标</template>
                <el-descriptions :column="1" border>
                  <el-descriptions-item label="Offer总数">{{ metrics?.total_offers || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="已接受">{{ metrics?.accepted_offers || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="已拒绝">{{ metrics?.rejected_offers || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="Offer接受率">
                    <el-tag :type="offerAcceptRate >= 50 ? 'success' : 'warning'">
                      {{ offerAcceptRate }}%
                    </el-tag>
                  </el-descriptions-item>
                </el-descriptions>
              </el-card>
            </el-col>
          </el-row>
        </div>
      </el-card>
    </template>
  </div>
</template>

<style scoped>
.analytics-view {
  height: 100%;
  overflow-y: auto;
  padding: 16px;
}
.analytics-error {
  padding: 40px 0;
}
.filter-bar {
  margin-bottom: 16px;
}
.filter-bar :deep(.el-form-item) {
  margin-bottom: 0;
}
.kpi-row {
  margin-bottom: 16px;
}
.kpi-col {
  margin-bottom: 12px;
}
.kpi-card :deep(.el-card__body) {
  text-align: center;
  padding: 20px 12px;
}
.kpi-value {
  font-size: 28px;
  font-weight: 700;
}
.kpi-label {
  font-size: 13px;
  color: #909399;
  margin-top: 4px;
}
.metrics-summary {
  padding: 0 8px;
}
</style>
