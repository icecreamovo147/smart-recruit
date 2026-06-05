<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getOffer, sendOffer, withdrawOffer, listOfferEvents } from '@/api/offer'
import type { Offer, OfferEvent } from '@/types/domain'

const props = defineProps<{
  visible: boolean
  offerId: number | null
}>()

const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
  (e: 'success'): void
}>()

const loading = ref(false)
const offer = ref<Offer | null>(null)
const events = ref<OfferEvent[]>([])
const actionLoading = ref(false)

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

const eventTypeLabel = (eventType: string): string => {
  const map: Record<string, string> = {
    created: '创建',
    updated: '编辑',
    sent: '发送',
    withdrawn: '撤回',
    accepted: '接受',
    rejected: '拒绝',
    expired: '过期',
  }
  return map[eventType] || eventType
}

const loadOffer = async () => {
  if (!props.offerId) return
  loading.value = true
  try {
    const data = await getOffer(props.offerId)
    offer.value = data.offer
    const eventData = await listOfferEvents(props.offerId)
    events.value = eventData.list || []
  } catch (error: unknown) {
    const msg = error instanceof Error ? error.message : '加载Offer详情失败'
    ElMessage.error(msg)
  } finally {
    loading.value = false
  }
}

const handleSend = async () => {
  if (!props.offerId) return
  try {
    await ElMessageBox.confirm('确认发送该Offer？发送后Offer内容将被快照，修改需撤回后重新发送。', '确认发送', {
      confirmButtonText: '确认发送',
      cancelButtonText: '取消',
      type: 'warning',
    })
    actionLoading.value = true
    await sendOffer(props.offerId)
    ElMessage.success('Offer已发送')
    emit('success')
    await loadOffer()
  } catch (error: unknown) {
    if (error !== 'cancel') {
      const msg = error instanceof Error ? error.message : '发送Offer失败'
      ElMessage.error(msg)
    }
  } finally {
    actionLoading.value = false
  }
}

const handleWithdraw = async () => {
  if (!props.offerId) return
  try {
    const { value: reason } = await ElMessageBox.prompt('请输入撤回原因', '撤回Offer', {
      confirmButtonText: '确认撤回',
      cancelButtonText: '取消',
      inputPlaceholder: '撤回原因（必填）',
      inputValidator: (val: string) => !!val.trim() || '请输入撤回原因',
    })
    actionLoading.value = true
    await withdrawOffer(props.offerId, reason)
    ElMessage.success('Offer已撤回')
    emit('success')
    await loadOffer()
  } catch (error: unknown) {
    if (error !== 'cancel') {
      const msg = error instanceof Error ? error.message : '撤回Offer失败'
      ElMessage.error(msg)
    }
  } finally {
    actionLoading.value = false
  }
}

const parseTerms = (json: string): Record<string, string> | null => {
  if (!json) return null
  try {
    return JSON.parse(json)
  } catch {
    return null
  }
}

onMounted(() => {
  if (props.visible && props.offerId) {
    loadOffer()
  }
})
</script>

<template>
  <el-dialog
    :model-value="props.visible"
    @update:model-value="(val: boolean) => emit('update:visible', val)"
    title="Offer详情"
    width="700px"
    :close-on-click-modal="false"
    @open="loadOffer"
  >
    <div v-loading="loading">
      <template v-if="offer">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="状态" :span="2">
            <el-tag :type="offerStatusType(offer.status) as any">
              {{ offerStatusLabel(offer.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="候选人">{{ offer.candidate_name || `#${offer.candidate_user_id}` }}</el-descriptions-item>
          <el-descriptions-item label="岗位">{{ offer.job_title || offer.title }}</el-descriptions-item>
          <el-descriptions-item label="Offer职位">{{ offer.title }}</el-descriptions-item>
          <el-descriptions-item label="薪资范围">{{ offer.salary_range || '-' }}</el-descriptions-item>
          <el-descriptions-item label="职级">{{ offer.level || '-' }}</el-descriptions-item>
          <el-descriptions-item label="工作地点">{{ offer.work_location || '-' }}</el-descriptions-item>
          <el-descriptions-item label="预计入职日期">{{ offer.start_date || '-' }}</el-descriptions-item>
          <el-descriptions-item label="过期时间">{{ formatDateTime(offer.expires_at) || '-' }}</el-descriptions-item>
          <el-descriptions-item label="创建人">#{{ offer.created_by }}</el-descriptions-item>
          <el-descriptions-item label="发送人">{{ offer.sent_by ? `#${offer.sent_by}` : '-' }}</el-descriptions-item>
          <el-descriptions-item label="决策时间">{{ formatDateTime(offer.decided_at) || '-' }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatDateTime(offer.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="更新时间">{{ formatDateTime(offer.updated_at) }}</el-descriptions-item>
        </el-descriptions>

        <!-- Offer条款 -->
        <template v-if="offer.sent_snapshot_json">
          <h4 class="section-title">已发送快照条款</h4>
          <el-input
            :model-value="offer.sent_snapshot_json"
            type="textarea"
            :rows="3"
            readonly
            style="margin-bottom: 16px"
          />
        </template>
        <template v-else-if="offer.terms_json">
          <h4 class="section-title">条款内容</h4>
          <el-input
            :model-value="offer.terms_json"
            type="textarea"
            :rows="3"
            readonly
            style="margin-bottom: 16px"
          />
        </template>

        <!-- Actions -->
        <div v-if="offer.status === 'draft'" class="action-buttons">
          <el-button type="primary" :loading="actionLoading" @click="handleSend">发送Offer</el-button>
          <el-button type="danger" plain :loading="actionLoading" @click="handleWithdraw">撤回</el-button>
        </div>
        <div v-else-if="offer.status === 'sent'" class="action-buttons">
          <el-tag type="warning">等待候选人决策</el-tag>
          <el-button type="danger" plain :loading="actionLoading" @click="handleWithdraw" style="margin-left: 12px">撤回Offer</el-button>
        </div>

        <!-- Event history -->
        <template v-if="events.length > 0">
          <h4 class="section-title">事件记录</h4>
          <el-timeline>
            <el-timeline-item
              v-for="event in events"
              :key="event.id"
              :timestamp="formatDateTime(event.created_at)"
              placement="top"
            >
              <div>
                <el-tag size="small">{{ eventTypeLabel(event.event_type) }}</el-tag>
                <span style="margin-left: 8px; color: #666;">
                  {{ event.actor_account_type === 'candidate' ? '候选人' : 'HR' }}
                  #{{ event.actor_user_id }}
                </span>
                <p v-if="event.reason" style="margin: 4px 0 0; font-size: 13px; color: #999;">
                  {{ event.reason }}
                </p>
              </div>
            </el-timeline-item>
          </el-timeline>
        </template>
      </template>
      <el-empty v-else-if="!loading" description="暂无Offer数据" />
    </div>
    <template #footer>
      <el-button @click="emit('update:visible', false)">关闭</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.section-title {
  margin: 20px 0 12px;
  font-size: 15px;
  font-weight: 600;
  color: #333;
}
.action-buttons {
  margin: 16px 0;
}
</style>
