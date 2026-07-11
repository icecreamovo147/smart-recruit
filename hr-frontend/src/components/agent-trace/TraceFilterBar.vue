<script setup lang="ts">
import type { TraceFilterState, TraceStatusCategory, TraceTypeCategory } from './agentTraceViewModel'

const filter = defineModel<TraceFilterState>({ required: true })

const emit = defineEmits<{
  (e: 'reset'): void
}>()

const statusOptions: Array<{ label: string; value: TraceFilterState['statusFilter'] }> = [
  { label: '全部状态', value: 'all' },
  { label: '失败', value: 'failed' },
  { label: '告警', value: 'warning' },
  { label: '成功', value: 'succeeded' },
  { label: '进行中', value: 'active' },
]

const typeOptions: Array<{ label: string; value: TraceFilterState['typeFilter'] }> = [
  { label: '全部类型', value: 'all' },
  { label: '规划', value: 'plan' },
  { label: '工具', value: 'tool' },
  { label: '证据', value: 'evidence' },
  { label: '记忆', value: 'memory' },
  { label: '技能/提示', value: 'prompt' },
  { label: '模型', value: 'model' },
  { label: '降级/恢复', value: 'fallback' },
  { label: '兼容轨迹', value: 'legacy' },
]

const updateKeyword = (value: string) => {
  filter.value = { ...filter.value, keyword: value }
}

const updateStatus = (value: TraceFilterState['statusFilter']) => {
  filter.value = { ...filter.value, statusFilter: value }
}

const updateType = (value: TraceFilterState['typeFilter']) => {
  filter.value = { ...filter.value, typeFilter: value }
}

const updateIssueOnly = (value: boolean) => {
  filter.value = { ...filter.value, issueOnly: value }
}
</script>

<template>
  <section class="trace-filter-bar" data-testid="trace-filter-bar">
    <el-input
      :model-value="filter.keyword"
      clearable
      placeholder="搜索运行、步骤、工具、错误、策略、输入输出..."
      data-testid="trace-filter-keyword"
      @update:model-value="updateKeyword"
    />
    <div class="trace-filter-bar__row">
      <el-select
        :model-value="filter.statusFilter"
        data-testid="trace-filter-status"
        style="width: 140px"
        @update:model-value="updateStatus"
      >
        <el-option
          v-for="opt in statusOptions"
          :key="opt.value"
          :label="opt.label"
          :value="opt.value"
        />
      </el-select>
      <el-select
        :model-value="filter.typeFilter"
        data-testid="trace-filter-type"
        style="width: 150px"
        @update:model-value="updateType"
      >
        <el-option
          v-for="opt in typeOptions"
          :key="opt.value"
          :label="opt.label"
          :value="opt.value"
        />
      </el-select>
      <el-checkbox
        :model-value="filter.issueOnly"
        data-testid="trace-filter-issue-only"
        @update:model-value="updateIssueOnly"
      >
        仅看异常
      </el-checkbox>
      <el-button data-testid="trace-filter-reset" @click="emit('reset')">重置筛选</el-button>
    </div>
  </section>
</template>

<style scoped>
.trace-filter-bar {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}

.trace-filter-bar__row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}
</style>
