<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { getUsageStats, getUsageTrend } from '@/api/admin'
import type {
  UsageStatsDimension,
  UsageStatsItem,
  UsageStatsSummary,
  UsageTrendPoint,
} from '@/api/admin'
import * as echarts from 'echarts'
import { toShanghaiRFC3339 } from '@shared/utils/format'

// ── Constants ────────────────────────────────────────────────────────────

const RANK_DIMENSION_OPTIONS: { label: string; value: UsageStatsDimension }[] = [
  { label: '按模型', value: 'model' },
  { label: '按用户', value: 'user' },
  { label: '按会话', value: 'session' },
]

const GRANULARITY_OPTIONS = [
  { label: '按天', value: 'day' },
  { label: '按周', value: 'week' },
  { label: '按月', value: 'month' },
]

const SERVICE_TYPE_LABEL: Record<string, string> = {
  ai_chat: 'AI 对话',
  ai_analyze: 'AI 分析',
  oss_presign: '上传签名',
  oss_confirm: '上传确认',
}

// ── State ────────────────────────────────────────────────────────────────

const statsLoading = ref(false)
const summary = ref<UsageStatsSummary | null>(null)
const trendPoints = ref<UsageTrendPoint[]>([])
const providerRows = ref<UsageStatsItem[]>([])
const serviceTypeRows = ref<UsageStatsItem[]>([])
const rankRows = ref<UsageStatsItem[]>([])
const activeRankDimension = ref<UsageStatsDimension>('model')
const activeGranularity = ref<'day' | 'week' | 'month'>('day')
const statsDateRange = ref<[string, string] | null>(null)

const toNum = (v: unknown): number => {
  if (typeof v === 'number' && Number.isFinite(v)) return v
  if (typeof v === 'string' && v.trim() !== '') {
    const n = Number(v)
    return Number.isFinite(n) ? n : 0
  }
  return 0
}

/** Normalize API item (snake_case / camelCase / int64-as-string). */
const normalizeStatsItem = (raw: any): UsageStatsItem => ({
  name: String(raw?.name ?? raw?.Name ?? '').trim() || '(空)',
  total_tokens: toNum(raw?.total_tokens ?? raw?.totalTokens),
  call_count: toNum(raw?.call_count ?? raw?.callCount),
  avg_cost_ms: toNum(raw?.avg_cost_ms ?? raw?.avgCostMs),
  estimated_cost: toNum(raw?.estimated_cost ?? raw?.estimatedCost),
  success_count: toNum(raw?.success_count ?? raw?.successCount),
  failed_count: toNum(raw?.failed_count ?? raw?.failedCount),
})

const normalizeSummary = (raw: any, fallbackRows: UsageStatsItem[] = []): UsageStatsSummary => {
  const fromApi: UsageStatsSummary = {
    total_tokens: toNum(raw?.total_tokens ?? raw?.totalTokens),
    call_count: toNum(raw?.call_count ?? raw?.callCount),
    success_count: toNum(raw?.success_count ?? raw?.successCount),
    failed_count: toNum(raw?.failed_count ?? raw?.failedCount),
    avg_cost_ms: toNum(raw?.avg_cost_ms ?? raw?.avgCostMs),
    estimated_cost: toNum(raw?.estimated_cost ?? raw?.estimatedCost),
    success_rate: toNum(raw?.success_rate ?? raw?.successRate),
  }
  if (fromApi.call_count > 0) {
    if (fromApi.success_rate <= 0 && fromApi.success_count > 0) {
      fromApi.success_rate = (fromApi.success_count * 100) / fromApi.call_count
    }
    return fromApi
  }
  // Fallback: aggregate from dimension rows when summary missing/zero (old backend or parse miss).
  if (fallbackRows.length === 0) return fromApi
  const call_count = fallbackRows.reduce((s, r) => s + toNum(r.call_count), 0)
  const total_tokens = fallbackRows.reduce((s, r) => s + toNum(r.total_tokens), 0)
  const success_count = fallbackRows.reduce((s, r) => s + toNum(r.success_count), 0)
  const failed_count = fallbackRows.reduce((s, r) => s + toNum(r.failed_count), 0)
  const weightedMs = fallbackRows.reduce((s, r) => s + toNum(r.avg_cost_ms) * toNum(r.call_count), 0)
  return {
    total_tokens,
    call_count,
    success_count,
    failed_count,
    avg_cost_ms: call_count > 0 ? weightedMs / call_count : 0,
    estimated_cost: fallbackRows.reduce((s, r) => s + toNum(r.estimated_cost), 0),
    success_rate: call_count > 0 ? (success_count * 100) / call_count : 0,
  }
}

const successRateText = computed(() => {
  const rate = toNum(summary.value?.success_rate)
  return `${rate.toFixed(1)}%`
})

const displayName = (name: string, dimension?: UsageStatsDimension): string => {
  if (dimension === 'service_type') {
    return SERVICE_TYPE_LABEL[name] || name || '(空)'
  }
  return name || '(空)'
}

const successRateOf = (row: UsageStatsItem): number => {
  const total = toNum(row.call_count)
  if (total <= 0) return 0
  return (toNum(row.success_count) * 100) / total
}

const getDefaultStatsRange = (): [string, string] => {
  const end = new Date()
  const start = new Date(end.getTime() - 30 * 24 * 60 * 60 * 1000)
  return [toShanghaiRFC3339(start), toShanghaiRFC3339(end)]
}

const resolveRange = (): { startTime: string; endTime: string } => {
  if (statsDateRange.value) {
    return { startTime: statsDateRange.value[0], endTime: statsDateRange.value[1] }
  }
  const [s, e] = getDefaultStatsRange()
  return { startTime: s, endTime: e }
}

const loadStats = async () => {
  statsLoading.value = true
  try {
    const { startTime, endTime } = resolveRange()
    const [rankResp, providerResp, serviceResp, trendResp] = await Promise.all([
      getUsageStats({
        start_time: startTime,
        end_time: endTime,
        dimension: activeRankDimension.value,
      }),
      getUsageStats({
        start_time: startTime,
        end_time: endTime,
        dimension: 'provider',
      }),
      getUsageStats({
        start_time: startTime,
        end_time: endTime,
        dimension: 'service_type',
      }),
      getUsageTrend({
        start_time: startTime,
        end_time: endTime,
        granularity: activeGranularity.value,
      }),
    ])
    const providers = (providerResp.list || []).map(normalizeStatsItem)
    const services = (serviceResp.list || []).map(normalizeStatsItem)
    const ranks = (rankResp.list || []).map(normalizeStatsItem)
    providerRows.value = providers
    serviceTypeRows.value = services
    rankRows.value = ranks.slice(0, 10)
    // Prefer API summary; fall back to provider-dimension totals (covers full call volume).
    summary.value = normalizeSummary(
      rankResp.summary || providerResp.summary || serviceResp.summary,
      providers.length > 0 ? providers : services.length > 0 ? services : ranks,
    )
    trendPoints.value = (trendResp.list || []).map((p: any) => ({
      date: String(p?.date ?? ''),
      total_tokens: toNum(p?.total_tokens ?? p?.totalTokens),
      call_count: toNum(p?.call_count ?? p?.callCount),
      avg_cost_ms: toNum(p?.avg_cost_ms ?? p?.avgCostMs),
      estimated_cost: toNum(p?.estimated_cost ?? p?.estimatedCost),
    }))
  } catch {
    ElMessage.error('加载统计数据失败')
  } finally {
    statsLoading.value = false
  }
}

const handleStatsDateChange = () => {
  loadStats()
}

const handleRankDimensionChange = (dim: UsageStatsDimension) => {
  activeRankDimension.value = dim
  loadStats()
}

const handleGranularityChange = (g: 'day' | 'week' | 'month') => {
  activeGranularity.value = g
  reloadTrend()
}

const reloadTrend = async () => {
  try {
    const { startTime, endTime } = resolveRange()
    const trendResp = await getUsageTrend({
      start_time: startTime,
      end_time: endTime,
      granularity: activeGranularity.value,
    })
    trendPoints.value = trendResp.list || []
  } catch {
    ElMessage.error('加载趋势数据失败')
  }
}

// ── Charts ───────────────────────────────────────────────────────────────

const trendChartRef = ref<HTMLElement | null>(null)
const providerChartRef = ref<HTMLElement | null>(null)
const serviceChartRef = ref<HTMLElement | null>(null)
let trendChart: echarts.ECharts | null = null
let providerChart: echarts.ECharts | null = null
let serviceChart: echarts.ECharts | null = null

const renderTrendChart = () => {
  if (!trendChartRef.value) return
  if (!trendChart) trendChart = echarts.init(trendChartRef.value)

  const dates = trendPoints.value.map((p) => String(p.date || '').slice(0, 10))
  const tokens = trendPoints.value.map((p) => toNum(p.total_tokens))
  const calls = trendPoints.value.map((p) => toNum(p.call_count))
  const latency = trendPoints.value.map((p) => Number(toNum(p.avg_cost_ms).toFixed(1)))
  const cost = trendPoints.value.map((p) => Number(toNum(p.estimated_cost).toFixed(6)))

  trendChart.setOption({
    tooltip: { trigger: 'axis' },
    legend: {
      data: ['Token 消耗', '调用次数', '平均耗时(ms)', '花费估算($)'],
      top: 0,
      type: 'scroll',
    },
    grid: { left: '3%', right: '6%', bottom: '3%', top: 48, containLabel: true },
    xAxis: {
      type: 'category',
      data: dates,
      axisLabel: { rotate: 45, fontSize: 11 },
    },
    yAxis: [
      { type: 'value', name: 'Token / 次数', nameTextStyle: { fontSize: 11 } },
      { type: 'value', name: '耗时 / 花费', nameTextStyle: { fontSize: 11 } },
    ],
    series: [
      {
        name: 'Token 消耗',
        type: 'bar',
        data: tokens,
        itemStyle: { color: '#409eff' },
      },
      {
        name: '调用次数',
        type: 'line',
        data: calls,
        itemStyle: { color: '#67c23a' },
        smooth: true,
      },
      {
        name: '平均耗时(ms)',
        type: 'line',
        yAxisIndex: 1,
        data: latency,
        itemStyle: { color: '#e6a23c' },
        smooth: true,
      },
      {
        name: '花费估算($)',
        type: 'line',
        yAxisIndex: 1,
        data: cost,
        itemStyle: { color: '#f56c6c' },
        smooth: true,
      },
    ],
  }, true)
}

const renderPie = (
  el: HTMLElement | null,
  chart: echarts.ECharts | null,
  rows: UsageStatsItem[],
  dimension: UsageStatsDimension,
): echarts.ECharts | null => {
  if (!el) return chart
  if (chart) {
    chart.dispose()
    chart = null
  }
  chart = echarts.init(el)
  const pieData = rows
    .map((item) => ({
      name: displayName(item.name, dimension),
      value: Math.max(0, toNum(item.call_count)),
    }))
    .filter((item) => item.value > 0 || item.name)
  chart.setOption({
    tooltip: {
      trigger: 'item',
      formatter: (params: any) => {
        const row = rows.find((r) => displayName(r.name, dimension) === params.name)
        const rate = row ? successRateOf(row) : 0
        return `${params.name}<br/>调用 ${params.value}（${params.percent}%）<br/>成功率 ${rate.toFixed(1)}%`
      },
    },
    series: [
      {
        type: 'pie',
        radius: ['34%', '62%'],
        center: ['50%', '52%'],
        data: pieData.length > 0 ? pieData : [{ name: '暂无数据', value: 0 }],
        label: {
          formatter: (params: any) => `${params.name}\n${params.percent}%`,
          fontSize: 11,
        },
      },
    ],
  }, true)
  return chart
}

const renderDistributionCharts = () => {
  providerChart = renderPie(providerChartRef.value, providerChart, providerRows.value, 'provider')
  serviceChart = renderPie(serviceChartRef.value, serviceChart, serviceTypeRows.value, 'service_type')
}

watch(trendPoints, () => renderTrendChart(), { deep: true })
watch([providerRows, serviceTypeRows], () => renderDistributionCharts(), { deep: true })

const handleResize = () => {
  trendChart?.resize()
  providerChart?.resize()
  serviceChart?.resize()
}

onMounted(() => {
  loadStats()
  window.addEventListener('resize', handleResize)
})
</script>

<template>
  <div class="console-page console-page--fill usage-stats-view">
    <div class="workspace-surface">
      <div class="workspace-surface__header">
        <div class="workspace-surface__header-copy">
          <p class="console-eyebrow">USAGE STATS</p>
          <h2 class="console-title">使用统计</h2>
          <p class="console-description">
            汇总第三方服务调用的用量、成功率、耗时与花费，并按供应商 / 服务类型 / 模型拆分。
          </p>
        </div>
        <div class="workspace-surface__header-actions">
          <el-button :icon="Refresh" :loading="statsLoading" @click="loadStats">刷新</el-button>
        </div>
      </div>

      <div class="workspace-surface__divider"></div>

      <div class="workspace-surface__toolbar">
        <div class="workspace-surface__filters">
          <el-date-picker
            v-model="statsDateRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
            style="width: 360px"
            @change="handleStatsDateChange"
          />
        </div>
        <div class="workspace-surface__actions">
          <el-button @click="() => { statsDateRange = null; loadStats() }">重置</el-button>
        </div>
      </div>

      <div class="workspace-surface__body usage-stats-body">
        <el-row :gutter="12" class="stats-cards" v-loading="statsLoading">
          <el-col :xs="12" :sm="8" :md="4">
            <div class="stats-card">
              <div class="stats-card-label">总 Token</div>
              <div class="stats-card-value">{{ toNum(summary?.total_tokens).toLocaleString() }}</div>
            </div>
          </el-col>
          <el-col :xs="12" :sm="8" :md="4">
            <div class="stats-card">
              <div class="stats-card-label">总调用</div>
              <div class="stats-card-value">{{ toNum(summary?.call_count).toLocaleString() }}</div>
            </div>
          </el-col>
          <el-col :xs="12" :sm="8" :md="4">
            <div class="stats-card">
              <div class="stats-card-label">成功率</div>
              <div class="stats-card-value stats-card-value--ok">{{ successRateText }}</div>
            </div>
          </el-col>
          <el-col :xs="12" :sm="8" :md="4">
            <div class="stats-card">
              <div class="stats-card-label">失败次数</div>
              <div class="stats-card-value stats-card-value--danger">{{ toNum(summary?.failed_count).toLocaleString() }}</div>
            </div>
          </el-col>
          <el-col :xs="12" :sm="8" :md="4">
            <div class="stats-card">
              <div class="stats-card-label">平均耗时 (ms)</div>
              <div class="stats-card-value">{{ toNum(summary?.avg_cost_ms).toFixed(1) }}</div>
            </div>
          </el-col>
          <el-col :xs="12" :sm="8" :md="4">
            <div class="stats-card">
              <div class="stats-card-label">花费估算 ($)</div>
              <div class="stats-card-value">{{ toNum(summary?.estimated_cost).toFixed(4) }}</div>
            </div>
          </el-col>
        </el-row>

        <div class="chart-panel chart-panel--wide">
          <div class="chart-header">
            <span class="chart-header__title">调用趋势</span>
            <el-radio-group
              :model-value="activeGranularity"
              size="small"
              @change="(v: string | number | boolean | undefined) => handleGranularityChange(String(v) as 'day' | 'week' | 'month')"
            >
              <el-radio-button
                v-for="opt in GRANULARITY_OPTIONS"
                :key="opt.value"
                :value="opt.value"
              >
                {{ opt.label }}
              </el-radio-button>
            </el-radio-group>
          </div>
          <div ref="trendChartRef" class="chart-container chart-container--tall" v-loading="statsLoading" />
        </div>

        <el-row :gutter="16" class="stats-charts">
          <el-col :span="12">
            <div class="chart-panel">
              <div class="chart-header">
                <span class="chart-header__title">供应商占比（按调用次数）</span>
              </div>
              <div ref="providerChartRef" class="chart-container" v-loading="statsLoading" />
            </div>
          </el-col>
          <el-col :span="12">
            <div class="chart-panel">
              <div class="chart-header">
                <span class="chart-header__title">服务类型占比（按调用次数）</span>
              </div>
              <div ref="serviceChartRef" class="chart-container" v-loading="statsLoading" />
            </div>
          </el-col>
        </el-row>

        <div class="chart-panel chart-panel--wide">
          <div class="chart-header">
            <span class="chart-header__title">排行榜 Top 10</span>
            <el-radio-group
              :model-value="activeRankDimension"
              size="small"
              @change="(v: string | number | boolean | undefined) => handleRankDimensionChange(String(v) as UsageStatsDimension)"
            >
              <el-radio-button
                v-for="opt in RANK_DIMENSION_OPTIONS"
                :key="opt.value"
                :value="opt.value"
              >
                {{ opt.label }}
              </el-radio-button>
            </el-radio-group>
          </div>
          <el-table
            :data="rankRows"
            v-loading="statsLoading"
            class="console-table"
            stripe
            empty-text="暂无数据"
          >
            <el-table-column type="index" label="#" width="50" />
            <el-table-column prop="name" label="名称" min-width="160" show-overflow-tooltip>
              <template #default="{ row }">
                {{ displayName(row.name, activeRankDimension) }}
              </template>
            </el-table-column>
            <el-table-column prop="call_count" label="调用" width="90" align="center">
              <template #default="{ row }">{{ toNum(row.call_count).toLocaleString() }}</template>
            </el-table-column>
            <el-table-column prop="total_tokens" label="Token" width="110" align="center">
              <template #default="{ row }">{{ toNum(row.total_tokens).toLocaleString() }}</template>
            </el-table-column>
            <el-table-column prop="estimated_cost" label="供应商成本（元）" width="140" align="center">
              <template #default="{ row }">{{ toNum(row.estimated_cost).toFixed(4) }}</template>
            </el-table-column>
            <el-table-column prop="avg_cost_ms" label="平均耗时" width="110" align="center">
              <template #default="{ row }">{{ toNum(row.avg_cost_ms).toFixed(1) }} ms</template>
            </el-table-column>
            <el-table-column label="成功 / 失败" width="120" align="center">
              <template #default="{ row }">
                {{ toNum(row.success_count) }} / {{ toNum(row.failed_count) }}
              </template>
            </el-table-column>
            <el-table-column label="成功率" width="100" align="center">
              <template #default="{ row }">
                <el-tag
                  size="small"
                  :type="successRateOf(row) >= 95 ? 'success' : successRateOf(row) >= 80 ? 'warning' : 'danger'"
                  effect="light"
                >
                  {{ successRateOf(row).toFixed(1) }}%
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.usage-stats-view {
  min-height: 0;
}

.usage-stats-body {
  padding: 16px 24px 24px;
  overflow: auto;
}

.stats-cards {
  margin-bottom: 16px;
}

.stats-card {
  text-align: center;
  padding: 16px 10px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
  box-shadow: var(--admin-console-card-shadow);
  min-height: 92px;
  margin-bottom: 12px;
}

.stats-card-label {
  font-size: 12px;
  color: var(--el-text-color-secondary, var(--text-muted));
  margin-bottom: 8px;
}

.stats-card-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--el-color-primary, var(--brand));
  font-family: 'SF Mono', 'Menlo', 'Monaco', monospace;
  line-height: 1.2;
}

.stats-card-value--ok {
  color: var(--el-color-success);
}

.stats-card-value--danger {
  color: var(--el-color-danger);
}

.stats-charts {
  margin-bottom: 16px;
}

.chart-panel {
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
  box-shadow: var(--admin-console-card-shadow);
  padding: 14px 16px 16px;
  margin-bottom: 16px;
  min-height: 340px;
}

.chart-panel--wide {
  min-height: auto;
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.chart-header__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.chart-container {
  width: 100%;
  height: 260px;
}

.chart-container--tall {
  height: 320px;
}
</style>
