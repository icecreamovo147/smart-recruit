<script setup lang="ts">
import type { TraceOverviewVM } from './agentTraceViewModel'

defineProps<{
  overview: TraceOverviewVM
}>()

const formatDuration = (ms: number | null): string => {
  if (ms == null) return '未记录'
  if (ms < 1000) return `${ms} ms`
  return `${(ms / 1000).toFixed(1)} s`
}
</script>

<template>
  <section class="trace-overview" data-testid="trace-overview">
    <div class="trace-overview__title">执行概览</div>
    <div class="trace-overview__grid">
      <div class="trace-overview__metric">
        <span class="trace-overview__label">最新状态</span>
        <strong>{{ overview.latestRunStatusLabel || '未记录' }}</strong>
      </div>
      <div class="trace-overview__metric">
        <span class="trace-overview__label">模型</span>
        <strong>{{ overview.modelName || '未记录' }}</strong>
      </div>
      <div class="trace-overview__metric">
        <span class="trace-overview__label">意图</span>
        <strong>{{ overview.intentLabel || '未记录' }}</strong>
      </div>
      <div class="trace-overview__metric">
        <span class="trace-overview__label">运行时</span>
        <strong>{{ overview.runtimeLabel || '未记录' }}</strong>
      </div>
      <div class="trace-overview__metric">
        <span class="trace-overview__label">运行数</span>
        <strong>{{ overview.runCount }}</strong>
      </div>
      <div class="trace-overview__metric">
        <span class="trace-overview__label">步骤 / 工具</span>
        <strong>{{ overview.stepCount }} / {{ overview.toolStepCount }}</strong>
      </div>
      <div class="trace-overview__metric">
        <span class="trace-overview__label">失败 / 告警</span>
        <strong>{{ overview.failureCount }} / {{ overview.warningCount }}</strong>
      </div>
      <div class="trace-overview__metric">
        <span class="trace-overview__label">风险 / 策略</span>
        <strong>{{ overview.riskCount }} / {{ overview.policyIssueCount }}</strong>
      </div>
      <div class="trace-overview__metric">
        <span class="trace-overview__label">耗时</span>
        <strong>{{ formatDuration(overview.durationMs) }}</strong>
      </div>
      <div class="trace-overview__metric">
        <span class="trace-overview__label">人工确认</span>
        <strong>{{ overview.confirmationRequired ? '需要' : '否' }}</strong>
      </div>
      <div v-if="overview.legacyTraceCount > 0" class="trace-overview__metric">
        <span class="trace-overview__label">兼容轨迹</span>
        <strong>{{ overview.legacyTraceCount }}</strong>
      </div>
      <div v-if="overview.activeLive" class="trace-overview__metric trace-overview__metric--live">
        <span class="trace-overview__label">实时</span>
        <strong>执行中</strong>
      </div>
    </div>
  </section>
</template>

<style scoped>
.trace-overview {
  margin-bottom: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 12px;
  background: var(--el-fill-color-extra-light);
}

.trace-overview__title {
  font-size: 14px;
  font-weight: 650;
  margin-bottom: 10px;
  color: var(--el-text-color-primary);
}

.trace-overview__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 8px;
}

.trace-overview__metric {
  min-width: 0;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 8px;
  background: var(--el-bg-color);
}

.trace-overview__metric--live {
  border-color: var(--el-color-primary-light-5);
}

.trace-overview__label {
  display: block;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
}

.trace-overview__metric strong {
  display: block;
  font-size: 13px;
  color: var(--el-text-color-primary);
  overflow-wrap: anywhere;
}

@media (max-width: 640px) {
  .trace-overview__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
