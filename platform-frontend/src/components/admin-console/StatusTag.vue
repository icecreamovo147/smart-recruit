<script setup lang="ts">
import { computed } from 'vue'
import type { ConsoleStatus } from '@shared/types/ui'

const props = withDefaults(defineProps<{
  status?: ConsoleStatus
  label?: string
  size?: 'large' | 'default' | 'small'
}>(), {
  status: 'default',
  size: 'small',
})

const statusMeta: Record<ConsoleStatus, { label: string; type: '' | 'success' | 'info' | 'warning' | 'danger' }> = {
  enabled: { label: '已启用', type: 'success' },
  disabled: { label: '已停用', type: 'info' },
  default: { label: '默认', type: 'info' },
  bound: { label: '已绑定', type: 'success' },
  unbound: { label: '未绑定', type: 'info' },
  healthy: { label: '健康', type: 'success' },
  error: { label: '异常', type: 'danger' },
  untested: { label: '未测试', type: 'info' },
  current: { label: '当前', type: '' },
  warning: { label: '需关注', type: 'warning' },
}

const meta = computed(() => statusMeta[props.status])
</script>

<template>
  <el-tag
    class="admin-status-tag"
    :class="`admin-status-tag--${status}`"
    :type="meta.type"
    :size="size"
    effect="light"
    round
  >
    <span class="admin-status-tag__dot" />
    <slot>{{ label || meta.label }}</slot>
  </el-tag>
</template>

<style scoped>
.admin-status-tag {
  --admin-status-color: var(--text-muted);
  border-color: color-mix(in srgb, var(--admin-status-color) 24%, transparent);
  background: color-mix(in srgb, var(--admin-status-color) 10%, var(--surface));
  color: var(--admin-status-color);
  font-weight: 650;
}

.admin-status-tag__dot {
  width: 6px;
  height: 6px;
  margin-right: 6px;
  border-radius: 999px;
  background: currentColor;
}

.admin-status-tag--enabled,
.admin-status-tag--bound,
.admin-status-tag--healthy {
  --admin-status-color: #16a34a;
}

.admin-status-tag--error {
  --admin-status-color: #dc2626;
}

.admin-status-tag--warning {
  --admin-status-color: #d97706;
}

.admin-status-tag--current {
  --admin-status-color: var(--brand);
}

:root[data-theme='dark'] .admin-status-tag--enabled,
:root[data-theme='dark'] .admin-status-tag--bound,
:root[data-theme='dark'] .admin-status-tag--healthy {
  --admin-status-color: #4ade80;
}

:root[data-theme='dark'] .admin-status-tag--error {
  --admin-status-color: #f87171;
}

:root[data-theme='dark'] .admin-status-tag--warning {
  --admin-status-color: #fbbf24;
}
</style>
