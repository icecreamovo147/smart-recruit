# AI 付费能力与支付宝沙箱联调

当前开发版本只允许支付宝沙箱，支付服务会拒绝 `alipay.environment` 不是 `sandbox` 的启动配置。沙箱订单、支付、回调均写入 `payment_environment=sandbox`，前端持续显示沙箱提示，经营统计必须排除这些记录。

## start-dev.sh 本地配置

复制示例文件并填写支付宝沙箱参数：

```bash
cp smart-recruit-billing-service/internal/config/config.example.yaml smart-recruit-billing-service/internal/config/config.yaml
```

`config.yaml` 已被 Git 忽略。`start-dev.sh` 启动 Billing Service 时会显式读取该文件；如果文件不存在会直接给出错误，不会使用示例值启动。RSA2 密钥推荐通过相对于 `config.yaml` 的 `private_key_file` 和 `verify_public_key_file` 配置。应用私钥文件必须为 `0600`；验签公钥必须是沙箱控制台提供的“支付宝公钥”，不能是由应用私钥派生的应用公钥，服务启动时会主动检测并拒绝这种误配。

如需使用其他路径，可以在启动脚本前设置 `BILLING_CONFIG_PATH`。完整字段与注释以 `smart-recruit-billing-service/internal/config/config.example.yaml` 为准。

## Docker/兼容环境变量

未传递 `--config` 的容器部署仍可使用以下环境变量：

```dotenv
AI_BILLING_MODE=shadow
BILLING_GRPC_ADDR=127.0.0.1:50069
ALIPAY_ENV=sandbox
ALIPAY_REQUIRED=true
ALIPAY_GATEWAY_URL=https://openapi-sandbox.dl.alipaydev.com/gateway.do
ALIPAY_APP_ID=<沙箱应用 ID>
ALIPAY_PRIVATE_KEY=<应用 RSA2 私钥，PKCS#8 或 PKCS#1 PEM>
ALIPAY_VERIFY_PUBLIC_KEY=<支付宝 RSA2 公钥 PEM>
ALIPAY_SELLER_ID=<沙箱卖家 PID>
ALIPAY_NOTIFY_URL=https://<公网联调域名>/api/v1/public/billing/webhooks/alipay
ALIPAY_RETURN_URL=http://localhost:8080/api/v1/public/billing/returns/alipay
BILLING_HR_RETURN_URL=http://localhost:5173/hr/billing
BILLING_CANDIDATE_RETURN_URL=http://localhost:5174/candidate/billing
ALIPAY_DESKTOP_ENABLED=true
ALIPAY_WAP_ENABLED=true
```

`ALIPAY_NOTIFY_URL` 必须是支付宝沙箱可以访问的 HTTPS 地址。本地开发可使用受控的 HTTPS 隧道，但不要把隧道令牌或密钥写入仓库。

支付宝开放平台要求沙箱使用沙箱 App ID、RSA2 应用私钥和对应的支付宝公钥。沙箱网关以开放平台当前沙箱控制台显示值为准；本项目默认使用 `https://openapi-sandbox.dl.alipaydev.com/gateway.do`。

## 联调顺序

1. 执行 Commons 迁移，启动 Billing Service、AI Agent Service、Gateway 和三个前端。
2. 在平台管理端“套餐与配额”页面发布与 AI 运行时 `provider_key/model_key` 一致的模型费率卡，再为候选人 Pro、企业套餐或加量包创建价格版本并发布。
3. 在候选人端“AI 套餐”或 HR 端“AI 套餐与额度”创建订单。
4. 使用支付宝沙箱买家账号完成桌面网页或 WAP 支付。
5. 等待异步通知。同步 `return_url` 固定进入 Gateway，Gateway 验签后根据支付尝试的 `source_app` 跳回 HR 或候选人端，并签发一次性返回令牌。前端使用登录态和该令牌主动查单；同步返回本身不能直接结算订单。
6. 验证订单为 `paid`、订阅或额度已生效、`ai_credit_ledger` 存在发放流水、`billing_webhook_events.signature_verified=1`（或主动查单写入的 `active_query` / return 同步入账）。
7. 对完全未消费订单申请退款，应自动调用沙箱退款；存在已结算额度、续费或升级订单进入平台“套餐与商业化 → 退款审批”。退款请求必须携带幂等键，超时后使用原 `refund_no/out_request_no` 查询，禁止换号重试。

## 阶段开关

- 本地配置 `billing.mode: shadow`：Billing Service 记录真实用量和供应商成本，但不因额度不足拒绝预占。
- 本地配置 `billing.mode: enforce`：Billing Service 在调用模型前强制预占额度；权益关闭或余额不足时拒绝调用。
- AI Agent Service 的故障关闭策略仍由进程变量 `AI_BILLING_MODE` 控制。切换到强制模式时，应使用 `AI_BILLING_MODE=enforce ./start-dev.sh ...`，确保 Billing Service 不可用时 AI 也会拒绝调用。

从 `shadow` 切换到 `enforce` 前，应至少完成一个完整自然月的成本数据校准，发布正式费率卡和价格版本，并确认所有存量试点租户已有 AI 权益与月度额度。当前阶段不包含生产支付宝密钥、自动续费、后付费、发票或生产营收统计。

## 支付安全约束

- 只接受 RSA2 验签通过且 `app_id`、`seller_id`、商户订单号、人民币金额、交易状态和沙箱环境完全匹配的异步通知。
- 回调通过 `notify_id` 与沙箱环境联合幂等；重复通知不会重复发放订阅或额度。
- 只有 `TRADE_SUCCESS` 或 `TRADE_FINISHED` 进入成功处理。
- 查询、关单、退款及退款查询响应也必须通过支付宝公钥验签；不能只验证异步通知。
- 未收到异步通知时，Billing Service 使用数据库租约和 `SKIP LOCKED` 领取支付/退款对账任务，按指数退避持续核对；未知状态不会创建新的支付或退款编号。
- 同一所有者最多一个开放订单，同一订单最多一个活跃支付，同一支付最多一个活跃退款，均由数据库唯一约束保证。
- 回调只有在验签并完成数据库事务后才返回精确的 `200 success`；签名错误返回 4xx，内部失败返回 5xx。
- 已发布价格、用量事件和额度流水不可原地修改；纠错使用新价格版本和补偿流水。

参考：[支付宝沙箱使用说明](https://developer.alibaba.com/docs/doc.htm?articleId=105311&docType=1&treeId=292)、[支付宝异步通知与主动查询要求](https://aipay.alipay.com/docs/ai-web-app-payment-qianyi/api-list/async-notify-verify.html)。
