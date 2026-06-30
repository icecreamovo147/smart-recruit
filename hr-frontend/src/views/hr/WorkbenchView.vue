<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  Briefcase, User, Bell, TrendCharts, DataAnalysis, Clock, Monitor, ArrowRight, CircleCheck, Warning,
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
import { useAuthStore } from '@/stores/auth'
import { PERM } from '@/types/domain'

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
const auth = useAuthStore()
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

const username = computed(() => auth.username || 'admin')

const primaryKpiCards = computed(() => {
  const kpi = data.value?.kpi
  if (!kpi) return []
  return [
    { label: '在招岗位', value: animatedOnlineJobs.value, icon: Briefcase, tone: 'blue', desc: '当前可投递岗位' },
    { label: '候选人总数', value: animatedTotalApps.value, icon: User, tone: 'green', desc: '累计收到投递' },
    { label: '今日新增投递', value: animatedTodayApps.value, icon: TrendCharts, tone: 'amber', desc: '今日招聘增量' },
    { label: '待处理事项', value: animatedPending.value, icon: Clock, tone: 'red', desc: '需要及时跟进' },
  ]
})

const secondaryKpiCards = computed(() => {
  const kpi = data.value?.kpi
  if (!kpi) return []
  return [
    { label: '已下线岗位', value: animatedOfflineJobs.value, icon: Briefcase },
    { label: '未读通知', value: animatedUnread.value, icon: Bell },
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
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '4%', top: '10%', bottom: '3%', containLabel: true },
    xAxis: { type: 'category', data: d?.labels || [] },
    yAxis: { type: 'value', minInterval: 1, splitLine: { lineStyle: { color: '#e5e7eb' } } },
    series: [{ type: 'bar', data: d?.values || [], barMaxWidth: 52, itemStyle: { color: '#2563eb', borderRadius: [6, 6, 0, 0] } }],
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
    tooltip: { trigger: 'item' },
    legend: { bottom: '0%', itemWidth: 8, itemHeight: 8 },
    series: [{
      type: 'pie',
      center: ['50%', '46%'],
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

const stageItems = computed(() => {
  const d = data.value?.stage_distribution
  if (!d) return []
  return d.labels
    .map((label, index) => ({ label, value: d.values[index] || 0 }))
    .filter((item) => item.value > 0)
})

const topDepartmentSummary = computed(() => {
  const d = data.value?.job_distribution
  if (!d || d.labels.length === 0) return '暂无部门岗位数据'
  const items = d.labels.map((label, index) => ({ label, value: d.values[index] || 0 }))
  const top = items.reduce((prev, curr) => (curr.value > prev.value ? curr : prev), items[0])
  return top.value > 0 ? `${top.label}岗位需求最集中，共 ${top.value} 个岗位` : '暂无有效岗位分布数据'
})

const actionItems = computed(() => [
  {
    title: '待处理事项',
    value: animatedPending.value,
    desc: animatedPending.value > 0 ? '建议优先处理候选人筛选、面试安排与业务提醒' : '当前没有待处理事项',
    icon: animatedPending.value > 0 ? Warning : CircleCheck,
    path: '/hr/jobs',
    action: '查看岗位',
  },
  {
    title: '未读通知',
    value: animatedUnread.value,
    desc: animatedUnread.value > 0 ? '有新的系统或招聘流程通知等待查看' : '通知已全部查看',
    icon: Bell,
    path: '/hr/workbench',
    action: '刷新概览',
  },
])

const quickLinks = computed(() => {
  const links: { label: string; path: string; icon: any }[] = [
    { label: '岗位管理', path: '/hr/jobs', icon: Briefcase },
    { label: 'AI 数据助手', path: '/hr/ai', icon: DataAnalysis },
    { label: '数据分析', path: '/hr/analytics', icon: DataAnalysis },
  ]
  if (auth.hasPermission(PERM.AUDIT_USAGE_READ)) {
    links.push({ label: '服务审计', path: '/hr/admin/usage-audit', icon: Monitor })
  }
  return links
})

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
      <div class="console-header">
        <div class="console-header__copy">
          <p class="console-eyebrow">HR DASHBOARD</p>
          <h1 class="console-title">工作台</h1>
          <p class="console-description">欢迎回来，{{ username }}。今日招聘运营概览与待处理事项一目了然。</p>
        </div>
      </div>

      <section class="quick-actions" aria-label="快捷入口">
        <span class="section-kicker">快捷入口</span>
        <div class="quick-actions__list">
          <el-button
            v-for="link in quickLinks"
            :key="link.path"
            class="quick-action"
            text
            :icon="link.icon"
            @click="goTo(link.path)"
          >
            {{ link.label }}
            <el-icon class="quick-action__arrow"><ArrowRight /></el-icon>
          </el-button>
        </div>
      </section>

      <section class="metrics-section">
        <div class="metrics-grid">
          <article
            v-for="card in primaryKpiCards"
            :key="card.label"
            class="metric-card"
            :class="`metric-card--${card.tone}`"
          >
            <div class="metric-card__icon">
              <el-icon :size="22"><component :is="card.icon" /></el-icon>
            </div>
            <div class="metric-card__body">
              <div class="metric-card__label">{{ card.label }}</div>
              <div class="metric-card__value">{{ card.value }}</div>
              <div class="metric-card__desc">{{ card.desc }}</div>
            </div>
          </article>
        </div>

        <div class="secondary-metrics">
          <div v-for="card in secondaryKpiCards" :key="card.label" class="secondary-metric">
            <el-icon><component :is="card.icon" /></el-icon>
            <span>{{ card.label }}</span>
            <strong>{{ card.value }}</strong>
          </div>
        </div>
      </section>

      <section class="main-workspace">
        <div class="workspace-left">
          <el-card shadow="never" class="workspace-card overview-card">
            <div class="panel-header">
              <div>
                <h2>招聘运营概览</h2>
                <p>按部门查看当前岗位供给，快速识别招聘资源投入重点。</p>
              </div>
              <el-tag effect="plain">{{ topDepartmentSummary }}</el-tag>
            </div>
            <template v-if="emptyDeptData">
              <el-empty description="暂无岗位数据">
                <el-button type="primary" @click="goTo('/hr/jobs')">去发布岗位</el-button>
              </el-empty>
            </template>
            <v-chart v-else :option="jobDistOption" class="overview-chart" autoresize />
          </el-card>
        </div>

        <div class="workspace-right">
          <el-card shadow="never" class="workspace-card activity-card">
            <div class="panel-header panel-header--compact">
              <div>
                <h2>候选人动态</h2>
                <p>聚合候选人阶段变化与当前推荐处理动作。</p>
              </div>
            </div>

            <template v-if="emptyStageData">
              <el-empty class="actionable-empty" description="当前暂无候选人动态">
                <template #description>
                  <div class="empty-copy">
                    <strong>当前暂无候选人动态</strong>
                    <span>发布岗位后，投递与候选人动态会显示在这里。</span>
                  </div>
                </template>
                <div class="empty-actions">
                  <el-button type="primary" @click="goTo('/hr/jobs')">去发布岗位</el-button>
                  <el-button @click="goTo('/hr/jobs')">查看岗位管理</el-button>
                </div>
              </el-empty>
            </template>
            <template v-else>
              <v-chart :option="stageDistOption" class="stage-chart" autoresize />
              <div class="stage-list">
                <div v-for="item in stageItems" :key="item.label" class="stage-item">
                  <span>{{ item.label }}</span>
                  <strong>{{ item.value }}</strong>
                </div>
              </div>
            </template>

            <div class="todo-list">
              <button
                v-for="item in actionItems"
                :key="item.title"
                class="todo-item"
                type="button"
                @click="item.path === '/hr/workbench' ? fetchData() : goTo(item.path)"
              >
                <span class="todo-item__icon">
                  <el-icon><component :is="item.icon" /></el-icon>
                </span>
                <span class="todo-item__content">
                  <span class="todo-item__title">{{ item.title }}</span>
                  <span class="todo-item__desc">{{ item.desc }}</span>
                </span>
                <span class="todo-item__meta">
                  <strong>{{ item.value }}</strong>
                  <span>{{ item.action }}</span>
                </span>
              </button>
            </div>
          </el-card>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.workbench {
  height: 100%;
  overflow-y: auto;
  padding: 20px;
  color: var(--text-primary);
}
.workbench-error { padding: 40px 0; }

.quick-actions {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 18px;
  padding: 0 2px;
}

.section-kicker {
  color: var(--text-muted);
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
}

.quick-actions__list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.quick-action {
  height: 34px;
  padding: 0 10px;
  border-radius: 8px;
  color: var(--text-secondary);
}

.quick-action:hover {
  background: var(--brand-soft);
  color: var(--brand);
}

.quick-action__arrow {
  margin-left: 4px;
  font-size: 12px;
}

.metrics-section {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 220px;
  gap: 14px;
  margin-bottom: 18px;
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}

.metric-card {
  position: relative;
  min-height: 150px;
  padding: 18px;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
  box-shadow: var(--admin-console-card-shadow);
}

.metric-card::after {
  content: "";
  position: absolute;
  right: -22px;
  top: -28px;
  width: 92px;
  height: 92px;
  border-radius: 999px;
  opacity: 0.13;
  background: currentColor;
}

.metric-card--blue { color: #2563eb; }
.metric-card--green { color: #16a34a; }
.metric-card--amber { color: #d97706; }
.metric-card--red { color: #dc2626; }

.metric-card__icon {
  width: 42px;
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 14px;
  border-radius: 10px;
  background: color-mix(in srgb, currentColor 12%, transparent);
}

.metric-card__label {
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 600;
}

.metric-card__value {
  margin-top: 8px;
  color: var(--text-primary);
  font-size: 34px;
  line-height: 1;
  font-weight: 800;
}

.metric-card__desc {
  margin-top: 10px;
  color: var(--text-muted);
  font-size: 12px;
}

.secondary-metrics {
  display: grid;
  gap: 10px;
}

.secondary-metric {
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 10px;
  min-height: 70px;
  padding: 14px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: color-mix(in srgb, var(--surface) 82%, transparent);
  color: var(--text-muted);
}

.secondary-metric strong {
  color: var(--text-primary);
  font-size: 22px;
}

.main-workspace {
  display: grid;
  grid-template-columns: minmax(0, 3fr) minmax(340px, 2fr);
  gap: 16px;
  align-items: stretch;
}

.workspace-card {
  height: 100%;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
}

.workspace-card :deep(.el-card__body) {
  height: 100%;
  padding: 20px;
}

.panel-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.panel-header h2 {
  margin: 0;
  color: var(--text-primary);
  font-size: 18px;
  line-height: 1.3;
  font-weight: 750;
}

.panel-header p {
  margin: 6px 0 0;
  color: var(--text-muted);
  font-size: 13px;
}

.panel-header--compact {
  margin-bottom: 8px;
}

.overview-chart {
  height: 360px;
}

.stage-chart {
  height: 230px;
}

.stage-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-top: 8px;
}

.stage-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 8px;
  background: var(--surface-muted);
  color: var(--text-secondary);
  font-size: 12px;
}

.stage-item strong {
  color: var(--text-primary);
  font-size: 15px;
}

.actionable-empty {
  padding: 18px 0 10px;
}

.empty-copy {
  display: grid;
  gap: 4px;
  color: var(--text-muted);
}

.empty-copy strong {
  color: var(--text-primary);
  font-weight: 700;
}

.empty-actions {
  display: flex;
  justify-content: center;
  gap: 8px;
}

.todo-list {
  display: grid;
  gap: 10px;
  margin-top: 16px;
}

.todo-item {
  width: 100%;
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 12px;
  padding: 13px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: transparent;
  color: inherit;
  cursor: pointer;
  text-align: left;
  transition: border-color var(--motion-normal) var(--motion-ease), background-color var(--motion-normal) var(--motion-ease);
}

.todo-item:hover {
  border-color: var(--el-color-primary-light-5);
  background: var(--el-color-primary-light-9);
}

.todo-item__icon {
  width: 34px;
  height: 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 9px;
  background: var(--brand-soft);
  color: var(--brand);
}

.todo-item__content,
.todo-item__meta {
  display: grid;
  gap: 4px;
}

.todo-item__title {
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 700;
}

.todo-item__desc,
.todo-item__meta span {
  color: var(--text-muted);
  font-size: 12px;
}

.todo-item__meta {
  justify-items: end;
  white-space: nowrap;
}

.todo-item__meta strong {
  color: var(--text-primary);
  font-size: 20px;
  line-height: 1;
}

@media (max-width: 768px) {
  .workbench { padding: 14px; }

  .quick-actions {
    align-items: flex-start;
    flex-direction: column;
    gap: 8px;
  }

  .metrics-section,
  .main-workspace {
    grid-template-columns: 1fr;
  }

  .metrics-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .metric-card { min-height: 138px; }
  .metric-card__value { font-size: 28px; }
  .secondary-metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .panel-header { flex-direction: column; }
  .overview-chart { height: 300px; }
}

@media (max-width: 520px) {
  .metrics-grid,
  .secondary-metrics,
  .stage-list {
    grid-template-columns: 1fr;
  }

  .todo-item {
    grid-template-columns: auto 1fr;
  }

  .todo-item__meta {
    grid-column: 2;
    justify-items: start;
  }
}
</style>
