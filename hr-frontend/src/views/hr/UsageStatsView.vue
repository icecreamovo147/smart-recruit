<script setup lang="ts">
import { ref, computed, watch, onMounted } from "vue";
import { ElMessage } from "element-plus";
import { getUsageStats, getUsageTrend } from "@/api/admin";
import type { UsageStatsItem, UsageTrendPoint } from "@/api/admin";
import * as echarts from "echarts";

// ── Constants ────────────────────────────────────────────────────────────

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

// ── Stats state ─────────────────────────────────────────────────────────

const statsLoading = ref(false);
const statsItems = ref<UsageStatsItem[]>([]);
const trendPoints = ref<UsageTrendPoint[]>([]);
const activeDimension = ref<'user' | 'model' | 'session'>('model');
const activeGranularity = ref<'day' | 'week' | 'month'>('day');
const statsDateRange = ref<[string, string] | null>(null);

// Computed summary — NB: protojson serializes int64 as strings.
const toNum = (v: unknown): number => Number(v) || 0;

const totalTokens = computed(() =>
    statsItems.value.reduce((sum, item) => sum + toNum(item.total_tokens), 0)
);
const totalCalls = computed(() =>
    statsItems.value.reduce((sum, item) => sum + toNum(item.call_count), 0)
);
const avgCostMs = computed(() => {
    if (totalCalls.value === 0) return 0;
    const totalMs = statsItems.value.reduce((sum, item) => sum + toNum(item.avg_cost_ms) * toNum(item.call_count), 0);
    return totalCalls.value > 0 ? totalMs / totalCalls.value : 0;
});
const estimatedCostTotal = computed(() =>
    statsItems.value.reduce((sum, item) => sum + toNum(item.estimated_cost), 0)
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

// ── ECharts Trend ────────────────────────────────────────────────────────

const trendChartRef = ref<HTMLElement | null>(null);
let trendChart: echarts.ECharts | null = null;

const renderTrendChart = () => {
    if (!trendChartRef.value) return;
    if (!trendChart) {
        trendChart = echarts.init(trendChartRef.value);
    }

    const dates = trendPoints.value.map((p) => p.date?.slice(0, 10));
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

watch(trendPoints, () => {
    renderTrendChart();
}, { deep: true });

// ── Dimension chart ──────────────────────────────────────────────────────

const dimensionChartRef = ref<HTMLElement | null>(null);
let dimensionChart: echarts.ECharts | null = null;

const renderDimensionChart = () => {
    if (!dimensionChartRef.value) return;
    if (activeDimension.value === 'user' || activeDimension.value === 'session') return;

    // Dispose and re-init when DOM node changed (v-if destroys/recreates the element)
    if (dimensionChart) {
        dimensionChart.dispose();
        dimensionChart = null;
    }
    dimensionChart = echarts.init(dimensionChartRef.value);

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

const handleResize = () => {
    trendChart?.resize();
    dimensionChart?.resize();
};

// ── Lifecycle ────────────────────────────────────────────────────────────

onMounted(() => {
    loadStats();
    window.addEventListener("resize", handleResize);
});
</script>

<template>
    <div class="usage-stats">
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
</template>

<style scoped>
.usage-stats {
    padding: 16px;
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
</style>
