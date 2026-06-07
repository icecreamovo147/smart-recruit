<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft } from '@element-plus/icons-vue'
import { listOffersByApplication } from '@/api/offer'
import { getJobDetail } from '@/api/job'
import type { Offer } from '@/types/domain'
import OfferCreateDialog from '@/components/business/OfferCreateDialog.vue'
import OfferDetailDialog from '@/components/business/OfferDetailDialog.vue'

const route = useRoute()
const router = useRouter()

const applicationId = Number(route.params.applicationId)
const loading = ref(false)
const offers = ref<Offer[]>([])
const createDialogVisible = ref(false)
const detailDialogVisible = ref(false)
const selectedOfferId = ref<number | null>(null)
const jobTitle = ref('')
const candidateName = ref('')
const jobSalaryRange = ref('')
const jobWorkLocation = ref('')

const formatDateTime = (value: string): string => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (num: number): string => String(num).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

const offerStatusLabel = (status: string): string => {
  const map: Record<string, string> = {
    draft: '草稿',
    sent: '已发送',
    accepted: '已接受',
    rejected: '已拒绝',
    withdrawn: '已撤回',
  }
  return map[status] || status
}

const offerStatusType = (status: string): string => {
  const map: Record<string, string> = {
    draft: 'info',
    sent: 'primary',
    accepted: 'success',
    rejected: 'danger',
    withdrawn: 'warning',
  }
  return map[status] || 'info'
}

const loadOffers = async () => {
  if (!applicationId) return
  loading.value = true
  try {
    const data = await listOffersByApplication(applicationId)
    offers.value = data.list || []
    if (offers.value.length > 0) {
      jobTitle.value = offers.value[0].job_title || ''
      candidateName.value = offers.value[0].candidate_name || ''
    }
  } catch (error: unknown) {
    ElMessage.error(error instanceof Error ? error.message : '加载Offer列表失败')
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
    // Silently fail — job detail is best-effort
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

onMounted(() => {
  loadOffers()
  const jobId = Number(route.query.job_id)
  if (jobId) {
    loadJobDetail(jobId)
  }
})
</script>

<template>
  <div class="offer-manage-view">
    <div class="page-header">
      <el-button text @click="goBack">
        <el-icon><ArrowLeft /></el-icon>
        返回
      </el-button>
      <h2>Offer管理</h2>
      <el-button type="primary" @click="showCreateDialog">
        创建Offer
      </el-button>
    </div>

    <div v-loading="loading" class="offer-list">
      <el-empty v-if="!loading && offers.length === 0" description="暂未创建Offer">
        <el-button type="primary" @click="showCreateDialog">创建Offer</el-button>
      </el-empty>

      <div
        v-for="offer in offers"
        :key="offer.id"
        class="offer-card"
        @click="showDetail(offer.id)"
      >
        <div class="offer-card__top">
          <div class="offer-card__title-row">
            <span class="offer-card__title">{{ offer.title }}</span>
            <el-tag :type="offerStatusType(offer.status) as any" size="small" effect="plain">
              {{ offerStatusLabel(offer.status) }}
            </el-tag>
          </div>
          <div class="offer-card__meta">
            <div class="offer-card__meta-item">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
              <span>{{ formatDateTime(offer.created_at) }}</span>
            </div>
            <div class="offer-card__meta-item">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
              <span>{{ offer.candidate_name || `#${offer.candidate_user_id}` }}</span>
            </div>
          </div>
        </div>
        <div class="offer-card__divider" />
        <div class="offer-card__grid">
          <div class="offer-card__field">
            <span class="offer-card__label">薪资</span>
            <strong class="offer-card__value">{{ offer.salary_range || '-' }}</strong>
          </div>
          <div class="offer-card__field">
            <span class="offer-card__label">职级</span>
            <strong class="offer-card__value">{{ offer.level || '-' }}</strong>
          </div>
          <div class="offer-card__field">
            <span class="offer-card__label">地点</span>
            <strong class="offer-card__value">{{ offer.work_location || '-' }}</strong>
          </div>
        </div>
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
  </div>
</template>

<style scoped>
.offer-manage-view {
  padding: 20px;
}
.page-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
}
.page-header h2 {
  flex: 1;
  margin: 0;
  font-size: 22px;
  font-weight: 700;
}
.offer-list {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}

@media (max-width: 1100px) {
  .offer-list {
    grid-template-columns: repeat(2, 1fr);
  }
}
.offer-card {
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  background: #fff;
  cursor: pointer;
  transition: box-shadow 0.2s, border-color 0.2s;
  padding: 16px 20px;
}
.offer-card:hover {
  border-color: var(--el-color-primary-light-5);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.06);
}
.offer-card__top {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.offer-card__title-row {
  display: flex;
  align-items: center;
  gap: 10px;
}
.offer-card__title {
  font-size: 16px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.offer-card__meta {
  display: flex;
  gap: 20px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.offer-card__meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.offer-card__meta-item svg {
  flex-shrink: 0;
  opacity: 0.6;
}
.offer-card__divider {
  height: 1px;
  background: var(--el-border-color-lighter);
  margin: 12px 0;
}
.offer-card__grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}
.offer-card__field {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.offer-card__label {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.offer-card__value {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
</style>
