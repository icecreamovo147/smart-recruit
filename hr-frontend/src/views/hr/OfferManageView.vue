<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft } from '@element-plus/icons-vue'
import { listOffersByApplication } from '@/api/offer'
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
      <el-button type="primary" @click="showCreateDialog" v-if="offers.length === 0 || offers.every(o => o.status === 'withdrawn')">
        创建Offer
      </el-button>
    </div>

    <div v-loading="loading" class="offer-list">
      <el-empty v-if="!loading && offers.length === 0" description="暂未创建Offer">
        <el-button type="primary" @click="showCreateDialog">创建Offer</el-button>
      </el-empty>

      <el-card
        v-for="offer in offers"
        :key="offer.id"
        class="offer-card"
        shadow="hover"
        @click="showDetail(offer.id)"
      >
        <div class="offer-card-header">
          <span class="offer-title">{{ offer.title }}</span>
          <el-tag :type="offerStatusType(offer.status) as any" size="small">
            {{ offerStatusLabel(offer.status) }}
          </el-tag>
        </div>
        <div class="offer-card-body">
          <el-descriptions :column="3" :size="'small'">
            <el-descriptions-item label="薪资">{{ offer.salary_range || '-' }}</el-descriptions-item>
            <el-descriptions-item label="职级">{{ offer.level || '-' }}</el-descriptions-item>
            <el-descriptions-item label="地点">{{ offer.work_location || '-' }}</el-descriptions-item>
            <el-descriptions-item label="创建时间">{{ formatDateTime(offer.created_at) }}</el-descriptions-item>
            <el-descriptions-item label="候选人">{{ offer.candidate_name || `#${offer.candidate_user_id}` }}</el-descriptions-item>
          </el-descriptions>
        </div>
      </el-card>
    </div>

    <OfferCreateDialog
      v-model:visible="createDialogVisible"
      :application-id="applicationId"
      :job-title="jobTitle"
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
  margin-bottom: 20px;
}
.page-header h2 {
  flex: 1;
  margin: 0;
  font-size: 20px;
}
.offer-card {
  margin-bottom: 12px;
  cursor: pointer;
}
.offer-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
.offer-title {
  font-size: 16px;
  font-weight: 600;
  color: #333;
}
.offer-card-body {
  padding: 0;
}
</style>
