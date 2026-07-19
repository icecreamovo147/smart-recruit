<template>
  <section class="state-page">
    <el-result icon="warning" title="暂无访问权限" sub-title="当前平台角色未被授予这个功能的访问权限。">
      <template #extra><el-button type="primary" @click="$router.push('/')">返回可访问首页</el-button></template>
    </el-result>
  </section>
</template>
