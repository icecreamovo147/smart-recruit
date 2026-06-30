<template>
  <div class="not-found-page">
    <el-result icon="warning" title="404" sub-title="页面不存在或已失效">
      <template #extra>
        <el-button type="primary" @click="$router.push('/')">返回首页</el-button>
      </template>
    </el-result>
  </div>
</template>

<style scoped>
.not-found-page {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  background: var(--bg);
}

.not-found-page :deep(.el-result) {
  padding: 0;
}
</style>
