<template>
  <div class="content-surface">
    <el-result icon="warning" title="403" sub-title="当前账号无权访问该页面">
      <template #extra>
        <el-button type="primary" @click="$router.push('/login')">重新登录</el-button>
      </template>
    </el-result>
  </div>
</template>
