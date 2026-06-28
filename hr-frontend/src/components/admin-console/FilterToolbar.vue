<template>
  <section class="admin-filter-toolbar">
    <div class="admin-filter-toolbar__filters">
      <slot />
      <slot name="filters" />
    </div>
    <div v-if="$slots.actions" class="admin-filter-toolbar__actions">
      <slot name="actions" />
    </div>
  </section>
</template>

<style scoped>
.admin-filter-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 14px;
  border: 1px solid var(--admin-console-border, var(--border));
  border-radius: 8px;
  background: var(--admin-console-surface, var(--surface));
}

.admin-filter-toolbar__filters,
.admin-filter-toolbar__actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.admin-filter-toolbar__filters {
  min-width: 0;
  flex: 1 1 auto;
}

.admin-filter-toolbar__actions {
  flex: 0 0 auto;
  justify-content: flex-end;
}

@media (max-width: 760px) {
  .admin-filter-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .admin-filter-toolbar__actions {
    justify-content: flex-start;
  }
}
</style>
