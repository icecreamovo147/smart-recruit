<script setup lang="ts">
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import type { User } from '@/types/domain'

const auth = useAuthStore()
const user = computed((): Partial<User> => auth.user || {})
</script>

<template>
  <section>
    <div class="page-header">
      <h1 class="page-title">个人信息</h1>
    </div>
    <div class="content-surface profile-panel">
      <div class="profile-avatar">{{ (auth.username || 'H').slice(0, 1).toUpperCase() }}</div>
      <div class="profile-info">
        <h2>{{ auth.username || 'HR 用户' }}</h2>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="账号角色">{{ auth.isRecruitingAdmin ? '招聘管理员' : auth.isSystemAdmin ? '系统管理员' : '招聘专员' }}</el-descriptions-item>
          <el-descriptions-item label="登录状态">已登录</el-descriptions-item>
        </el-descriptions>
      </div>
    </div>
  </section>
</template>
