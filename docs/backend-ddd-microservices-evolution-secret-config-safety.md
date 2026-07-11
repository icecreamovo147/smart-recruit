# Backend Secret And Config Safety Baseline

本文档记录 `backend-ddd-microservices-evolution` 的后端 secret 与配置安全基线，对应 TASK-BDME-002。

## 目标

- 仓库内只保留占位示例，不提交 live secret、token、私钥或生产相似凭证。
- 生产环境必须通过外部 secret 注入关键配置。
- 本地开发可以使用 `ALLOW_INSECURE_DEV_CONFIG=true` 放宽校验，但该开关不得用于生产。
- HTTP 网关和 gRPC logic 服务之间必须使用 `GRPC_INTERNAL_TOKEN`，logic 生产环境必须设置 `GRPC_INTERNAL_AUTH=required`。
- 生产或预发环境的内部 gRPC 通信必须设置 `GRPC_INTERNAL_TLS=required`，并为 logic 服务挂载 server cert/key，为 gateway 挂载 CA bundle。

## 生产必填项

logic 服务生产启动会拒绝以下缺失值、`CHANGE_ME*` 占位值和已知弱默认值：

- `MYSQL_DSN`
- `JWT_SECRET`
- `GRPC_INTERNAL_AUTH=required`
- `GRPC_INTERNAL_TOKEN`
- `GRPC_INTERNAL_TLS=required` 时必须设置 `GRPC_TLS_CERT_FILE` 和 `GRPC_TLS_KEY_FILE`
- `ENCRYPTION_KEY`
- `RABBITMQ_URL`
- `OSS_ACCESS_KEY_ID`
- `OSS_ACCESS_KEY_SECRET`
- `OSS_BUCKET_NAME`

web 网关生产启动会拒绝以下缺失值、占位值和过短值：

- `JWT_SECRET`
- `GRPC_INTERNAL_TOKEN`
- `GRPC_INTERNAL_TLS=required` 时必须设置 `GRPC_TLS_CA_FILE`

`RABBITMQ_URL` 在生产环境不允许使用 `guest:guest` 默认凭证。`SMTP_REQUIRED=true` 时，logic 服务还会要求 `SMTP_HOST`、`SMTP_USERNAME`、`SMTP_PASSWORD` 和 `SMTP_FROM_ADDRESS` 均已注入真实值。

## 本地开发

本地 Docker Compose 已显式设置 `ALLOW_INSECURE_DEV_CONFIG=true`，因此可以继续用 `docker/.env.example` 复制出 `docker/.env` 后填入本地专用值。不要提交 `docker/.env`、`logic-grpc-service/config/config.yaml`、`deploy/k8s/secret.yaml` 或任何 `*.local.yaml`。

本地 Docker Compose 默认设置 `GRPC_INTERNAL_TLS=optional`，不要求挂载 gRPC TLS 证书；生产和预发环境不得依赖该本地兼容默认值。

裸 Go 本地运行如需使用示例 YAML，可设置：

```bash
ALLOW_INSECURE_DEV_CONFIG=true
```

生产、预发和共享测试环境不得设置该开关。

## 示例文件规则

- `logic-grpc-service/config/config.example.yaml` 只保留占位值。
- `docker/.env.example` 只保留 `CHANGE_ME_*` 占位值和非敏感默认值。
- `deploy/k8s/secret.example.yaml` 只保留 `CHANGE_ME_*` 占位值；部署前必须复制为未跟踪的 secret manifest 或由平台 secret 管理系统生成。
- `deploy/k8s/secret.example.yaml` 中的 `recruitment-grpc-tls` 只展示 `ca.crt`、`tls.crt`、`tls.key` 结构；生产证书和私钥必须由平台 secret 管理系统注入。
- `deploy/k8s/configmap.yaml` 不应包含 secret；如果其中仍有 `CHANGE_ME_*` 运行时配置，生产发布前必须替换为环境专属非敏感配置，否则服务会 fail fast。

## 验证

本任务覆盖的校验包括：

- logic config 单元测试覆盖生产 secret 完整、内部认证缺失、内部 TLS 文件缺失、RabbitMQ 默认凭证、JWT 占位和本地 bypass。
- logic/server 与 web/rpc 单元测试覆盖内部 gRPC TLS 必填配置、证书/CA 加载、缺失项 fail fast。
- web config 单元测试覆盖内部 token 缺失、占位 token、内部 TLS CA 缺失和本地 bypass。
- Harness scope check 保证没有修改 live `.env`、前端、依赖、schema 或契约文件。
