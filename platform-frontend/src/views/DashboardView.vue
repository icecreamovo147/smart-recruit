<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ArrowRight } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'
import { getPlatformDashboard } from '@/api/dashboard'
import type { PlatformDashboard } from '@/types'

const router = useRouter()
const loading = ref(false)
const errorMessage = ref('')
const dashboard = ref<PlatformDashboard | null>(null)

const load = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    dashboard.value = await getPlatformDashboard()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '运营总览加载失败'
  } finally {
    loading.value = false
  }
}

const cards = computed(() => {
  const value = dashboard.value
  return [
    { label: '企业租户', value: value?.total_tenants ?? 0, hint: `近 30 天新增 ${value?.new_tenants_30d ?? 0}`, tone: 'brand' },
    { label: '正常运营', value: value?.active_tenants ?? 0, hint: '当前允许企业成员访问', tone: 'success' },
    { label: '平台成员关系', value: value?.active_memberships ?? 0, hint: `累计 ${value?.total_memberships ?? 0} 条`, tone: 'blue' },
    { label: '待完善租户', value: value?.tenants_without_admin ?? 0, hint: '缺少有效招聘管理员', tone: 'warning' },
  ]
})

const distribution = computed(() => {
  const total = Math.max(dashboard.value?.total_tenants || 0, 1)
  return [
    { key: 'active', label: '正常', value: dashboard.value?.active_tenants || 0, percent: ((dashboard.value?.active_tenants || 0) / total) * 100 },
    { key: 'suspended', label: '已暂停', value: dashboard.value?.suspended_tenants || 0, percent: ((dashboard.value?.suspended_tenants || 0) / total) * 100 },
    { key: 'disabled', label: '已停用', value: dashboard.value?.disabled_tenants || 0, percent: ((dashboard.value?.disabled_tenants || 0) / total) * 100 },
  ]
})

onMounted(load)
</script>

<template>
  <section class="console-page dashboard-page" v-loading="loading">
    <el-alert v-if="errorMessage" type="error" :title="errorMessage" show-icon :closable="false"><template #default><el-button link type="danger" @click="load">重新加载</el-button></template></el-alert>

    <div class="metric-grid">
      <article v-for="card in cards" :key="card.label" class="metric-card" :data-tone="card.tone">
        <span class="metric-card__label">{{ card.label }}</span>
        <strong>{{ card.value.toLocaleString('zh-CN') }}</strong>
        <small>{{ card.hint }}</small>
      </article>
    </div>

    <div class="dashboard-grid">
      <article class="surface-card">
        <div class="surface-card__header"><div><h2>租户状态分布</h2><p>平台租户当前生命周期状态</p></div><el-button link type="primary" :icon="ArrowRight" @click="router.push('/tenants')">查看租户</el-button></div>
        <div class="distribution-list">
          <button v-for="item in distribution" :key="item.key" type="button" @click="router.push({ path: '/tenants', query: { status: item.key } })">
            <span class="distribution-label"><i :data-status="item.key"></i>{{ item.label }}</span><span class="distribution-track"><b :data-status="item.key" :style="{ width: `${Math.max(item.percent, item.value ? 3 : 0)}%` }"></b></span><strong>{{ item.value }}</strong>
          </button>
        </div>
      </article>

      <article class="surface-card attention-card">
        <div class="surface-card__header"><div><h2>运营待办</h2><p>需要平台侧尽快确认的准入问题</p></div></div>
        <div v-if="(dashboard?.tenants_without_admin || 0) > 0" class="attention-item attention-item--warning">
          <span class="attention-index">01</span><div><strong>{{ dashboard?.tenants_without_admin }} 家企业缺少主管理员</strong><p>企业无法完成成员和招聘权限治理，请补充有效招聘管理员。</p></div><el-button type="warning" plain @click="router.push('/tenants')">立即处理</el-button>
        </div>
        <div v-if="(dashboard?.suspended_tenants || 0) > 0" class="attention-item">
          <span class="attention-index">02</span><div><strong>{{ dashboard?.suspended_tenants }} 家企业处于暂停状态</strong><p>检查暂停原因与恢复条件，避免长期遗留异常状态。</p></div><el-button plain @click="router.push({ path: '/tenants', query: { status: 'suspended' } })">查看列表</el-button>
        </div>
        <el-empty v-if="!(dashboard?.tenants_without_admin || dashboard?.suspended_tenants)" :image-size="72" description="当前没有需要处理的租户准入问题" />
      </article>
    </div>
  </section>
</template>
