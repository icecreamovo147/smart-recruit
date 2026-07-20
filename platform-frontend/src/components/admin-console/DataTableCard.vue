<script setup lang="ts">
defineProps<{
  title?: string
  description?: string
  resultCount?: number
  resultLabel?: string
}>()
</script>

<template>
  <section class="admin-table-card">
    <div v-if="title || description || resultCount !== undefined || $slots.actions" class="admin-table-card__header">
      <div class="admin-table-card__copy">
        <div class="admin-table-card__title-row">
          <h2 v-if="title">{{ title }}</h2>
          <span v-if="resultCount !== undefined" class="admin-table-card__count">
            {{ resultLabel || `共 ${resultCount} 条` }}
          </span>
        </div>
        <p v-if="description">{{ description }}</p>
      </div>
      <div v-if="$slots.actions" class="admin-table-card__actions">
        <slot name="actions" />
      </div>
    </div>

    <div class="admin-table-card__body">
      <slot />
    </div>

    <footer v-if="$slots.footer" class="admin-table-card__footer">
      <slot name="footer" />
    </footer>
  </section>
</template>

<style scoped>
.admin-table-card {
  overflow: hidden;
  border: 1px solid var(--admin-console-border, var(--border));
  border-radius: 8px;
  background: var(--admin-console-surface, var(--surface));
  box-shadow: var(--admin-console-card-shadow, 0 10px 28px rgba(15, 23, 42, 0.06));
}

.admin-table-card__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 18px;
  border-bottom: 1px solid var(--admin-console-border, var(--border));
}

.admin-table-card__copy {
  min-width: 0;
}

.admin-table-card__title-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.admin-table-card h2 {
  margin: 0;
  color: var(--text-primary);
  font-size: 16px;
  font-weight: 700;
  line-height: 1.35;
}

.admin-table-card p {
  margin: 6px 0 0;
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.55;
}

.admin-table-card__count {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 0 8px;
  border-radius: 999px;
  background: var(--surface-muted);
  color: var(--text-muted);
  font-size: 12px;
  font-weight: 600;
}

.admin-table-card__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
  flex: 0 0 auto;
}

.admin-table-card__body {
  min-width: 0;
}

.admin-table-card__body :deep(.el-table) {
  --el-table-header-bg-color: var(--surface-muted);
}

.admin-table-card__body :deep(.el-table__header th.el-table__cell) {
  color: var(--text-secondary);
  font-weight: 700;
}

.admin-table-card__footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  padding: 12px 18px;
  border-top: 1px solid var(--admin-console-border, var(--border));
  background: color-mix(in srgb, var(--surface-muted) 58%, transparent);
}

@media (max-width: 720px) {
  .admin-table-card__header {
    display: grid;
  }

  .admin-table-card__actions,
  .admin-table-card__footer {
    justify-content: flex-start;
  }
}
</style>
