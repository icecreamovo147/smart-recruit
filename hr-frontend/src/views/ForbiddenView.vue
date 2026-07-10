<template>
  <div class="forbidden-page">
    <el-result icon="warning" title="403" sub-title="当前账号无权访问该页面">
      <template #extra>
        <el-button type="primary" @click="$router.push('/login')">重新登录</el-button>
      </template>
    </el-result>
  </div>
</template>

<style scoped>
.forbidden-page {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  background: var(--bg);
}

.forbidden-page :deep(.el-result) {
  padding: 0;
}
</style>
