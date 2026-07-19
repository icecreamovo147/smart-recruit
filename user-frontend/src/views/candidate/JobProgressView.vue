<script setup lang="ts">
import { computed, ref, reactive, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useOfferBadge } from '@/composables/useOfferBadge'
import { listMyApplications } from '@/api/application'
import { listMyInterviews } from '@/api/interview'
import { listMyOffers as fetchMyOffers, acceptOffer, rejectOffer } from '@/api/offer'
import { getCandidateStatusLabel, getStatusType } from '@/types/domain'
import type { Application, InterviewSchedule, JobQuery, Offer } from '@/types/domain'

const { pendingOfferCount, refreshPendingOfferCount } = useOfferBadge()

const route = useRoute()
const router = useRouter()
const activeTab = ref('applications')
const activeApplicationCount = computed(() => applications.value.filter((item) => item.is_current === 1).length)
const upcomingInterviewCount = computed(() =>
  interviews.value.filter((item) => item.status === 'pending' || item.status === 'scheduled').length,
)

// ---- Initialise tab from query param ----
const TAB_MAP: Record<string, string> = {
  applications: 'applications',
  interviews: 'interviews',
  offers: 'offers',
}
if (route.query.tab && TAB_MAP[route.query.tab as string]) {
  activeTab.value = TAB_MAP[route.query.tab as string]
}

// ============================================================
// Tab 1 — 我的投递
// ============================================================
const appLoading = ref(false)
const appError = ref('')
const applications = ref<any[]>([])
const appTotal = ref(0)
const appQuery = reactive<JobQuery>({ page: 1, page_size: 10 })

const fmtDT = (value: string): string => {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const appStatusLabel = (row: Application): string => {
  if (row.status_key) return getCandidateStatusLabel(row.status_key)
  return (['待查看', '已查看', '通过', '已进入公司公共人才库'])[row.status] || '未知'
}
const appStatusType = (row: Application): string => {
  if (row.status_key) return getStatusType(row.status_key)
  return (['info', 'primary', 'success', 'danger'])[row.status] || 'info'
}

const loadApplications = async () => {
  appLoading.value = true
  appError.value = ''
  try {
    appQuery.page = Number(appQuery.page) || 1
    appQuery.page_size = Number(appQuery.page_size) || 10
    const data = await listMyApplications(appQuery)
    applications.value = (data.list || []).map((item: any) => ({
      ...item,
      applied_time_display: fmtDT(item.applied_at),
      status: Number(item.status ?? 0),
      round_no: item.round_no || 1,
      is_current: Number(item.is_current ?? 0),
      application_id: item.application_id,
      job_id: item.job_id,
      job_title: item.job_title,
      applied_at: item.applied_at,
    }))
    appTotal.value = Number(data.total) || 0
  } catch (error: unknown) {
    appError.value = error instanceof Error ? error.message : '投递记录加载失败'
  } finally {
    appLoading.value = false
  }
}

// Navigate to offers tab with a specific offer highlighted
const goToOffer = (offerId: number) => {
  activeTab.value = 'offers'
  // Push a query param so the offers tab can highlight this offer
  router.replace({ query: { ...route.query, tab: 'offers', offer_id: String(offerId) } })
}

// ============================================================
// Tab 2 — 面试安排
// ============================================================
const intLoading = ref(false)
const intError = ref('')
const interviews = ref<InterviewSchedule[]>([])

const interviewStatusLabel = (status: string): string =>
  ({ pending: '待安排', scheduled: '已安排', completed: '已完成', cancelled: '已取消' })[status] || status
const interviewStatusType = (status: string): string =>
  ({ pending: 'info', scheduled: 'warning', completed: 'success', cancelled: 'danger' })[status] || 'info'
const modeLabel = (mode: string): string =>
  ({ video: '视频面试', phone: '电话面试', onsite: '现场面试' })[mode] || mode || '视频面试'

const loadInterviews = async () => {
  intLoading.value = true
  intError.value = ''
  try {
    const data = await listMyInterviews()
    interviews.value = data.list || []
  } catch (error: unknown) {
    intError.value = error instanceof Error ? error.message : '面试信息加载失败'
  } finally {
    intLoading.value = false
  }
}

// ============================================================
// Tab 3 — 录用通知
// ============================================================
const offerLoading = ref(false)
const offers = ref<Offer[]>([])
const offerTotal = ref(0)
const offerCursor = ref('')
const offerHasMore = ref(false)
const actionLoading = ref<number | null>(null)
const highlightOfferId = ref<number | null>(null)

const offerStatusLabel = (status: string): string =>
  ({ draft: '草稿', sent: '待处理', accepted: '已接受', rejected: '已拒绝', withdrawn: '已撤回' })[status] || status
const offerStatusType = (status: string): string =>
  ({ draft: 'info', sent: 'warning', accepted: 'success', rejected: 'danger', withdrawn: 'info' })[status] || 'info'

const offerTerms = (offer: Offer): Array<{ label: string; value: string }> => {
  const source = offer.sent_snapshot_json || offer.terms_json
  if (!source) return []
  try {
    const parsed = JSON.parse(source) as Record<string, unknown>
    const labels: Record<string, string> = {
      title: '录用职位',
      salary_range: '薪资范围',
      level: '职级',
      work_location: '工作地点',
      start_date: '入职日期',
      terms_json: '补充条款',
    }
    return Object.entries(parsed)
      .filter(([, value]) => value !== '' && value !== null && value !== undefined)
      .map(([key, value]) => ({
        label: labels[key] || key,
        value: typeof value === 'string' ? value : JSON.stringify(value),
      }))
  } catch {
    return [{ label: '补充条款', value: source }]
  }
}

/** Days remaining before an offer expires. */
const offerDaysRemaining = (expiresAt: string): number | null => {
  if (!expiresAt) return null
  const diff = new Date(expiresAt).getTime() - Date.now()
  if (diff <= 0) return 0
  return Math.ceil(diff / 86400000)
}

const loadOffers = async (reset = false) => {
  offerLoading.value = true
  try {
    if (reset) {
      offerCursor.value = ''
      offers.value = []
    }
    const data = await fetchMyOffers({ page_size: 10, cursor: offerCursor.value || undefined })
    if (reset) {
      offers.value = data.list || []
    } else {
      offers.value.push(...(data.list || []))
    }
    offerTotal.value = data.total || 0
    offerCursor.value = data.next_cursor || ''
    offerHasMore.value = data.has_more || false

    // Update badge after load
    await refreshPendingOfferCount()
  } catch (error: unknown) {
    ElMessage.error(error instanceof Error ? error.message : '加载Offer列表失败')
  } finally {
    offerLoading.value = false
  }
}

/** Scroll to and highlight the targeted offer card. */
const scrollToOffer = (offerId: number) => {
  highlightOfferId.value = offerId
  setTimeout(() => {
    const el = document.getElementById(`offer-card-${offerId}`)
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' })
      el.classList.add('offer-card--highlight')
      setTimeout(() => el.classList.remove('offer-card--highlight'), 3000)
    }
  }, 300)
}

// Watch for offer_id in query params when offers tab becomes active
watch(activeTab, (tab) => {
  if (tab === 'offers' && route.query.offer_id) {
    const id = Number(route.query.offer_id)
    if (id) {
      // If offers already loaded, scroll; otherwise flag for after load
      const found = offers.value.find((o) => o.id === id)
      if (found) {
        scrollToOffer(id)
      } else {
        highlightOfferId.value = id
      }
    }
  }
})

// When offers finish loading, check if we need to highlight
watch(offers, (list) => {
  if (highlightOfferId.value) {
    const found = list.find((o) => o.id === highlightOfferId.value)
    if (found) {
      scrollToOffer(highlightOfferId.value)
    }
  }
})

const handleAccept = async (offer: Offer) => {
  try {
    await ElMessageBox.confirm(
      `确认接受「${offer.job_title || offer.title}」的Offer？`,
      '确认接受Offer',
      { confirmButtonText: '确认接受', cancelButtonText: '取消', type: 'success' },
    )
    actionLoading.value = offer.id
    await acceptOffer(offer.id)
    ElMessage.success('Offer已接受，等待后续入职流程')
    await loadOffers(true)
    await refreshPendingOfferCount()
  } catch (error: unknown) {
    if (error !== 'cancel') {
      ElMessage.error(error instanceof Error ? error.message : '接受Offer失败')
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
      },
    )
    actionLoading.value = offer.id
    await rejectOffer(offer.id, reason)
    ElMessage.success('已拒绝Offer')
    await loadOffers(true)
    await refreshPendingOfferCount()
  } catch (error: unknown) {
    if (error !== 'cancel') {
      ElMessage.error(error instanceof Error ? error.message : '拒绝Offer失败')
    }
  } finally {
    actionLoading.value = null
  }
}

// ============================================================
// Lifecycle
// ============================================================
onMounted(async () => {
  loadApplications()
  loadInterviews()
  // Load offers in background for badge + display
  await loadOffers(true)

  // Handle initial deep-link
  if (route.query.offer_id) {
    const id = Number(route.query.offer_id)
    if (id) {
      activeTab.value = 'offers'
      highlightOfferId.value = id
      const found = offers.value.find((o) => o.id === id)
      if (found) scrollToOffer(id)
    }
  }
})
</script>

<template>
  <section class="job-progress">
    <div class="progress-workspace">
      <header class="progress-header">
        <div>
          <span class="progress-eyebrow">CANDIDATE JOURNEY</span>
          <h1 class="page-title">求职进展</h1>
          <p class="page-subtitle">集中查看你的投递状态、面试安排与录用通知。</p>
        </div>
        <el-button type="primary" @click="router.push('/jobs')">发现更多岗位</el-button>
      </header>

      <div class="progress-body">
        <nav class="progress-summary" aria-label="求职进展概览">
        <button
          type="button"
          class="summary-item"
          :class="{ active: activeTab === 'applications' }"
          :aria-pressed="activeTab === 'applications'"
          @click="activeTab = 'applications'"
        >
          <span class="summary-icon" aria-hidden="true">投</span>
          <span class="summary-copy">
            <span class="summary-label">全部投递</span>
            <small>{{ activeApplicationCount }} 个进行中</small>
          </span>
          <strong class="summary-value">{{ appTotal }}</strong>
        </button>
        <button
          type="button"
          class="summary-item"
          :class="{ active: activeTab === 'interviews' }"
          :aria-pressed="activeTab === 'interviews'"
          @click="activeTab = 'interviews'"
        >
          <span class="summary-icon" aria-hidden="true">面</span>
          <span class="summary-copy">
            <span class="summary-label">待参加面试</span>
            <small>共 {{ interviews.length }} 场记录</small>
          </span>
          <strong class="summary-value">{{ upcomingInterviewCount }}</strong>
        </button>
        <button
          type="button"
          class="summary-item summary-item--offer"
          :class="{ active: activeTab === 'offers' }"
          :aria-pressed="activeTab === 'offers'"
          @click="activeTab = 'offers'"
        >
          <span class="summary-icon" aria-hidden="true">录</span>
          <span class="summary-copy">
            <span class="summary-label">待处理录用</span>
            <small>请留意有效期限</small>
          </span>
          <strong class="summary-value">{{ pendingOfferCount }}</strong>
        </button>
        </nav>

        <main class="progress-content">
        <!-- ─── Tab 1: 我的投递 ─── -->
        <section v-if="activeTab === 'applications'" class="progress-section">
          <el-alert v-if="appError" :title="appError" type="error" show-icon :closable="false">
            <template #default>
              <el-button size="small" type="danger" plain @click="loadApplications">重试</el-button>
            </template>
          </el-alert>

          <div v-loading="appLoading" class="tab-content">
            <div class="section-heading">
              <div>
                <h2>我的投递</h2>
                <p>查看岗位当前所处阶段和历史投递记录。</p>
              </div>
              <el-button type="primary" plain @click="router.push('/jobs')">继续发现岗位</el-button>
            </div>
            <el-scrollbar class="tab-scroll">
              <el-table
                v-if="applications.length > 0"
                class="desktop-progress-table"
                :data="applications"
                empty-text="暂无投递记录"
              >
                <el-table-column label="岗位" min-width="180">
                  <template #default="{ row }">
                    <el-link type="primary" @click="router.push(`/jobs/${row.job_id}`)">{{ row.job_title }}</el-link>
                  </template>
                </el-table-column>
                <el-table-column label="状态" width="130">
                  <template #default="{ row }">
                    <el-tag :type="appStatusType(row)">{{ appStatusLabel(row) }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="applied_time_display" label="投递时间" width="160" sortable="custom" />
                <el-table-column label="轮次" width="80">
                  <template #default="{ row }">第 {{ row.round_no }} 轮</template>
                </el-table-column>
                <el-table-column label="流程" width="100">
                  <template #default="{ row }">
                    <el-tag :type="row.is_current === 1 ? 'success' : 'info'" size="small">
                      {{ row.is_current === 1 ? '当前投递' : '历史投递' }}
                    </el-tag>
                  </template>
                </el-table-column>
              </el-table>

              <div v-if="applications.length > 0" class="mobile-application-list">
                <article v-for="row in applications" :key="row.application_id" class="application-card">
                  <div class="application-card__top">
                    <div>
                      <span class="application-card__hint">应聘岗位</span>
                      <h3 @click="router.push(`/jobs/${row.job_id}`)">{{ row.job_title }}</h3>
                    </div>
                    <el-tag :type="appStatusType(row)">{{ appStatusLabel(row) }}</el-tag>
                  </div>
                  <div class="application-card__meta">
                    <span>{{ row.applied_time_display }}</span>
                    <span>第 {{ row.round_no }} 轮</span>
                    <span>{{ row.is_current === 1 ? '当前投递' : '历史投递' }}</span>
                  </div>
                </article>
              </div>

              <el-empty v-if="!appLoading && applications.length === 0" description="暂无投递记录">
                <el-button type="primary" @click="router.push('/jobs')">去看看岗位</el-button>
              </el-empty>
            </el-scrollbar>

            <div v-if="applications.length > 0" class="pagination-wrapper">
              <el-pagination
                v-model:current-page="appQuery.page"
                v-model:page-size="appQuery.page_size"
                layout="total, prev, pager, next"
                :total="appTotal"
                @current-change="loadApplications"
              />
            </div>
          </div>
        </section>

        <!-- ─── Tab 2: 面试安排 ─── -->
        <section v-if="activeTab === 'interviews'" class="progress-section">
          <el-alert v-if="intError" :title="intError" type="error" show-icon closable class="mb-4" />
          <div v-loading="intLoading" class="tab-content">
            <div class="section-heading">
              <div>
                <h2>面试安排</h2>
                <p>按时间查看面试方式、地点与注意事项。</p>
              </div>
            </div>
            <el-scrollbar class="tab-scroll">
              <el-empty v-if="!intLoading && interviews.length === 0" description="暂无面试安排" />

              <div v-else class="interview-list">
                <article v-for="item in interviews" :key="item.interview_id" class="interview-card">
                  <div class="interview-card__date-col">
                    <div class="interview-card__date-badge">
                      <span class="interview-card__month">{{ fmtDT(item.scheduled_at).slice(5, 7) }}月</span>
                      <span class="interview-card__day">{{ fmtDT(item.scheduled_at).slice(8, 10) }}</span>
                    </div>
                    <div class="interview-card__time">
                      {{ fmtDT(item.scheduled_at).slice(11, 16) }}
                    </div>
                  </div>
                  <div class="interview-card__body">
                    <div class="interview-card__top">
                      <div class="interview-card__title-area">
                        <span class="interview-card__meta-tag">第 {{ item.round_no || 1 }} 轮 · {{ modeLabel(item.mode) }}</span>
                        <h3 class="interview-card__title">{{ item.job_title }}</h3>
                        <span v-if="item.title" class="interview-card__subtitle">{{ item.title }}</span>
                      </div>
                      <el-tag :type="interviewStatusType(item.status)" effect="plain" size="small">
                        {{ interviewStatusLabel(item.status) }}
                      </el-tag>
                    </div>
                    <div class="interview-card__info-grid">
                      <div class="interview-card__info-item">
                        <span class="interview-card__info-label">面试官</span>
                        <strong class="interview-card__info-value">{{ item.interviewer_name || '-' }}</strong>
                      </div>
                      <div class="interview-card__info-item">
                        <span class="interview-card__info-label">预计时长</span>
                        <strong class="interview-card__info-value">{{ item.duration_minutes ? item.duration_minutes + ' 分钟' : '-' }}</strong>
                      </div>
                      <div v-if="item.location" class="interview-card__info-item">
                        <span class="interview-card__info-label">面试地点</span>
                        <strong class="interview-card__info-value">{{ item.location }}</strong>
                      </div>
                    </div>
                    <div v-if="item.mode === 'video' || item.meeting_url || item.candidate_note" class="interview-card__footer">
                      <div class="interview-card__footer-left">
                        <span v-if="item.mode === 'video'" class="interview-card__link-label">面试链接：</span>
                        <a
                          v-if="item.meeting_url"
                          :href="item.meeting_url"
                          target="_blank"
                          rel="noopener noreferrer"
                          class="interview-card__meeting-link"
                        >
                          进入面试会议
                        </a>
                        <span v-else-if="item.mode === 'video'" class="interview-card__no-link">
                          暂无会议链接，请与HR确认
                        </span>
                      </div>
                      <span v-if="item.candidate_note" class="interview-card__note">{{ item.candidate_note }}</span>
                    </div>
                  </div>
                </article>
              </div>
            </el-scrollbar>
          </div>
        </section>

        <!-- ─── Tab 3: 录用通知 ─── -->
        <section v-if="activeTab === 'offers'" class="progress-section">
          <div v-loading="offerLoading" class="tab-content">
            <div class="section-heading">
              <div>
                <h2>录用通知</h2>
                <p>查看录用条件，并在有效期内完成决定。</p>
              </div>
            </div>
            <el-scrollbar class="tab-scroll">
              <el-empty v-if="!offerLoading && offers.length === 0" description="暂无Offer记录" />

              <article
                v-for="offer in offers"
                :key="offer.id"
                :id="`offer-card-${offer.id}`"
                class="offer-card"
                :class="{ 'offer-card--highlight': highlightOfferId === offer.id }"
              >
                <div class="offer-card-header">
                  <div class="offer-card-title">
                    <span class="application-card__hint">录用岗位</span>
                    <h3>{{ offer.job_title || offer.title }}</h3>
                    <p>{{ offer.title }}</p>
                  </div>
                  <div class="offer-status">
                    <el-tag :type="offerStatusType(offer.status) as any">{{ offerStatusLabel(offer.status) }}</el-tag>
                    <strong v-if="offer.status === 'sent' && offerDaysRemaining(offer.expires_at) !== null">
                      {{ offerDaysRemaining(offer.expires_at) === 0 ? '即将到期' : `剩余 ${offerDaysRemaining(offer.expires_at)} 天` }}
                    </strong>
                  </div>
                </div>

                <div class="offer-facts">
                  <div><span>薪资范围</span><strong>{{ offer.salary_range || '-' }}</strong></div>
                  <div><span>职级</span><strong>{{ offer.level || '-' }}</strong></div>
                  <div><span>工作地点</span><strong>{{ offer.work_location || '-' }}</strong></div>
                  <div><span>预计入职</span><strong>{{ offer.start_date || '-' }}</strong></div>
                  <div><span>有效期至</span><strong>{{ fmtDT(offer.expires_at) }}</strong></div>
                  <div><span>发送时间</span><strong>{{ fmtDT(offer.created_at) }}</strong></div>
                </div>

                <div v-if="offerTerms(offer).length" class="offer-terms">
                  <h4>录用条款</h4>
                  <dl>
                    <div v-for="term in offerTerms(offer)" :key="term.label">
                      <dt>{{ term.label }}</dt>
                      <dd>{{ term.value }}</dd>
                    </div>
                  </dl>
                </div>

                <div v-if="offer.status === 'sent'" class="offer-actions">
                  <span>请确认信息无误后再完成决定。</span>
                  <div>
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
                </div>
              </article>

              <div v-if="offerHasMore" class="pagination-wrapper pagination-wrapper--in-scroll">
                <el-button :loading="offerLoading" @click="loadOffers(false)">
                  加载更多 ({{ offers.length }} / {{ offerTotal }})
                </el-button>
              </div>
            </el-scrollbar>
          </div>
        </section>
        </main>
      </div>
    </div>
  </section>
</template>

<style scoped>
.job-progress {
  width: 100%;
  margin: 0 auto;
  height: calc(100dvh - 120px);
  overflow: hidden;
}
.progress-workspace {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 18px;
  background: var(--surface);
  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.08);
}
.progress-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding: 24px 28px 20px;
  border-bottom: 1px solid var(--border);
}
.progress-eyebrow {
  display: block;
  margin-bottom: 8px;
  color: var(--brand-strong);
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.14em;
}
.progress-header .page-title {
  margin-bottom: 8px;
  font-size: 28px;
  line-height: 1.2;
}
.progress-header .page-subtitle {
  margin: 0;
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.6;
}
.progress-body {
  display: grid;
  grid-template-columns: 250px minmax(0, 1fr);
  flex: 1 1 0;
  min-height: 0;
  overflow: hidden;
}
.progress-summary {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  align-content: start;
  gap: 12px;
  min-height: 0;
  padding: 20px 16px;
  overflow-y: auto;
  background: color-mix(in srgb, var(--surface-muted) 72%, var(--surface));
  border-right: 1px solid var(--border);
}
.summary-item {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  padding: 12px 14px;
  text-align: left;
  color: var(--text-primary);
  border: 1px solid transparent;
  border-radius: 12px;
  background: transparent;
  cursor: pointer;
  transition: border-color var(--motion-normal), transform var(--motion-normal), box-shadow var(--motion-normal);
}
.summary-item:hover,
.summary-item.active {
  border-color: color-mix(in srgb, var(--brand) 55%, var(--border));
  box-shadow: 0 8px 22px rgba(37, 99, 235, 0.1);
  transform: translateX(2px);
}
.summary-item.active {
  background: var(--surface);
}
.summary-icon {
  display: grid;
  flex: 0 0 34px;
  width: 34px;
  height: 34px;
  place-items: center;
  border-radius: 10px;
  color: var(--brand-strong);
  background: var(--brand-soft);
  font-size: 13px;
  font-weight: 800;
}
.summary-copy {
  display: block;
  flex: 1 1 auto;
  min-width: 0;
}
.summary-value {
  flex: 0 0 auto;
  font-size: 22px;
  font-weight: 800;
}
.summary-label {
  display: block;
  font-size: 13px;
  font-weight: 700;
}
.summary-item small {
  display: block;
  margin-top: 5px;
  overflow: hidden;
  color: var(--text-muted);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.summary-item--offer .summary-value {
  color: #dc2626;
}
.progress-content {
  flex: 1 1 0;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.progress-section {
  flex: 1 1 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.tab-content {
  height: 100%;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: 24px;
  box-sizing: border-box;
}
.tab-content .section-heading {
  flex-shrink: 0;
}
.tab-content .pagination-wrapper {
  flex-shrink: 0;
}
.tab-scroll {
  flex: 1 1 0;
  min-height: 0;
  height: 100%;
}
.tab-scroll :deep(.el-scrollbar__wrap) {
  overflow-x: hidden;
}
.tab-scroll :deep(.el-scrollbar__view) {
  box-sizing: border-box;
  padding-right: 4px;
  padding-bottom: 4px;
}

.desktop-progress-table {
  width: 100%;
}
.section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
}
.section-heading h2 {
  margin: 0 0 4px;
  font-size: 19px;
}
.section-heading p {
  margin: 0;
  color: var(--text-muted);
  font-size: 13px;
}
.mb-4 {
  margin-bottom: 16px;
}
.mt-2 {
  margin-top: 8px;
}
.offer-card {
  margin-bottom: 16px;
  padding: 22px;
  border: 1px solid var(--border);
  border-radius: 14px;
  background: var(--surface);
  transition: box-shadow 0.3s, background 0.3s, border-color 0.3s;
}
.offer-card--highlight {
  border-color: var(--brand);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--brand) 18%, transparent) !important;
  background: color-mix(in srgb, var(--brand-soft) 36%, var(--surface));
}
.offer-card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 16px;
}
.offer-card-title {
  min-width: 0;
}
.offer-card-title h3 {
  margin: 4px 0;
  font-size: 20px;
}
.offer-card-title p {
  margin: 0;
  color: var(--text-muted);
  font-size: 13px;
}
.offer-status {
  display: flex;
  align-items: flex-end;
  gap: 7px;
  flex-direction: column;
}
.offer-status strong {
  color: #dc2626;
  font-size: 12px;
}
.offer-facts {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1px;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--border);
}
.offer-facts > div {
  min-width: 0;
  padding: 13px 14px;
  background: var(--surface-muted);
}
.offer-facts span,
.offer-facts strong {
  display: block;
}
.offer-facts span {
  margin-bottom: 4px;
  color: var(--text-muted);
  font-size: 11px;
}
.offer-facts strong {
  overflow: hidden;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.offer-terms {
  margin-top: 16px;
  padding: 16px;
  border-radius: 10px;
  background: var(--surface-muted);
}
.offer-terms h4 {
  margin: 0 0 12px;
  font-size: 13px;
}
.offer-terms dl {
  margin: 0;
}
.offer-terms dl > div {
  display: grid;
  grid-template-columns: 100px 1fr;
  gap: 16px;
  padding: 7px 0;
}
.offer-terms dt {
  color: var(--text-muted);
  font-size: 12px;
}
.offer-terms dd {
  margin: 0;
  line-height: 1.6;
  word-break: break-word;
}
.offer-actions {
  margin-top: 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid var(--border);
}
.offer-actions > span {
  color: var(--text-muted);
  font-size: 12px;
}
.pagination-wrapper {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}
.interview-list {
  display: grid;
  gap: 10px;
}

/* ── Redesigned Interview Card ── */
.interview-card {
  display: flex;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 14px;
  transition: box-shadow var(--motion-normal), border-color var(--motion-normal);
}
.interview-card:hover {
  border-color: color-mix(in srgb, var(--brand) 35%, var(--border));
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.08);
}
.interview-card__date-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 72px;
  padding: 12px 10px;
  background: linear-gradient(180deg, var(--brand-soft) 0%, color-mix(in srgb, var(--brand-soft) 70%, var(--surface)) 100%);
  border-right: 1px solid var(--border);
}
.interview-card__date-badge {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}
.interview-card__month {
  color: var(--brand-strong);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.06em;
}
.interview-card__day {
  color: var(--brand-strong);
  font-size: 22px;
  font-weight: 800;
  line-height: 1;
}
.interview-card__time {
  padding: 4px 10px;
  border-radius: 20px;
  background: color-mix(in srgb, var(--brand) 12%, transparent);
  color: var(--brand-strong);
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}
.interview-card__body {
  flex: 1;
  min-width: 0;
  padding: 12px 16px;
}
.interview-card__top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}
.interview-card__title-area {
  min-width: 0;
}
.interview-card__meta-tag {
  display: inline-block;
  margin-bottom: 4px;
  color: var(--text-muted);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.05em;
}
.interview-card__title {
  margin: 0;
  font-size: 15px;
  line-height: 1.4;
}
.interview-card__subtitle {
  display: block;
  color: var(--text-muted);
  font-size: 12px;
  margin-top: 1px;
}
.interview-card__info-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 20px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--border);
}
.interview-card__info-item {
  min-width: 0;
}
.interview-card__info-label {
  display: block;
  margin-bottom: 3px;
  color: var(--text-muted);
  font-size: 11px;
}
.interview-card__info-value {
  display: block;
  font-size: 13px;
  font-weight: 700;
  color: var(--text-primary);
}
.interview-card__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--border);
}
.interview-card__footer-left {
  display: flex;
  align-items: center;
  gap: 8px;
}
.interview-card__link-label {
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 500;
  flex-shrink: 0;
}
.interview-card__meeting-link {
  color: var(--brand);
  font-size: 13px;
  font-weight: 600;
  text-decoration: none;
  transition: color var(--motion-fast) var(--motion-ease);
}
.interview-card__meeting-link:hover {
  color: var(--brand-strong);
  text-decoration: underline;
}
.interview-card__no-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: 20px;
  background: color-mix(in srgb, var(--text-muted) 10%, transparent);
  color: var(--text-muted);
  font-size: 12px;
  font-weight: 600;
}
.interview-card__note {
  overflow: hidden;
  color: var(--text-muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.application-card__hint {
  color: var(--text-muted);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.05em;
}
.mobile-application-list {
  display: none;
}

@media (max-width: 960px) {
  .job-progress {
    height: auto;
    overflow: visible;
  }
  .progress-workspace,
  .progress-body,
  .progress-content {
    height: auto;
    overflow: visible;
  }
  .tab-scroll {
    height: auto;
  }
  .progress-section,
  .tab-content {
    height: auto;
    overflow: visible;
  }
}

@media (max-width: 768px) {
  .progress-workspace {
    border-radius: 14px;
  }
  .progress-header {
    align-items: flex-start;
    padding: 20px 18px 18px;
  }
  .progress-header .page-title {
    font-size: 26px;
  }
  .progress-summary {
    grid-template-columns: minmax(0, 1fr);
    gap: 8px;
    padding: 12px 14px;
    border-right: none;
    border-bottom: 1px solid var(--border);
  }
  .progress-body {
    grid-template-columns: minmax(0, 1fr);
  }
  .summary-item {
    padding: 12px 10px;
  }
  .summary-value {
    font-size: 21px;
  }
  .tab-content {
    flex: none;
    display: block;
  }
  .tab-content {
    padding: 18px 14px;
  }
  .section-heading {
    align-items: flex-start;
  }
  .desktop-progress-table {
    display: none;
  }
  .mobile-application-list {
    display: grid;
    gap: 12px;
  }
  .application-card {
    padding: 16px;
    border: 1px solid var(--border);
    border-radius: 12px;
  }
  .application-card__top {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
  }
  .application-card h3 {
    margin: 4px 0 0;
    font-size: 16px;
    cursor: pointer;
  }
  .application-card__meta {
    display: flex;
    flex-wrap: wrap;
    gap: 6px 12px;
    margin-top: 12px;
    color: var(--text-muted);
    font-size: 12px;
  }
  .interview-card {
    flex-direction: column;
  }
  .interview-card__date-col {
    flex-direction: row;
    justify-content: flex-start;
    gap: 12px;
    min-width: 0;
    padding: 12px 16px;
    border-right: none;
    border-bottom: 1px solid var(--border);
  }
  .interview-card__date-badge {
    flex-direction: row;
    gap: 4px;
  }
  .interview-card__day {
    font-size: 22px;
  }
  .interview-card__info-grid {
    gap: 8px 16px;
  }
  .offer-facts {
    grid-template-columns: 1fr 1fr;
  }
  .interview-card__footer {
    align-items: flex-start;
    flex-direction: column;
  }
  .offer-card {
    padding: 17px;
  }
  .offer-card-header,
  .offer-actions {
    align-items: flex-start;
    flex-direction: column;
  }
  .offer-status {
    align-items: flex-start;
    flex-direction: row;
  }
  .offer-actions > div {
    display: flex;
    width: 100%;
  }
  .offer-actions .el-button {
    flex: 1;
  }
}

@media (max-width: 480px) {
  .progress-header .el-button {
    display: none;
  }
  .section-heading .el-button {
    display: none;
  }
  .summary-label {
    font-size: 12px;
  }
  .offer-facts {
    grid-template-columns: 1fr;
  }
  .offer-terms dl > div {
    grid-template-columns: 1fr;
    gap: 3px;
  }
  .interview-card__info-grid {
    flex-direction: column;
    gap: 6px;
  }
}
</style>
