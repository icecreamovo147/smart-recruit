<script setup lang="ts">
import type { Component } from 'vue'
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowDown } from '@element-plus/icons-vue'

export interface SidebarNavItem {
  to: string
  label: string
  visible?: boolean
}

const props = defineProps<{
  icon: Component
  label: string
  open: boolean
  collapsed: boolean
  mobileOpen: boolean
  items: SidebarNavItem[]
}>()

const emit = defineEmits<{
  toggle: []
  closeMobile: []
}>()

const route = useRoute()

const useFlyout = computed(() => props.collapsed && !props.mobileOpen)
const showInlineSubmenu = computed(() => props.open && !useFlyout.value)
const visibleItems = computed(() => props.items.filter((item) => item.visible !== false))

const isItemActive = (to: string) => route.path === to || route.path.startsWith(`${to}/`)
const isGroupActive = computed(() => visibleItems.value.some((item) => isItemActive(item.to)))

const handleToggle = () => {
  if (useFlyout.value) return
  emit('toggle')
}

const handleItemClick = () => {
  emit('closeMobile')
}
</script>

<template>
  <div class="sidebar-nav-group">
    <el-popover
      v-if="useFlyout"
      placement="right-start"
      trigger="hover"
      :show-arrow="false"
      :offset="8"
      :width="196"
      :hide-after="100"
      popper-class="sidebar-flyout-popover"
    >
      <template #reference>
        <button
          class="sidebar-link sidebar-group-toggle"
          :class="{ 'sidebar-group-toggle--active': isGroupActive }"
          type="button"
          :aria-label="label"
          :aria-haspopup="true"
        >
          <el-icon><component :is="icon" /></el-icon>
          <span>{{ label }}</span>
        </button>
      </template>

      <div class="sidebar-flyout">
        <div class="sidebar-flyout__title">{{ label }}</div>
        <RouterLink
          v-for="item in visibleItems"
          :key="item.to"
          class="sidebar-flyout__link"
          :class="{ 'sidebar-flyout__link--active': isItemActive(item.to) }"
          :to="item.to"
          @click="handleItemClick"
        >
          {{ item.label }}
        </RouterLink>
      </div>
    </el-popover>

    <template v-else>
      <button
        class="sidebar-link sidebar-group-toggle"
        :class="{ 'sidebar-group-toggle--active': isGroupActive }"
        type="button"
        :aria-expanded="showInlineSubmenu"
        @click="handleToggle"
      >
        <el-icon><component :is="icon" /></el-icon>
        <span>{{ label }}</span>
        <el-icon class="group-arrow" :class="{ 'group-arrow--open': open }"><ArrowDown /></el-icon>
      </button>
      <div class="sidebar-sub-wrap" :class="{ 'sidebar-sub-wrap--open': showInlineSubmenu }">
        <div class="sidebar-sub-group">
          <RouterLink
            v-for="item in visibleItems"
            :key="item.to"
            class="sidebar-link sidebar-sub-link"
            :to="item.to"
            @click="handleItemClick"
          >
            <span>{{ item.label }}</span>
          </RouterLink>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.sidebar-nav-group :deep(.el-tooltip__trigger) {
  display: block;
  width: 100%;
}
</style>
