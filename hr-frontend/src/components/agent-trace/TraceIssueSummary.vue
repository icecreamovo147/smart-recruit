<script setup lang="ts">
import type { TraceIssueItem } from './agentTraceViewModel'

defineProps<{
  issues: TraceIssueItem[]
}>()

const emit = defineEmits<{
  (e: 'select-issue', issue: TraceIssueItem): void
}>()

const alertType = (severity: TraceIssueItem['severity']): 'error' | 'warning' | 'info' => {
  if (severity === 'error') return 'error'
  if (severity === 'warning') return 'warning'
  return 'info'
}
</script>

<template>
  <section
    v-if="issues.length > 0"
    class="trace-issue-summary"
    data-testid="trace-issue-summary"
  >
    <div class="trace-issue-summary__title">
      异常与风险摘要
      <el-tag size="small" type="danger" effect="plain">{{ issues.length }}</el-tag>
    </div>
    <ul class="trace-issue-summary__list">
      <li
        v-for="issue in issues"
        :key="issue.id"
        class="trace-issue-summary__item"
      >
        <el-alert
          :title="issue.label"
          :description="issue.detail"
          :type="alertType(issue.severity)"
          :closable="false"
          show-icon
          @click="emit('select-issue', issue)"
        />
      </li>
    </ul>
  </section>
</template>

<style scoped>
.trace-issue-summary {
  margin-bottom: 14px;
}

.trace-issue-summary__title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 650;
  margin-bottom: 8px;
  color: var(--el-text-color-primary);
}

.trace-issue-summary__list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.trace-issue-summary__item :deep(.el-alert) {
  cursor: default;
}

.trace-issue-summary__item :deep(.el-alert__description) {
  overflow-wrap: anywhere;
}
</style>
