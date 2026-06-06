<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <div class="error-page">
    <div class="error-content">
      <h1 class="error-code">404</h1>
      <p class="error-message">页面不存在</p>
      <p class="error-hint">您访问的页面不存在或已被移除</p>
      <el-button type="primary" @click="router.push('/dashboard')">返回首页</el-button>
    </div>
  </div>
</template>

<style scoped>
.error-page {
  min-height: 60vh;
  display: flex;
  align-items: center;
  justify-content: center;
}

.error-content {
  text-align: center;
}

.error-code {
  font-size: 48px;
  font-weight: 700;
  color: var(--text-disabled);
  line-height: 1;
  margin-bottom: 12px;
}

.error-message {
  font-size: 16px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.error-hint {
  font-size: 14px;
  color: var(--text-muted);
  margin-bottom: 24px;
}
</style>
