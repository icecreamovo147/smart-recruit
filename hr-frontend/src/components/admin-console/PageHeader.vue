<script setup lang="ts">
defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <header class="admin-page-header">
    <div class="admin-page-header__content">
      <slot name="prefix" />
      <div class="admin-page-header__copy">
        <h1>{{ title }}</h1>
        <p v-if="description">{{ description }}</p>
        <slot />
      </div>
    </div>

    <div v-if="$slots.secondary || $slots.primary || $slots.actions" class="admin-page-header__actions">
      <div v-if="$slots.secondary" class="admin-page-header__secondary">
        <slot name="secondary" />
      </div>
      <slot name="actions" />
      <slot name="primary" />
    </div>
  </header>
</template>

<style scoped>
.admin-page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  padding: 4px 0 18px;
}

.admin-page-header__content {
  min-width: 0;
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.admin-page-header__copy {
  min-width: 0;
}

.admin-page-header h1 {
  margin: 0;
  color: var(--text-primary);
  font-size: 24px;
  font-weight: 700;
  line-height: 1.25;
  letter-spacing: 0;
}

.admin-page-header p {
  max-width: 760px;
  margin: 8px 0 0;
  color: var(--text-muted);
  font-size: 14px;
  line-height: 1.6;
}

.admin-page-header__actions,
.admin-page-header__secondary {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  flex-wrap: wrap;
}

.admin-page-header__actions {
  flex: 0 0 auto;
  padding-top: 2px;
}

@media (max-width: 720px) {
  .admin-page-header {
    display: grid;
  }

  .admin-page-header__actions {
    justify-content: flex-start;
  }
}
</style>
