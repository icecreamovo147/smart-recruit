<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Back, Document, Plus, Refresh, Search, View } from '@element-plus/icons-vue'
import { listOffersByApplication } from '@/api/offer'
import { getJobDetail } from '@/api/job'
import type { Offer } from '@/types/domain'
import OfferCreateDialog from '@/components/business/OfferCreateDialog.vue'
import OfferDetailDialog from '@/components/business/OfferDetailDialog.vue'
import { formatShanghaiDateTime } from '@shared/utils/format'

type OfferTagType = 'primary' | 'success' | 'warning' | 'info' | 'danger'

const route = useRoute()
const router = useRouter()

const applicationId = Number(route.params.applicationId)
const loading = ref(false)
const errorMessage = ref('')
const offers = ref<Offer[]>([])
const keyword = ref('')
const statusFilter = ref('')
const createDialogVisible = ref(false)
const detailDialogVisible = ref(false)
const selectedOfferId = ref<number | null>(null)
const jobTitle = ref(String(route.query.job_title || ''))
const candidateName = ref(String(route.query.candidate_name || ''))
const jobSalaryRange = ref('')
const jobWorkLocation = ref('')

const statusOptions = [
  { value: 'draft', label: '草稿' },
  { value: 'sent', label: '已发送' },
  { value: 'accepted', label: '已接受' },
  { value: 'rejected', label: '已拒绝' },
  { value: 'withdrawn', label: '已撤回' },
]

const formatDateTime = (value: string): string => formatShanghaiDateTime(value, '-', false)

const formatDate = (value: string): string => formatShanghaiDateTime(value, '-', false).slice(0, 10)

const offerStatusLabel = (status: string): string => {
  return statusOptions.find((item) => item.value === status)?.label || status || '未知状态'
}

const offerStatusType = (status: string): OfferTagType => {
  const map: Record<string, OfferTagType> = {
    draft: 'info',
    sent: 'primary',
    accepted: 'success',
    rejected: 'danger',
    withdrawn: 'warning',
  }
  return map[status] || 'info'
}

const filteredOffers = computed(() => {
  const normalizedKeyword = keyword.value.trim().toLowerCase()
  return offers.value.filter((offer) => {
    const matchesKeyword = !normalizedKeyword || [
      offer.title,
      offer.candidate_name,
      offer.job_title,
      offer.salary_range,
      offer.level,
      offer.work_location,
    ].some((value) => String(value || '').toLowerCase().includes(normalizedKeyword))
    const matchesStatus = !statusFilter.value || offer.status === statusFilter.value
    return matchesKeyword && matchesStatus
  })
})

const statusSummary = computed(() => statusOptions.map((item) => ({
  ...item,
  count: offers.value.filter((offer) => offer.status === item.value).length,
})))

const pageDescription = computed(() => {
  if (candidateName.value && jobTitle.value) {
    return `管理 ${candidateName.value} 应聘「${jobTitle.value}」的 Offer 创建、发送与决策记录。`
  }
  if (jobTitle.value) {
    return `管理「${jobTitle.value}」对应投递的 Offer 创建、发送与决策记录。`
  }
  return '管理当前投递的 Offer 创建、发送、候选人决策与历史记录。'
})

const loadOffers = async () => {
  if (!applicationId) {
    errorMessage.value = '投递记录不存在或链接不完整'
    return
  }
  loading.value = true
  errorMessage.value = ''
  try {
    const data = await listOffersByApplication(applicationId)
    offers.value = data.list || []
    if (offers.value.length > 0) {
      jobTitle.value = offers.value[0].job_title || jobTitle.value
      candidateName.value = offers.value[0].candidate_name || candidateName.value
    }
  } catch (error: unknown) {
    errorMessage.value = error instanceof Error ? error.message : '加载 Offer 列表失败'
  } finally {
    loading.value = false
  }
}

const loadJobDetail = async (jobId: number) => {
  try {
    const job = await getJobDetail(jobId)
    jobTitle.value = job.title || ''
    jobSalaryRange.value = job.salary_range || ''
    jobWorkLocation.value = job.location || ''
  } catch {
    // Job context is supplementary; the Offer list remains usable without it.
  }
}

const showCreateDialog = () => {
  createDialogVisible.value = true
}

const showDetail = (offerId: number) => {
  selectedOfferId.value = offerId
  detailDialogVisible.value = true
}

const goBack = () => {
  router.back()
}

onMounted(async () => {
  const jobId = Number(route.query.job_id)
  await Promise.all([
    loadOffers(),
    jobId ? loadJobDetail(jobId) : Promise.resolve(),
  ])
  if (route.query.action === 'create') {
    showCreateDialog()
    const nextQuery = { ...route.query }
    delete nextQuery.action
    void router.replace({ query: nextQuery })
  }
})
</script>

<template>
  <section class="console-page console-page--fill offer-management-page">
    <div class="workspace-surface">
      <div class="workspace-surface__header">
        <div class="workspace-surface__header-copy">
          <p class="console-eyebrow">OFFER MANAGEMENT</p>
          <h1 class="console-title">Offer 管理</h1>
          <p class="console-description">{{ pageDescription }}</p>
          <div class="offer-context-meta">
            <span>投递编号 #{{ applicationId || '-' }}</span>
            <span v-if="candidateName">候选人：{{ candidateName }}</span>
            <span v-if="jobTitle">岗位：{{ jobTitle }}</span>
          </div>
        </div>
        <div class="workspace-surface__header-actions">
          <el-button :icon="Back" @click="goBack">返回台账</el-button>
          <el-button :icon="Refresh" :loading="loading" @click="loadOffers">刷新</el-button>
          <el-button type="primary" :icon="Plus" @click="showCreateDialog">创建 Offer</el-button>
        </div>
      </div>

      <div class="workspace-surface__divider"></div>

      <div class="offer-status-strip" aria-label="Offer 状态概览">
        <button
          class="offer-status-item"
          :class="{ 'is-active': statusFilter === '' }"
          type="button"
          @click="statusFilter = ''"
        >
          <span>全部</span>
          <strong>{{ offers.length }}</strong>
        </button>
        <button
          v-for="item in statusSummary"
          :key="item.value"
          class="offer-status-item"
          :class="{ 'is-active': statusFilter === item.value }"
          type="button"
          @click="statusFilter = statusFilter === item.value ? '' : item.value"
        >
          <span>{{ item.label }}</span>
          <strong>{{ item.count }}</strong>
        </button>
      </div>

      <div class="workspace-surface__toolbar">
        <div class="workspace-surface__filters">
          <el-input
            v-model="keyword"
            :prefix-icon="Search"
            clearable
            placeholder="搜索职位、薪资、职级或地点"
            class="offer-search"
          />
          <el-select v-model="statusFilter" clearable placeholder="全部状态" class="offer-status-select">
            <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </div>
        <div class="offer-result-count">
          当前显示 <strong>{{ filteredOffers.length }}</strong> 条记录
        </div>
      </div>

      <el-alert
        v-if="errorMessage"
        class="workspace-surface__error"
        type="error"
        :title="errorMessage"
        show-icon
        :closable="false"
      >
        <template #default>
          <el-button size="small" type="danger" plain @click="loadOffers">重新加载</el-button>
        </template>
      </el-alert>

      <div class="workspace-surface__body desktop-only">
        <el-table
          v-loading="loading"
          class="console-table offer-table"
          height="100%"
          :data="filteredOffers"
          row-key="id"
          @row-click="(row: Offer) => showDetail(row.id)"
        >
          <template #empty>
            <div class="offer-empty-state">
              <div class="offer-empty-state__icon"><el-icon><Document /></el-icon></div>
              <h3>{{ offers.length === 0 ? '尚未创建 Offer' : '没有符合条件的 Offer' }}</h3>
              <p>{{ offers.length === 0 ? '创建 Offer 后，可在这里持续跟进发送、接受、拒绝和撤回状态。' : '请调整搜索内容或状态筛选条件。' }}</p>
              <el-button v-if="offers.length === 0" type="primary" :icon="Plus" @click.stop="showCreateDialog">创建 Offer</el-button>
              <el-button v-else @click.stop="keyword = ''; statusFilter = ''">清除筛选</el-button>
            </div>
          </template>
          <el-table-column label="Offer 信息" min-width="240">
            <template #default="{ row }">
              <div class="console-entity">
                <div class="console-entity__name">{{ row.title || '未命名 Offer' }}</div>
                <div class="console-entity__meta">#{{ row.id }} · 创建于 {{ formatDateTime(row.created_at) }}</div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="薪酬与职级" min-width="180">
            <template #default="{ row }">
              <div class="offer-field-stack">
                <strong>{{ row.salary_range || '薪资待定' }}</strong>
                <span>{{ row.level || '职级待定' }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="地点与入职" min-width="180">
            <template #default="{ row }">
              <div class="offer-field-stack">
                <strong>{{ row.work_location || '地点待定' }}</strong>
                <span>入职日期 {{ formatDate(row.start_date) }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="有效期" width="170">
            <template #default="{ row }">{{ formatDateTime(row.expires_at) }}</template>
          </el-table-column>
          <el-table-column label="状态" width="110" align="center">
            <template #default="{ row }">
              <el-tag :type="offerStatusType(row.status)" effect="light">{{ offerStatusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="110" fixed="right" align="center">
            <template #default="{ row }">
              <el-button type="primary" link :icon="View" @click.stop="showDetail(row.id)">查看详情</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div v-loading="loading" class="mobile-card-list mobile-only offer-mobile-list">
        <div v-if="!loading && filteredOffers.length === 0" class="offer-empty-state">
          <div class="offer-empty-state__icon"><el-icon><Document /></el-icon></div>
          <h3>{{ offers.length === 0 ? '尚未创建 Offer' : '没有符合条件的 Offer' }}</h3>
          <p>{{ offers.length === 0 ? '创建后可在这里跟进完整状态。' : '请调整当前筛选条件。' }}</p>
          <el-button v-if="offers.length === 0" type="primary" :icon="Plus" @click="showCreateDialog">创建 Offer</el-button>
        </div>
        <article v-for="offer in filteredOffers" :key="offer.id" class="mobile-offer-row" @click="showDetail(offer.id)">
          <div class="mobile-card__header">
            <h3 class="mobile-card__title">{{ offer.title || '未命名 Offer' }}</h3>
            <el-tag :type="offerStatusType(offer.status)" size="small">{{ offerStatusLabel(offer.status) }}</el-tag>
          </div>
          <div class="mobile-card__meta">
            <span>{{ offer.salary_range || '薪资待定' }}</span>
            <span>{{ offer.level || '职级待定' }}</span>
            <span>{{ offer.work_location || '地点待定' }}</span>
            <span>入职 {{ formatDate(offer.start_date) }}</span>
            <span>创建于 {{ formatDateTime(offer.created_at) }}</span>
          </div>
          <div class="mobile-card__actions">
            <el-button type="primary" plain size="small" :icon="View" @click.stop="showDetail(offer.id)">查看详情</el-button>
          </div>
        </article>
      </div>
    </div>

    <OfferCreateDialog
      v-model:visible="createDialogVisible"
      :application-id="applicationId"
      :job-title="jobTitle"
      :salary-range="jobSalaryRange"
      :work-location="jobWorkLocation"
      :candidate-name="candidateName"
      @success="loadOffers"
    />

    <OfferDetailDialog
      v-model:visible="detailDialogVisible"
      :offer-id="selectedOfferId"
      @success="loadOffers"
    />
  </section>
</template>

<style scoped>
.offer-context-meta {
  display: flex;
  align-items: center;
  gap: 8px 16px;
  flex-wrap: wrap;
  margin-top: 12px;
  color: var(--text-muted);
  font-size: 12px;
}

.offer-context-meta span {
  position: relative;
}

.offer-context-meta span + span::before {
  position: absolute;
  left: -9px;
  color: var(--border-strong, var(--border));
  content: '·';
}

.offer-status-strip {
  display: flex;
  align-items: stretch;
  padding: 0 24px;
  border-bottom: 1px solid var(--border);
  overflow-x: auto;
  flex-shrink: 0;
}

.offer-status-item {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 48px;
  padding: 0 16px;
  border: 0;
  border-bottom: 2px solid transparent;
  background: transparent;
  color: var(--text-muted);
  font: inherit;
  cursor: pointer;
  white-space: nowrap;
}

.offer-status-item strong {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 22px;
  height: 22px;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--surface-muted);
  color: var(--text-secondary);
  font-size: 12px;
}

.offer-status-item:hover,
.offer-status-item.is-active {
  color: var(--brand);
}

.offer-status-item.is-active {
  border-bottom-color: var(--brand);
  font-weight: 650;
}

.offer-status-item.is-active strong {
  background: var(--brand-soft);
  color: var(--brand);
}

.offer-search {
  width: 300px;
}

.offer-status-select {
  width: 150px;
}

.offer-result-count {
  flex: 0 0 auto;
  color: var(--text-muted);
  font-size: 13px;
}

.offer-result-count strong {
  color: var(--text-primary);
}

.offer-table :deep(.el-table__row) {
  cursor: pointer;
}

.offer-field-stack {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.offer-field-stack strong {
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 650;
}

.offer-field-stack span {
  color: var(--text-muted);
  font-size: 12px;
}

.offer-empty-state {
  display: grid;
  justify-items: center;
  gap: 10px;
  padding: 54px 20px;
  text-align: center;
}

.offer-empty-state__icon {
  display: grid;
  place-items: center;
  width: 50px;
  height: 50px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface-muted);
  color: var(--brand);
  font-size: 20px;
  font-weight: 750;
}

.offer-empty-state h3 {
  margin: 2px 0 0;
  color: var(--text-primary);
  font-size: 16px;
}

.offer-empty-state p {
  max-width: 480px;
  margin: 0 0 4px;
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.6;
}

.offer-mobile-list {
  padding: 14px;
  overflow-y: auto;
}

.mobile-offer-row {
  padding: 16px;
  border-bottom: 1px solid var(--border);
  cursor: pointer;
}

.mobile-offer-row:last-child {
  border-bottom: 0;
}

@media (max-width: 720px) {
  .offer-context-meta span + span::before {
    display: none;
  }

  .offer-status-strip {
    padding: 0 14px;
  }

  .offer-status-item {
    padding: 0 12px;
  }

  .offer-search,
  .offer-status-select {
    width: 100%;
  }

  .offer-result-count {
    width: 100%;
  }
}
</style>
