<script setup lang="ts">
/**
 * Presentational wrapper for durable-run timeline content.
 * Parent still owns detailed run/step rendering via default slot to avoid
 * duplicating large template logic across tasks.
 */
defineProps<{
  title?: string
  emptyText?: string
  isEmpty?: boolean
}>()
</script>

<template>
  <section class="trace-run-section" data-testid="trace-run-section">
    <div v-if="title" class="trace-run-section__title">{{ title }}</div>
    <el-empty v-if="isEmpty" :description="emptyText || '当前筛选下没有运行步骤'" />
    <slot v-else />
  </section>
</template>

<style scoped>
.trace-run-section__title {
  font-size: 14px;
  font-weight: 650;
  margin-bottom: 10px;
}
</style>
