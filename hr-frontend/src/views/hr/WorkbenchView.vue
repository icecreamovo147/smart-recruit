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
import { useTheme } from '@/composables/useTheme'
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
const { isDark } = useTheme()
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
    { label: '已下线岗位', value: animatedOfflineJobs.value, icon: Briefcase, tone: 'slate', desc: '已关闭岗位数' },
    { label: '未读通知', value: animatedUnread.value, icon: Bell, tone: 'slate', desc: '未读系统通知' },
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

const chartPalette = computed(() => {
  if (isDark.value) {
    return {
      text: '#cbd5e1',
      textMuted: '#94a3b8',
      axis: '#334155',
      splitLine: '#1e293b',
      tooltipBg: '#1e293b',
      tooltipBorder: '#334155',
      bar: '#3b82f6',
    }
  }
  return {
    text: '#334155',
    textMuted: '#64748b',
    axis: '#e2e8f0',
    splitLine: '#e5e7eb',
    tooltipBg: '#ffffff',
    tooltipBorder: '#e2e8f0',
    bar: '#2563eb',
  }
})

const jobDistOption = computed(() => {
  const d = data.value?.job_distribution
  const colors = chartPalette.value
  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: colors.tooltipBg,
      borderColor: colors.tooltipBorder,
      textStyle: { color: colors.text },
    },
    grid: { left: '3%', right: '4%', top: '10%', bottom: '3%', containLabel: true },
    xAxis: {
      type: 'category',
      data: d?.labels || [],
      axisLabel: { color: colors.textMuted },
      axisLine: { lineStyle: { color: colors.axis } },
    },
    yAxis: {
      type: 'value',
      minInterval: 1,
      axisLabel: { color: colors.textMuted },
      splitLine: { lineStyle: { color: colors.splitLine } },
    },
    series: [{
      type: 'bar',
      data: d?.values || [],
      barMaxWidth: 52,
      itemStyle: { color: colors.bar, borderRadius: [6, 6, 0, 0] },
    }],
  }
})

const stageDistOption = computed(() => {
  const d = data.value?.stage_distribution
  const values = d?.values || []
  const total = values.reduce((a, b) => a + b, 0)
  const colors = chartPalette.value
  // Filter out zero-value labels.
  const pieData = (d?.labels || [])
    .map((label, i) => ({ name: label, value: values[i] || 0 }))
    .filter((item) => item.value > 0)
  return {
    tooltip: {
      trigger: 'item',
      backgroundColor: colors.tooltipBg,
      borderColor: colors.tooltipBorder,
      textStyle: { color: colors.text },
    },
    legend: {
      bottom: '0%',
      itemWidth: 8,
      itemHeight: 8,
      textStyle: { color: colors.textMuted },
    },
    series: [{
      type: 'pie',
      center: ['50%', '46%'],
      radius: total > 0 ? ['40%', '65%'] : ['40%', '65%'],
      avoidLabelOverlap: false,
      label: {
        show: true,
        formatter: '{b}: {c}',
        color: colors.text,
      },
      data: pieData,
      itemStyle: {
        color: (params: { dataIndex: number }) => ['#409EFF', '#67C23A', '#E6A23C', '#F56C6C'][params.dataIndex] || '#409EFF',
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

    <div v-else class="dashboard-shell">
      <!-- ── Header ── -->
      <div class="dashboard-header">
        <div class="dashboard-header__title">
          <span class="dashboard-header__eyebrow">HR DASHBOARD</span>
          <h1 class="dashboard-header__heading">工作台</h1>
          <p class="dashboard-header__welcome">欢迎回来，{{ username }}。今日招聘运营概览与待处理事项一目了然。</p>
        </div>
      </div>

      <!-- ── Quick Actions ── -->
      <div class="quick-actions">
        <span class="quick-actions__label">快捷操作</span>
        <div class="quick-actions__list">
          <el-button
            v-for="link in quickLinks"
            :key="link.path"
            class="quick-actions__btn"
            text
            :icon="link.icon"
            @click="goTo(link.path)"
          >
            {{ link.label }}
            <el-icon class="quick-action__arrow"><ArrowRight /></el-icon>
          </el-button>
        </div>
      </div>

      <!-- ── KPI Strip ── -->
      <div class="kpi-strip">
        <div
          v-for="card in primaryKpiCards"
          :key="card.label"
          class="kpi-item"
          :class="`kpi-item--${card.tone}`"
        >
          <div class="kpi-item__icon">
            <el-icon :size="18"><component :is="card.icon" /></el-icon>
          </div>
          <div class="kpi-item__body">
            <div class="kpi-item__label">{{ card.label }}</div>
            <div class="kpi-item__value">{{ card.value }}</div>
            <div class="kpi-item__desc">{{ card.desc }}</div>
          </div>
        </div>
        <div class="kpi-strip__group-divider" />
        <div
          v-for="card in secondaryKpiCards"
          :key="card.label"
          class="kpi-item"
          :class="`kpi-item--${card.tone}`"
        >
          <div class="kpi-item__icon">
            <el-icon :size="18"><component :is="card.icon" /></el-icon>
          </div>
          <div class="kpi-item__body">
            <div class="kpi-item__label">{{ card.label }}</div>
            <div class="kpi-item__value">{{ card.value }}</div>
            <div class="kpi-item__desc">{{ card.desc }}</div>
          </div>
        </div>
      </div>

      <!-- ── Dashboard Body ── -->
      <div class="dashboard-body">
        <!-- Overview Panel -->
        <div class="overview-panel">
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
        </div>

        <!-- Activity Panel -->
        <div class="activity-panel">
          <div class="panel-header panel-header--compact">
            <div>
              <h2>候选人动态</h2>
              <p>聚合候选人阶段变化与当前推荐处理动作。</p>
            </div>
          </div>

          <template v-if="emptyStageData">
            <el-empty class="actionable-empty">
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
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ── Page Level ── */
.workbench {
  height: 100%;
  overflow-y: auto;
  padding: 24px;
  background: var(--bg);
  color: var(--text-primary);
}
.workbench-error {
  padding: 40px 0;
}

/* ── Main Container ── */
.dashboard-shell {
  background: var(--surface);
  border-radius: 20px;
  padding: 28px;
  border: 1px solid var(--border);
  box-shadow: var(--admin-console-card-shadow);
}

/* ── Dashboard Header ── */
.dashboard-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 24px;
}
.dashboard-header__title {
  flex: 1;
  min-width: 0;
}
.dashboard-header__eyebrow {
  display: block;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 1.2px;
  color: var(--brand, #2563eb);
  text-transform: uppercase;
  margin-bottom: 4px;
}
.dashboard-header__heading {
  margin: 0;
  font-size: 22px;
  font-weight: 750;
  color: var(--text-primary);
  line-height: 1.3;
}
.dashboard-header__welcome {
  margin: 6px 0 0;
  font-size: 14px;
  color: var(--text-muted);
}
.dashboard-header__actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

/* ── Quick Actions ── */
.quick-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
  padding: 12px 16px;
  background: var(--surface-muted);
  border-radius: 10px;
  border: 1px solid var(--border);
}
.quick-actions__label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
  white-space: nowrap;
  letter-spacing: 0.5px;
}
.quick-actions__list {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.quick-actions__btn {
  height: 32px;
  padding: 0 12px;
  border-radius: 6px;
  font-size: 13px;
  color: var(--text-secondary);
}
.quick-actions__btn:hover {
  background: var(--brand-soft);
  color: var(--brand);
}
.quick-action__arrow {
  margin-left: 4px;
  font-size: 11px;
}

/* ── KPI Strip ── */
.kpi-strip {
  display: flex;
  align-items: stretch;
  margin-bottom: 24px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
  overflow: hidden;
}
.kpi-item {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 18px 20px;
}
.kpi-item + .kpi-item {
  border-left: 1px solid var(--border);
}
.kpi-strip__group-divider {
  width: 2px;
  min-height: 40px;
  align-self: center;
  background: var(--border);
  border-radius: 1px;
  flex-shrink: 0;
}
.kpi-item__icon {
  width: 38px;
  height: 38px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  flex-shrink: 0;
  background: color-mix(in srgb, currentColor 10%, transparent);
}
.kpi-item--blue { color: #2563eb; }
.kpi-item--green { color: #16a34a; }
.kpi-item--amber { color: #d97706; }
.kpi-item--red { color: #dc2626; }
.kpi-item--slate { color: #64748b; }
.kpi-item__body {
  min-width: 0;
}
.kpi-item__label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 2px;
}
.kpi-item__value {
  font-size: 26px;
  font-weight: 800;
  color: var(--text-primary);
  line-height: 1.2;
}
.kpi-item__desc {
  font-size: 11px;
  color: var(--text-muted);
  margin-top: 1px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ── Dashboard Body ── */
.dashboard-body {
  display: grid;
  grid-template-columns: 1.8fr 1fr;
  gap: 24px;
  align-items: stretch;
}

/* ── Panels ── */
.overview-panel,
.activity-panel {
  padding: 20px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface-muted);
}

.panel-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}
.panel-header h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1.3;
}
.panel-header p {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--text-muted);
}
.panel-header--compact {
  margin-bottom: 12px;
}

.overview-chart {
  height: 340px;
}
.stage-chart {
  height: 200px;
}
.stage-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
  margin-top: 12px;
}
.stage-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 6px;
  background: var(--surface);
  border: 1px solid var(--border);
  color: var(--text-secondary);
  font-size: 12px;
}
.stage-item strong {
  color: var(--text-primary);
  font-size: 14px;
}

/* ── Empty States ── */
.actionable-empty {
  padding: 12px 0 8px;
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

/* ── Todo List ── */
.todo-list {
  display: grid;
  gap: 8px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--border);
}
.todo-item {
  width: 100%;
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  color: inherit;
  cursor: pointer;
  text-align: left;
  transition: border-color var(--motion-normal), background-color var(--motion-normal);
}
.todo-item:hover {
  border-color: color-mix(in srgb, var(--brand) 40%, var(--border));
  background: color-mix(in srgb, var(--brand-soft) 70%, var(--surface));
}
.todo-item__icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: var(--brand-soft);
  color: var(--brand);
}
.todo-item__content,
.todo-item__meta {
  display: grid;
  gap: 2px;
}
.todo-item__title {
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 700;
}
.todo-item__desc,
.todo-item__meta span {
  color: var(--text-muted);
  font-size: 11px;
}
.todo-item__meta {
  justify-items: end;
  white-space: nowrap;
}
.todo-item__meta strong {
  color: var(--text-primary);
  font-size: 18px;
  line-height: 1;
}

/* ── Responsive ── */
@media (max-width: 1200px) {
  .kpi-item {
    padding: 14px 16px;
  }
  .kpi-item__value {
    font-size: 22px;
  }
}

@media (max-width: 992px) {
  .kpi-strip {
    flex-wrap: wrap;
  }
  .kpi-item {
    flex: 1 1 calc(33.33% - 1px);
    border-left: none !important;
    border-bottom: 1px solid var(--border);
  }
  .kpi-item:nth-child(3n) {
    border-right: none;
  }
  .kpi-item:nth-last-child(-n+3) {
    border-bottom: none;
  }
  .kpi-strip__group-divider {
    display: none;
  }

  .dashboard-body {
    grid-template-columns: 1fr;
  }
  .overview-chart {
    height: 300px;
  }
}

@media (max-width: 768px) {
  .workbench {
    padding: 16px;
  }
  .dashboard-shell {
    padding: 20px;
  }

  .dashboard-header {
    flex-direction: column;
  }
  .dashboard-header__actions {
    width: 100%;
  }

  .quick-actions {
    flex-direction: column;
    align-items: flex-start;
  }

  .kpi-item {
    flex: 1 1 calc(50% - 1px);
    border-right: 1px solid var(--border);
  }
  .kpi-item:nth-child(2n) {
    border-right: none;
  }
  .kpi-item:nth-last-child(-n+2) {
    border-bottom: none;
  }
  .kpi-item:nth-child(3n) {
    border-right: 1px solid var(--border);
  }

  .overview-chart {
    height: 280px;
  }
}

@media (max-width: 520px) {
  .dashboard-shell {
    padding: 16px;
    border-radius: 16px;
  }

  .kpi-strip {
    flex-direction: column;
  }
  .kpi-item {
    border-right: none !important;
    border-bottom: 1px solid var(--border);
  }
  .kpi-item:last-child {
    border-bottom: none;
  }

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

  .panel-header {
    flex-direction: column;
  }
}
</style>
