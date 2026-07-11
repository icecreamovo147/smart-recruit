<script setup lang="ts">
defineProps<{
  title?: string
  emptyText?: string
  isEmpty?: boolean
  collapsedDefault?: boolean
}>()
</script>

<template>
  <section class="trace-legacy-section" data-testid="trace-legacy-section">
    <div class="trace-legacy-section__title">{{ title || '兼容工具调用明细' }}</div>
    <el-empty v-if="isEmpty" :description="emptyText || '当前筛选下没有兼容轨迹'" />
    <slot v-else />
  </section>
</template>

<style scoped>
.trace-legacy-section__title {
  font-size: 14px;
  font-weight: 650;
  margin: 8px 0 10px;
  color: var(--el-text-color-secondary);
}
</style>
