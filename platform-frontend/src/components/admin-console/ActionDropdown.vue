<script setup lang="ts">
import { MoreFilled } from '@element-plus/icons-vue'
import type { ConsoleAction } from '@shared/types/ui'

withDefaults(defineProps<{
  actions: ConsoleAction[]
  triggerLabel?: string
  disabled?: boolean
  placement?: 'top' | 'top-start' | 'top-end' | 'bottom' | 'bottom-start' | 'bottom-end'
}>(), {
  triggerLabel: '更多操作',
  placement: 'bottom-end',
})

const emit = defineEmits<{
  select: [action: ConsoleAction]
}>()

function handleCommand(action: ConsoleAction) {
  if (!action.disabled) {
    emit('select', action)
  }
}
</script>

<template>
  <el-dropdown
    class="admin-action-dropdown"
    :disabled="disabled || actions.length === 0"
    :placement="placement"
    trigger="click"
    @command="handleCommand"
  >
    <slot name="trigger">
      <el-button :disabled="disabled || actions.length === 0" :aria-label="triggerLabel" circle>
        <el-icon><MoreFilled /></el-icon>
      </el-button>
    </slot>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item
          v-for="action in actions"
          :key="action.key"
          :command="action"
          :disabled="action.disabled"
          :divided="action.divided"
          :class="{ 'admin-action-dropdown__item--danger': action.danger }"
        >
          <el-icon v-if="action.icon"><component :is="action.icon" /></el-icon>
          <span>{{ action.label }}</span>
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<style scoped>
.admin-action-dropdown :deep(.el-button.is-circle) {
  color: var(--text-secondary);
}

.admin-action-dropdown__item--danger {
  color: var(--el-color-danger);
}

.admin-action-dropdown__item--danger.is-disabled {
  color: var(--el-text-color-disabled);
}
</style>
