<script setup lang="ts">
defineProps<{
  title: string
  value: string | number
  description?: string
}>()
</script>

<template>
  <section class="admin-stat-card">
    <div class="admin-stat-card__top">
      <span class="admin-stat-card__title">{{ title }}</span>
      <span v-if="$slots.icon || $slots.status" class="admin-stat-card__meta">
        <slot name="icon" />
        <slot name="status" />
      </span>
    </div>
    <div class="admin-stat-card__value">{{ value }}</div>
    <p v-if="description" class="admin-stat-card__description">{{ description }}</p>
    <slot />
  </section>
</template>

<style scoped>
.admin-stat-card {
  min-width: 0;
  padding: 18px;
  border: 1px solid var(--admin-console-border);
  border-radius: var(--surface-soft-radius, 12px);
  background: var(--admin-console-surface);
  box-shadow: var(--admin-console-card-shadow);
}

.admin-stat-card__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.admin-stat-card__title {
  min-width: 0;
  color: var(--text-muted);
  font-size: 13px;
  font-weight: 600;
  line-height: 1.4;
}

.admin-stat-card__meta {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--text-muted);
}

.admin-stat-card__value {
  margin-top: 12px;
  color: var(--text-primary);
  font-size: 28px;
  font-weight: 750;
  line-height: 1.15;
}

.admin-stat-card__description {
  margin: 8px 0 0;
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.5;
}
</style>
