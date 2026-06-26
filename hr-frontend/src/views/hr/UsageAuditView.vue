<script setup lang="ts">
import { ref, reactive, onMounted, computed, watch } from "vue";
import { ElMessage } from "element-plus";
import { listUsageLogs, getUsageStats, getUsageTrend } from "@/api/admin";
import type { UsageLogQuery, UsageLogItem, UsageStatsItem, UsageTrendPoint } from "@/api/admin";
import * as echarts from "echarts";

// ── Constants ────────────────────────────────────────────────────────────

const SERVICE_TYPE_MAP: Record<string, string> = {
    ai_chat: "AI 对话",
    ai_analyze: "AI 分析",
    oss_presign: "上传签名",
    oss_confirm: "上传确认",
};

const STATUS_TAG_TYPE: Record<string, string> = {
    ok: "success",
    error: "danger",
    timeout: "warning",
    rate_limited: "info",
};

const ROLE_MAP: Record<number, string> = { 1: "候选人", 2: "HR", 3: "管理员" };

const DIMENSION_OPTIONS = [
    { label: "按模型", value: "model" },
    { label: "按用户", value: "user" },
    { label: "按会话", value: "session" },
];

const GRANULARITY_OPTIONS = [
    { label: "按天", value: "day" },
    { label: "按周", value: "week" },
    { label: "按月", value: "month" },
];

// ── Log list state ───────────────────────────────────────────────────────

const loading = ref(false);
const errorMessage = ref("");
const logs = ref<UsageLogItem[]>([]);
const total = ref(0);

const query = reactive<UsageLogQuery>({
    page: 1,
    page_size: 20,
    service_type: "",
    provider: "",
    status: "",
    user_id: undefined,
    request_id: "",
    start_time: "",
    end_time: "",
});

const dateRange = ref<[string, string] | null>(null);

const load = async () => {
    loading.value = true;
    errorMessage.value = "";
    try {
        query.page = Number(query.page) || 1;
        query.page_size = Number(query.page_size) || 20;
        const data = await listUsageLogs(query);
        logs.value = data.list || [];
        total.value = Number(data.total) || 0;
    } catch (error: unknown) {
        errorMessage.value =
            error instanceof Error ? error.message : "加载审计日志失败";
    } finally {
        loading.value = false;
    }
};

const handleSearch = () => {
    query.page = 1;
    load();
};

const handleReset = () => {
    query.service_type = "";
    query.provider = "";
    query.status = "";
    query.user_id = undefined;
    query.request_id = "";
    query.start_time = "";
    query.end_time = "";
    dateRange.value = null;
    query.page = 1;
    load();
};

const handleDateChange = (val: [string, string] | null) => {
    if (val) {
        query.start_time = val[0];
        query.end_time = val[1];
    } else {
        query.start_time = "";
        query.end_time = "";
    }
};

const copyText = async (text: string) => {
    try {
        await navigator.clipboard.writeText(text);
        ElMessage.success("已复制");
    } catch {
        ElMessage.error("复制失败");
    }
};

const formatSize = (bytes: number): string => {
    if (!bytes) return "-";
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
};

// ── Stats state (P1-003) ─────────────────────────────────────────────────

const statsLoading = ref(false);
const statsItems = ref<UsageStatsItem[]>([]);
const trendPoints = ref<UsageTrendPoint[]>([]);
const activeDimension = ref<'user' | 'model' | 'session'>('model');
const activeGranularity = ref<'day' | 'week' | 'month'>('day');
const statsDateRange = ref<[string, string] | null>(null);

// Computed summary from stats
const totalTokens = computed(() =>
    statsItems.value.reduce((sum, item) => sum + item.total_tokens, 0)
);
const totalCalls = computed(() =>
    statsItems.value.reduce((sum, item) => sum + item.call_count, 0)
);
const avgCostMs = computed(() => {
    if (totalCalls.value === 0) return 0;
    const totalMs = statsItems.value.reduce((sum, item) => sum + item.avg_cost_ms * item.call_count, 0);
    return totalCalls.value > 0 ? totalMs / totalCalls.value : 0;
});
const estimatedCostTotal = computed(() =>
    statsItems.value.reduce((sum, item) => sum + item.estimated_cost, 0)
);

// Default stats date range: last 30 days
const getDefaultStatsRange = (): [string, string] => {
    const end = new Date();
    const start = new Date();
    start.setDate(start.getDate() - 30);
    return [start.toISOString(), end.toISOString()];
};

const loadStats = async () => {
    statsLoading.value = true;
    try {
        let startTime = "";
        let endTime = "";
        if (statsDateRange.value) {
            startTime = statsDateRange.value[0];
            endTime = statsDateRange.value[1];
        } else {
            const [s, e] = getDefaultStatsRange();
            startTime = s;
            endTime = e;
        }

        const [statsResp, trendResp] = await Promise.all([
            getUsageStats({
                start_time: startTime,
                end_time: endTime,
                dimension: activeDimension.value,
            }),
            getUsageTrend({
                start_time: startTime,
                end_time: endTime,
                granularity: activeGranularity.value,
            }),
        ]);
        statsItems.value = statsResp.list || [];
        trendPoints.value = trendResp.list || [];
    } catch (error: unknown) {
        ElMessage.error("加载统计数据失败");
    } finally {
        statsLoading.value = false;
    }
};

const handleStatsDateChange = () => {
    loadStats();
};

const handleDimensionChange = (dim: 'user' | 'model' | 'session') => {
    activeDimension.value = dim;
    loadStats();
};

const handleGranularityChange = (g: 'day' | 'week' | 'month') => {
    activeGranularity.value = g;
    // Reload only trend when granularity changes
    reloadTrend();
};

const reloadTrend = async () => {
    try {
        let startTime = "";
        let endTime = "";
        if (statsDateRange.value) {
            startTime = statsDateRange.value[0];
            endTime = statsDateRange.value[1];
        } else {
            const [s, e] = getDefaultStatsRange();
            startTime = s;
            endTime = e;
        }
        const trendResp = await getUsageTrend({
            start_time: startTime,
            end_time: endTime,
            granularity: activeGranularity.value,
        });
        trendPoints.value = trendResp.list || [];
    } catch {
        ElMessage.error("加载趋势数据失败");
    }
};

// ── ECharts ──────────────────────────────────────────────────────────────

const trendChartRef = ref<HTMLElement | null>(null);
let trendChart: echarts.ECharts | null = null;

const renderTrendChart = () => {
    if (!trendChartRef.value) return;
    if (!trendChart) {
        trendChart = echarts.init(trendChartRef.value);
    }

    const dates = trendPoints.value.map((p) => p.date);
    const tokens = trendPoints.value.map((p) => p.total_tokens);
    const calls = trendPoints.value.map((p) => p.call_count);

    trendChart.setOption({
        tooltip: {
            trigger: "axis",
        },
        legend: {
            data: ["Token 消耗", "调用次数"],
            top: 0,
        },
        grid: {
            left: "3%",
            right: "4%",
            bottom: "3%",
            containLabel: true,
        },
        xAxis: {
            type: "category",
            data: dates,
            axisLabel: { rotate: 45, fontSize: 11 },
        },
        yAxis: [
            {
                type: "value",
                name: "Token 数",
                nameTextStyle: { fontSize: 11 },
            },
            {
                type: "value",
                name: "调用次数",
                nameTextStyle: { fontSize: 11 },
            },
        ],
        series: [
            {
                name: "Token 消耗",
                type: "bar",
                data: tokens,
                itemStyle: { color: "#409eff" },
            },
            {
                name: "调用次数",
                type: "line",
                yAxisIndex: 1,
                data: calls,
                itemStyle: { color: "#67c23a" },
                smooth: true,
            },
        ],
    });
};

// Watch trend data changes and re-render chart
watch(trendPoints, () => {
    renderTrendChart();
}, { deep: true });

// ── Dimension chart (pie or list) ────────────────────────────────────────

const dimensionChartRef = ref<HTMLElement | null>(null);
let dimensionChart: echarts.ECharts | null = null;

const renderDimensionChart = () => {
    if (!dimensionChartRef.value) return;
    if (activeDimension.value === 'user' || activeDimension.value === 'session') return; // Only pie for model

    if (!dimensionChart) {
        dimensionChart = echarts.init(dimensionChartRef.value);
    }

    const pieData = statsItems.value.map((item) => ({
        name: item.name,
        value: item.total_tokens,
    }));

    dimensionChart.setOption({
        tooltip: {
            trigger: "item",
            formatter: "{b}: {c} tokens ({d}%)",
        },
        series: [
            {
                type: "pie",
                radius: ["30%", "60%"],
                center: ["50%", "50%"],
                data: pieData,
                label: {
                    formatter: "{b}\n{d}%",
                    fontSize: 11,
                },
                emphasis: {
                    itemStyle: {
                        shadowBlur: 10,
                        shadowOffsetX: 0,
                        shadowColor: "rgba(0, 0, 0, 0.5)",
                    },
                },
            },
        ],
    });
};

watch([statsItems, activeDimension], () => {
    renderDimensionChart();
}, { deep: true });

// Resize charts on window resize
const handleResize = () => {
    trendChart?.resize();
    dimensionChart?.resize();
};

// ── Lifecycle ────────────────────────────────────────────────────────────

onMounted(() => {
    load();
    loadStats();
    window.addEventListener("resize", handleResize);
});
</script>

<template>
    <div class="usage-audit">
        <!-- ═══════════════════════════════════════════════════════════════
             Stats Section (P1-003)
             ═══════════════════════════════════════════════════════════════ -->
        <div class="stats-section">
            <div class="stats-header">
                <h3>AI 使用统计</h3>
                <div class="stats-controls">
                    <el-date-picker
                        v-model="statsDateRange"
                        type="datetimerange"
                        range-separator="至"
                        start-placeholder="开始时间"
                        end-placeholder="结束时间"
                        value-format="YYYY-MM-DDTHH:mm:ssZ"
                        style="width: 340px"
                        @change="handleStatsDateChange"
                    />
                    <el-button
                        type="primary"
                        size="small"
                        :loading="statsLoading"
                        @click="loadStats"
                    >
                        刷新
                    </el-button>
                </div>
            </div>

            <!-- Stats cards -->
            <el-row :gutter="16" class="stats-cards">
                <el-col :span="6">
                    <el-card shadow="hover" class="stats-card" v-loading="statsLoading">
                        <div class="stats-card-label">总 Token 消耗</div>
                        <div class="stats-card-value">{{ totalTokens.toLocaleString() }}</div>
                    </el-card>
                </el-col>
                <el-col :span="6">
                    <el-card shadow="hover" class="stats-card" v-loading="statsLoading">
                        <div class="stats-card-label">总调用次数</div>
                        <div class="stats-card-value">{{ totalCalls.toLocaleString() }}</div>
                    </el-card>
                </el-col>
                <el-col :span="6">
                    <el-card shadow="hover" class="stats-card" v-loading="statsLoading">
                        <div class="stats-card-label">平均耗时 (ms)</div>
                        <div class="stats-card-value">{{ avgCostMs.toFixed(1) }}</div>
                    </el-card>
                </el-col>
                <el-col :span="6">
                    <el-card shadow="hover" class="stats-card" v-loading="statsLoading">
                        <div class="stats-card-label">花费估算 ($)</div>
                        <div class="stats-card-value">{{ estimatedCostTotal.toFixed(4) }}</div>
                    </el-card>
                </el-col>
            </el-row>

            <!-- Trend chart & dimension analysis -->
            <el-row :gutter="16" class="stats-charts">
                <el-col :span="14">
                    <el-card shadow="hover">
                        <template #header>
                            <div class="chart-header">
                                <span>Token 消耗趋势</span>
                                <el-radio-group
                                    :model-value="activeGranularity"
                                    size="small"
                                    @change="handleGranularityChange"
                                >
                                    <el-radio-button
                                        v-for="opt in GRANULARITY_OPTIONS"
                                        :key="opt.value"
                                        :value="opt.value"
                                    >
                                        {{ opt.label }}
                                    </el-radio-button>
                                </el-radio-group>
                            </div>
                        </template>
                        <div
                            ref="trendChartRef"
                            class="chart-container"
                            v-loading="statsLoading"
                        />
                    </el-card>
                </el-col>
                <el-col :span="10">
                    <el-card shadow="hover">
                        <template #header>
                            <div class="chart-header">
                                <span>维度分布</span>
                                <el-radio-group
                                    :model-value="activeDimension"
                                    size="small"
                                    @change="handleDimensionChange"
                                >
                                    <el-radio-button
                                        v-for="opt in DIMENSION_OPTIONS"
                                        :key="opt.value"
                                        :value="opt.value"
                                    >
                                        {{ opt.label }}
                                    </el-radio-button>
                                </el-radio-group>
                            </div>
                        </template>
                        <!-- Pie chart for model dimension -->
                        <div
                            v-if="activeDimension === 'model'"
                            ref="dimensionChartRef"
                            class="chart-container"
                            v-loading="statsLoading"
                        />
                        <!-- Table for user/session dimension -->
                        <el-table
                            v-else
                            :data="statsItems"
                            v-loading="statsLoading"
                            stripe
                            size="small"
                            max-height="300"
                            empty-text="暂无数据"
                        >
                            <el-table-column
                                prop="name"
                                :label="activeDimension === 'user' ? '用户 ID' : '会话 ID'"
                                min-width="100"
                                show-overflow-tooltip
                            />
                            <el-table-column
                                prop="total_tokens"
                                label="Token"
                                width="100"
                                align="center"
                            >
                                <template #default="{ row }">
                                    {{ row.total_tokens.toLocaleString() }}
                                </template>
                            </el-table-column>
                            <el-table-column
                                prop="call_count"
                                label="调用次数"
                                width="80"
                                align="center"
                            />
                            <el-table-column
                                prop="avg_cost_ms"
                                label="平均耗时"
                                width="90"
                                align="center"
                            >
                                <template #default="{ row }">
                                    {{ row.avg_cost_ms.toFixed(1) }} ms
                                </template>
                            </el-table-column>
                        </el-table>
                    </el-card>
                </el-col>
            </el-row>
        </div>

        <!-- ═══════════════════════════════════════════════════════════════
             Log List Section (existing)
             ═══════════════════════════════════════════════════════════════ -->
        <el-divider content-position="left">详细日志</el-divider>

        <el-form
            :inline="true"
            class="filter-form"
            @submit.prevent="handleSearch"
        >
            <el-form-item label="服务类型">
                <el-select
                    v-model="query.service_type"
                    clearable
                    placeholder="全部"
                    style="width: 130px"
                >
                    <el-option label="AI 对话" value="ai_chat" />
                    <el-option label="AI 分析" value="ai_analyze" />
                    <el-option label="上传签名" value="oss_presign" />
                    <el-option label="上传确认" value="oss_confirm" />
                </el-select>
            </el-form-item>
            <el-form-item label="供应商">
                <el-select
                    v-model="query.provider"
                    clearable
                    placeholder="全部"
                    style="width: 140px"
                >
                    <el-option label="DashScope" value="dashscope" />
                    <el-option label="腾讯 COS" value="tencent_cos" />
                    <el-option label="阿里云 OSS" value="aliyun_oss" />
                </el-select>
            </el-form-item>
            <el-form-item label="状态">
                <el-select
                    v-model="query.status"
                    clearable
                    placeholder="全部"
                    style="width: 120px"
                >
                    <el-option label="成功" value="ok" />
                    <el-option label="错误" value="error" />
                    <el-option label="超时" value="timeout" />
                    <el-option label="限流" value="rate_limited" />
                </el-select>
            </el-form-item>
            <el-form-item label="用户 ID">
                <el-input
                    v-model.number="query.user_id"
                    placeholder="用户 ID"
                    style="width: 110px"
                    clearable
                />
            </el-form-item>
            <el-form-item label="Request ID">
                <el-input
                    v-model="query.request_id"
                    placeholder="Request ID"
                    style="width: 160px"
                    clearable
                />
            </el-form-item>
            <el-form-item label="时间范围">
                <el-date-picker
                    v-model="dateRange"
                    type="datetimerange"
                    range-separator="至"
                    start-placeholder="开始时间"
                    end-placeholder="结束时间"
                    value-format="YYYY-MM-DDTHH:mm:ssZ"
                    style="width: 360px"
                    @change="handleDateChange"
                />
            </el-form-item>
            <el-form-item>
                <el-button type="primary" @click="handleSearch">查询</el-button>
                <el-button @click="handleReset">重置</el-button>
            </el-form-item>
        </el-form>

        <el-alert
            v-if="errorMessage"
            class="page-error"
            type="error"
            :title="errorMessage"
            show-icon
            :closable="false"
        >
            <template #default>
                <el-button size="small" type="danger" plain @click="load"
                    >重试</el-button
                >
            </template>
        </el-alert>

        <div class="table-wrapper">
            <el-table
                v-loading="loading"
                :data="logs"
                empty-text="暂无审计日志"
                stripe
                height="100%"
            >
                <el-table-column
                    prop="created_at"
                    label="时间"
                    min-width="170"
                    align="center"
                >
                    <template #default="{ row }">
                        {{
                            row.created_at?.replace("T", " ").slice(0, 19) ||
                            "-"
                        }}
                    </template>
                </el-table-column>
                <el-table-column
                    prop="user_id"
                    label="用户ID"
                    width="90"
                    align="center"
                />
                <el-table-column
                    prop="role"
                    label="角色"
                    width="80"
                    align="center"
                >
                    <template #default="{ row }">{{
                        ROLE_MAP[row.role] || row.role
                    }}</template>
                </el-table-column>
                <el-table-column
                    prop="service_type"
                    label="服务类型"
                    width="100"
                    align="center"
                >
                    <template #default="{ row }">{{
                        SERVICE_TYPE_MAP[row.service_type] || row.service_type
                    }}</template>
                </el-table-column>
                <el-table-column
                    prop="provider"
                    label="供应商"
                    width="110"
                    align="center"
                />
                <el-table-column
                    prop="model"
                    label="模型"
                    min-width="130"
                    show-overflow-tooltip
                />
                <el-table-column
                    prop="status"
                    label="状态"
                    width="80"
                    align="center"
                >
                    <template #default="{ row }">
                        <el-tag
                            :type="
                                (STATUS_TAG_TYPE[row.status] as any) || 'info'
                            "
                            size="small"
                        >
                            {{ row.status }}
                        </el-tag>
                    </template>
                </el-table-column>
                <el-table-column
                    prop="estimated_tokens"
                    label="估算 Token"
                    width="100"
                    align="center"
                />
                <el-table-column
                    prop="object_size"
                    label="对象大小"
                    width="100"
                    align="center"
                >
                    <template #default="{ row }">{{
                        formatSize(row.object_size)
                    }}</template>
                </el-table-column>
                <el-table-column
                    prop="cost_ms"
                    label="耗时(ms)"
                    width="90"
                    align="center"
                />
                <el-table-column
                    prop="ip"
                    label="IP"
                    width="130"
                    align="center"
                    show-overflow-tooltip
                />
                <el-table-column
                    prop="request_id"
                    label="Request ID"
                    min-width="120"
                    align="center"
                >
                    <template #default="{ row }">
                        <el-tooltip
                            v-if="row.request_id"
                            :content="row.request_id"
                            placement="top"
                        >
                            <span
                                class="copyable"
                                @click="copyText(row.request_id)"
                            >
                                {{ row.request_id.slice(0, 8) }}...
                            </span>
                        </el-tooltip>
                        <span v-else>-</span>
                    </template>
                </el-table-column>
                <el-table-column
                    prop="endpoint"
                    label="接口"
                    min-width="160"
                    show-overflow-tooltip
                />
                <el-table-column
                    prop="object_key"
                    label="Object Key"
                    min-width="160"
                >
                    <template #default="{ row }">
                        <el-tooltip
                            v-if="row.object_key"
                            :content="row.object_key"
                            placement="top"
                        >
                            <span
                                class="copyable"
                                @click="copyText(row.object_key)"
                            >
                                {{
                                    row.object_key.length > 20
                                        ? row.object_key.slice(0, 20) + "..."
                                        : row.object_key
                                }}
                            </span>
                        </el-tooltip>
                        <span v-else>-</span>
                    </template>
                </el-table-column>
            </el-table>
        </div>

        <div class="pagination-wrapper">
            <el-pagination
                v-model:current-page="query.page"
                v-model:page-size="query.page_size"
                layout="total, prev, pager, next, sizes"
                :total="total"
                :page-sizes="[10, 20, 50, 100]"
                @current-change="load"
                @size-change="load"
            />
        </div>
    </div>
</template>

<style scoped>
.usage-audit {
    height: 100%;
    display: flex;
    flex-direction: column;
    overflow-y: auto;
    padding: 16px;
}

/* ── Stats Section ───────────────────────────────────────────────────── */

.stats-section {
    flex-shrink: 0;
    margin-bottom: 8px;
}

.stats-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
}

.stats-header h3 {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
    color: var(--el-text-color-primary);
}

.stats-controls {
    display: flex;
    align-items: center;
    gap: 8px;
}

.stats-cards {
    margin-bottom: 16px;
}

.stats-card {
    text-align: center;
}

.stats-card-label {
    font-size: 13px;
    color: var(--el-text-color-secondary);
    margin-bottom: 8px;
}

.stats-card-value {
    font-size: 22px;
    font-weight: 700;
    color: var(--el-color-primary);
    font-family: "SF Mono", "Menlo", "Monaco", monospace;
}

.stats-charts {
    margin-bottom: 8px;
}

.chart-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.chart-container {
    width: 100%;
    height: 280px;
}

/* ── Log List Section (unchanged) ──────────────────────────────────────── */

.filter-form {
    flex-shrink: 0;
    margin-bottom: 12px;
}
.filter-form :deep(.el-form-item) {
    margin-bottom: 8px;
}
.page-error {
    flex-shrink: 0;
    margin-bottom: 12px;
}

.table-wrapper {
    flex: 1;
    min-height: 200px;
    overflow: hidden;
}

.copyable {
    cursor: pointer;
    color: var(--el-color-primary);
    font-family: monospace;
}
.copyable:hover {
    text-decoration: underline;
}

.pagination-wrapper {
    flex-shrink: 0;
    display: flex;
    justify-content: flex-end;
    margin-top: 12px;
}
</style>
