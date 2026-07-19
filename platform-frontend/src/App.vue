<script setup lang="ts">
import { computed } from 'vue'
import { OfficeBuilding, SwitchButton } from '@element-plus/icons-vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const isLogin = computed(() => route.path === '/login')
const signOut = async () => { await auth.signOut(); await router.push('/login') }
</script>

<template>
  <RouterView v-if="isLogin" />
  <div v-else class="console-shell">
    <aside class="sidebar">
      <div class="brand">Smart Recruit<small>Platform Console</small></div>
      <RouterLink to="/tenants"><el-icon><OfficeBuilding /></el-icon>企业租户</RouterLink>
    </aside>
    <section class="workspace">
      <header><span>平台运营控制台</span><div>{{ auth.username }}<el-button link :icon="SwitchButton" @click="signOut">退出</el-button></div></header>
      <main><RouterView /></main>
    </section>
  </div>
</template>
