<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox, ElTag } from 'element-plus'
import { listMyOffers, acceptOffer, rejectOffer } from '@/api/offer'
import type { Offer } from '@/types/domain'

const loading = ref(false)
const offers = ref<Offer[]>([])
const total = ref(0)
const cursor = ref('')
const hasMore = ref(false)
const pageSize = 10
const actionLoading = ref<number | null>(null)

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
    sent: '待处理',
    accepted: '已接受',
    rejected: '已拒绝',
    withdrawn: '已撤回',
  }
  return map[status] || status
}

const offerStatusType = (status: string): string => {
  const map: Record<string, string> = {
    draft: 'info',
    sent: 'warning',
    accepted: 'success',
    rejected: 'danger',
    withdrawn: 'info',
  }
  return map[status] || 'info'
}

const loadOffers = async (reset = false) => {
  loading.value = true
  try {
    if (reset) {
      cursor.value = ''
      offers.value = []
    }
    const data = await listMyOffers({ page_size: pageSize, cursor: cursor.value || undefined })
    if (reset) {
      offers.value = data.list || []
    } else {
      offers.value.push(...(data.list || []))
    }
    total.value = data.total || 0
    cursor.value = data.next_cursor || ''
    hasMore.value = data.has_more || false
  } catch (error: unknown) {
    ElMessage.error(error instanceof Error ? error.message : '加载Offer列表失败')
  } finally {
    loading.value = false
  }
}

const handleAccept = async (offer: Offer) => {
  try {
    await ElMessageBox.confirm(
      `确认接受「${offer.job_title || offer.title}」的Offer？`,
      '确认接受Offer',
      { confirmButtonText: '确认接受', cancelButtonText: '取消', type: 'success' }
    )
    actionLoading.value = offer.id
    await acceptOffer(offer.id)
    ElMessage.success('Offer已接受，等待后续入职流程')
    await loadOffers(true)
  } catch (error: unknown) {
    if (error !== 'cancel') {
      const msg = error instanceof Error ? error.message : '接受Offer失败'
      ElMessage.error(msg)
    }
  } finally {
    actionLoading.value = null
  }
}

const handleReject = async (offer: Offer) => {
  try {
    const { value: reason } = await ElMessageBox.prompt(
      `确认拒绝「${offer.job_title || offer.title}」的Offer？`,
      '拒绝Offer',
      {
        confirmButtonText: '确认拒绝',
        cancelButtonText: '取消',
        inputPlaceholder: '请填写拒绝原因',
        inputValidator: (val: string) => !!val.trim() || '请输入拒绝原因',
      }
    )
    actionLoading.value = offer.id
    await rejectOffer(offer.id, reason)
    ElMessage.success('已拒绝Offer')
    await loadOffers(true)
  } catch (error: unknown) {
    if (error !== 'cancel') {
      const msg = error instanceof Error ? error.message : '拒绝Offer失败'
      ElMessage.error(msg)
    }
  } finally {
    actionLoading.value = null
  }
}

onMounted(() => {
  loadOffers(true)
})
</script>

<template>
  <div class="my-offers-view">
    <h2>我的Offer</h2>

    <div v-loading="loading" class="offer-list">
      <el-empty v-if="!loading && offers.length === 0" description="暂无Offer记录" />

      <el-card v-for="offer in offers" :key="offer.id" class="offer-card" shadow="hover">
        <div class="offer-card-header">
          <div class="offer-card-title">
            <h3>{{ offer.job_title || offer.title }}</h3>
            <el-tag :type="offerStatusType(offer.status) as any" size="small">
              {{ offerStatusLabel(offer.status) }}
            </el-tag>
          </div>
        </div>

        <el-descriptions :column="3" border size="small">
          <el-descriptions-item label="薪资">{{ offer.salary_range || '-' }}</el-descriptions-item>
          <el-descriptions-item label="职级">{{ offer.level || '-' }}</el-descriptions-item>
          <el-descriptions-item label="工作地点">{{ offer.work_location || '-' }}</el-descriptions-item>
          <el-descriptions-item label="预计入职日期">{{ offer.start_date || '-' }}</el-descriptions-item>
          <el-descriptions-item label="过期时间">{{ formatDateTime(offer.expires_at) }}</el-descriptions-item>
          <el-descriptions-item label="发送时间">{{ formatDateTime(offer.created_at) }}</el-descriptions-item>
        </el-descriptions>

        <div v-if="offer.sent_snapshot_json" class="offer-terms">
          <h4>Offer条款</h4>
          <pre>{{ offer.sent_snapshot_json }}</pre>
        </div>
        <div v-else-if="offer.terms_json" class="offer-terms">
          <h4>Offer条款</h4>
          <pre>{{ offer.terms_json }}</pre>
        </div>

        <!-- Actions for sent offers -->
        <div v-if="offer.status === 'sent'" class="offer-actions">
          <el-button
            type="success"
            :loading="actionLoading === offer.id"
            @click="handleAccept(offer)"
          >
            接受Offer
          </el-button>
          <el-button
            type="danger"
            plain
            :loading="actionLoading === offer.id"
            @click="handleReject(offer)"
          >
            拒绝Offer
          </el-button>
        </div>
      </el-card>

      <!-- Load More (cursor-based pagination) -->
      <div v-if="hasMore" class="pagination-wrapper">
        <el-button :loading="loading" @click="loadOffers(false)">
          加载更多 ({{ offers.length }} / {{ total }})
        </el-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.my-offers-view {
  padding: 20px;
  max-width: 900px;
  margin: 0 auto;
}
.my-offers-view h2 {
  margin-bottom: 20px;
  font-size: 22px;
  color: #333;
}
.offer-card {
  margin-bottom: 16px;
}
.offer-card-header {
  margin-bottom: 16px;
}
.offer-card-title {
  display: flex;
  align-items: center;
  gap: 12px;
}
.offer-card-title h3 {
  margin: 0;
  font-size: 17px;
}
.offer-terms {
  margin-top: 16px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 6px;
}
.offer-terms h4 {
  margin: 0 0 8px;
  font-size: 14px;
  color: #666;
}
.offer-terms pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-size: 13px;
  color: #333;
}
.offer-actions {
  margin-top: 16px;
  display: flex;
  gap: 12px;
}
.pagination-wrapper {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}
</style>
