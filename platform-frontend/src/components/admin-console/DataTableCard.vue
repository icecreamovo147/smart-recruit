<script setup lang="ts">
const props = defineProps<{
  resultCount?: number
  resultLabel?: string
}>()
</script>

<template>
  <section class="admin-table-card">
    <div class="admin-table-card__body">
      <slot />
    </div>

    <footer v-if="$slots.footer || props.resultCount !== undefined" class="admin-table-card__footer">
      <span v-if="props.resultCount !== undefined" class="admin-table-card__count">
        {{ props.resultLabel || `共 ${props.resultCount} 条` }}
      </span>
      <div v-if="$slots.footer" class="admin-table-card__footer-actions">
        <slot name="footer" />
      </div>
    </footer>
  </section>
</template>

<style scoped>
.admin-table-card {
  overflow: hidden;
}

.admin-table-card__body {
  min-width: 0;
  background: var(--surface-solid-bg);
}

.admin-table-card__body :deep(.el-table) {
  --el-table-header-bg-color: var(--table-header-solid);
  --el-table-bg-color: var(--surface-solid-bg);
  --el-table-tr-bg-color: var(--surface-solid-bg);
  --el-table-row-hover-bg-color: var(--table-hover-bg);
  --el-table-border-color: var(--border);
}

.admin-table-card__body :deep(.el-table__header th.el-table__cell) {
  color: var(--text-secondary);
  font-weight: 700;
}

.admin-table-card__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 12px var(--page-inset);
  border-top: 1px solid var(--admin-console-border);
  background: transparent;
}

.admin-table-card__count {
  color: var(--text-muted);
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.admin-table-card__footer-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  flex: 1 1 auto;
  min-width: 0;
}

@media (max-width: 720px) {
  .admin-table-card__footer {
    flex-wrap: wrap;
    justify-content: flex-start;
  }

  .admin-table-card__footer-actions {
    justify-content: flex-start;
    width: 100%;
  }
}
</style>
