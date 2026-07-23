<script setup lang="ts">
defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <section class="admin-empty-guide">
    <div v-if="$slots.icon" class="admin-empty-guide__icon">
      <slot name="icon" />
    </div>
    <h3>{{ title }}</h3>
    <p v-if="description">{{ description }}</p>
    <slot />
    <div v-if="$slots.primary || $slots.secondary" class="admin-empty-guide__actions">
      <slot name="primary" />
      <slot name="secondary" />
    </div>
  </section>
</template>

<style scoped>
.admin-empty-guide {
  display: grid;
  justify-items: center;
  gap: 10px;
  padding: 42px 20px;
  color: var(--text-muted);
  text-align: center;
}

.admin-empty-guide__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border: 1px solid var(--admin-console-border);
  border-radius: var(--surface-soft-radius, 12px);
  background: var(--surface-muted);
  color: var(--brand);
  font-size: 22px;
}

.admin-empty-guide h3 {
  margin: 4px 0 0;
  color: var(--text-primary);
  font-size: 16px;
  font-weight: 700;
  line-height: 1.35;
}

.admin-empty-guide p {
  max-width: 520px;
  margin: 0;
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.6;
}

.admin-empty-guide__actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-top: 6px;
}
</style>
