<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { ElMessage } from "element-plus";
import { listUsageLogs } from "@/api/admin";
import type { UsageLogQuery, UsageLogItem } from "@/api/admin";

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

onMounted(load);
</script>

<template>
    <div class="usage-audit">
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
    overflow: hidden;
    padding: 16px;
}

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
    min-height: 0;
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
