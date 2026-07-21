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
  margin: 0;
  padding: 12px var(--page-inset);
  border: 0;
  border-bottom: 1px solid var(--admin-console-border);
  border-radius: 0;
  background: transparent;
  box-shadow: none;
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
