<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  skills?: string[] | null
  limit?: number
  trigger?: 'hover' | 'click'
}>(), {
  skills: () => [],
  limit: 3,
  trigger: 'hover',
})

const normalizedSkills = computed(() => {
  const seen = new Set<string>()
  return (props.skills || []).reduce<string[]>((result, value) => {
    const skill = String(value || '').trim()
    const normalized = skill.toLocaleLowerCase()
    if (!skill || seen.has(normalized)) return result
    seen.add(normalized)
    result.push(skill)
    return result
  }, [])
})

const visibleLimit = computed(() => Math.max(1, Math.floor(Number(props.limit) || 3)))
const visibleSkills = computed(() => normalizedSkills.value.slice(0, visibleLimit.value))
const remainingCount = computed(() => Math.max(0, normalizedSkills.value.length - visibleSkills.value.length))
</script>

<template>
  <div class="skill-tag-summary" @click.stop>
    <span v-if="normalizedSkills.length === 0" class="skill-tag-summary__empty">-</span>
    <el-tag
      v-for="skill in visibleSkills"
      :key="skill"
      class="skill-tag-summary__tag"
      size="small"
      effect="plain"
      :title="skill"
    >
      {{ skill }}
    </el-tag>

    <el-popover
      v-if="remainingCount > 0"
      placement="bottom-start"
      :trigger="trigger"
      :width="320"
    >
      <template #reference>
        <button
          class="skill-tag-summary__more"
          type="button"
          :aria-label="`查看其余 ${remainingCount} 项技能`"
          @click.stop
        >
          +{{ remainingCount }}
        </button>
      </template>
      <div class="skill-tag-summary__popover" aria-label="完整技能列表">
        <el-tag
          v-for="skill in normalizedSkills"
          :key="skill"
          size="small"
          effect="plain"
        >
          {{ skill }}
        </el-tag>
      </div>
    </el-popover>
  </div>
</template>

<style scoped>
.skill-tag-summary {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.skill-tag-summary__tag {
  min-width: 0;
  max-width: 96px;
  flex: 0 1 auto;
}

.skill-tag-summary__tag :deep(.el-tag__content) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.skill-tag-summary__more {
  height: 24px;
  flex: 0 0 auto;
  padding: 0 8px;
  border: 1px solid var(--el-color-info-light-5);
  border-radius: var(--el-border-radius-base);
  background: var(--el-color-info-light-9);
  color: var(--el-color-info);
  font: inherit;
  font-size: 12px;
  line-height: 22px;
  cursor: pointer;
  transition: border-color 120ms ease, background-color 120ms ease;
}

.skill-tag-summary__more:hover,
.skill-tag-summary__more:focus-visible {
  border-color: var(--el-color-info-light-3);
  background: var(--el-color-info-light-8);
  outline: none;
}

.skill-tag-summary__empty {
  color: var(--text-muted);
}

.skill-tag-summary__popover {
  max-height: 240px;
  display: flex;
  align-items: flex-start;
  align-content: flex-start;
  flex-wrap: wrap;
  gap: 8px;
  overflow-y: auto;
}
</style>
