<script setup lang="ts">
import type { Session } from '@/types/ai'

defineProps<{
  currentSession: Session
  mobileContextTitle: string
  mobileContextSub: string
  /** 桌面端会话列表是否已收起 */
  sidebarCollapsed?: boolean
}>()

const emit = defineEmits<{
  (e: 'toggle-sidebar'): void
  (e: 'show-trace'): void
}>()
</script>

<template>
  <header class="chat-header">
    <div class="chat-header__left">
      <button
        class="chat-header__menu-btn"
        :class="{ 'chat-header__menu-btn--collapsed': sidebarCollapsed }"
        type="button"
        :aria-label="sidebarCollapsed ? '展开会话列表' : '收起会话列表'"
        :title="sidebarCollapsed ? '展开会话列表' : '收起会话列表'"
        @click="emit('toggle-sidebar')"
      >
        <!-- 侧栏面板图标：收起态与展开态镜像，便于识别 -->
        <svg
          width="18"
          height="18"
          viewBox="0 0 18 18"
          fill="none"
          stroke="currentColor"
          stroke-width="1.6"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <rect x="2.5" y="3" width="13" height="12" rx="2" />
          <path d="M7 3v12" />
          <path v-if="sidebarCollapsed" d="M10 9h3.5M12 7l2 2-2 2" />
          <path v-else d="M11.5 9H8M9.5 7L7.5 9l2 2" />
        </svg>
      </button>
      <div class="chat-header__info">
        <h3 class="chat-header__title">{{ currentSession.title }}</h3>
        <p class="chat-header__subtitle">
          <template v-if="currentSession.application_id">
            简历分析 · 候选人分析会话
          </template>
          <template v-else>
            招聘数据问答 · 招聘业务数据库
          </template>
        </p>
      </div>
    </div>
    <div class="chat-header__right">
      <el-tag
        size="small"
        :type="currentSession.application_id ? 'success' : ''"
        effect="plain"
        round
      >
        {{ currentSession.application_id ? '简历分析' : '数据问答' }}
      </el-tag>
      <el-button size="small" text @click="emit('show-trace')">执行轨迹</el-button>
    </div>
    <!-- Mobile expanded info -->
    <div v-if="currentSession.application_id" class="chat-header__mobile mobile-only">
      <div class="chat-header__mobile-name">{{ mobileContextTitle }}</div>
      <div class="chat-header__mobile-pos">{{ mobileContextSub }}</div>
    </div>
  </header>
</template>

<style scoped>
.chat-header__left {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.chat-header__menu-btn {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  color: var(--text-secondary);
  cursor: pointer;
  transition:
    background-color 0.15s ease,
    border-color 0.15s ease,
    color 0.15s ease;
}

.chat-header__menu-btn:hover {
  background: var(--surface-muted);
  color: var(--text-primary);
  border-color: color-mix(in srgb, var(--brand) 28%, var(--border));
}

.chat-header__menu-btn--collapsed {
  color: var(--brand);
  border-color: color-mix(in srgb, var(--brand) 36%, var(--border));
  background: var(--brand-soft);
}

.chat-header__info {
  min-width: 0;
}

.chat-header__title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-header__subtitle {
  margin: 2px 0 0;
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.4;
}

.chat-header__right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  margin-left: auto;
}

.chat-header__mobile {
  display: none;
  flex-direction: column;
  gap: 2px;
  width: 100%;
  padding-top: 4px;
}

.chat-header__mobile-name {
  font-weight: 700;
  font-size: 14px;
  color: var(--text-primary);
  text-align: center;
}

.chat-header__mobile-pos {
  font-size: 12px;
  color: var(--text-muted);
  text-align: center;
}

@media (max-width: 768px) {
  .chat-header__mobile {
    display: flex;
  }
}
</style>
