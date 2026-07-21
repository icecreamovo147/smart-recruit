<template>
  <section class="page-panel">
    <slot />
  </section>
</template>

<style scoped>
.page-panel {
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
  border: 1px solid var(--surface-soft-border);
  border-radius: var(--surface-soft-radius, 12px);
  background: var(--surface-soft-bg);
  box-shadow: var(--surface-soft-shadow);
}

/* Nested sections become divider-separated blocks, not nested cards. */
.page-panel :deep(.admin-page-header),
.page-panel :deep(.admin-filter-toolbar),
.page-panel :deep(.admin-table-card),
.page-panel :deep(.workspace-surface),
.page-panel :deep(.catalog-panel),
.page-panel :deep(.dashboard-panel),
.page-panel :deep(.ai-subnav.catalog-subnav) {
  margin: 0;
  border: 0;
  border-radius: 0;
  box-shadow: none;
  background: transparent;
}

.page-panel :deep(.admin-page-header),
.page-panel :deep(.admin-filter-toolbar),
.page-panel :deep(.ai-subnav.catalog-subnav),
.page-panel :deep(.workspace-surface__toolbar),
.page-panel :deep(.metric-grid),
.page-panel :deep(.console-tabs .el-tabs__header) {
  border-bottom: 1px solid var(--surface-soft-border);
}

.page-panel :deep(.console-tabs .el-tabs__header) {
  margin: 0;
  padding: 0 var(--page-inset);
  border-radius: 0;
  box-shadow: none;
  background: transparent;
}

.page-panel :deep(.admin-table-card__body) {
  background: var(--surface-solid-bg);
}

.page-panel :deep(.admin-table-card__footer),
.page-panel :deep(.workspace-surface__pagination) {
  border-top: 1px solid var(--surface-soft-border);
  background: transparent;
}

.page-panel :deep(.workspace-surface__body) {
  padding: 0;
  background: var(--surface-solid-bg);
}

.page-panel :deep(.workspace-surface__divider) {
  display: none;
}

.page-panel :deep(.catalog-panel),
.page-panel :deep(.dashboard-panel) {
  overflow: visible;
}
</style>
