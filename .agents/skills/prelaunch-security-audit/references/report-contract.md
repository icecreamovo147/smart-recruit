# Security Audit Report Contract

## Contents

1. Canonical JSON
2. Severity and confidence
3. Release gate
4. Coverage and check status
5. Evidence quality

## 1. Canonical JSON

Use one JSON document as the source for all rendered artifacts:

```json
{
  "schema_version": 1,
  "audit": {
    "title": "Smart Recruit 上线前安全审查报告",
    "repo": "smart-recruit",
    "branch": "main",
    "commit": "0123456789abcdef",
    "working_tree": "clean | dirty",
    "generated_at": "2026-07-21T10:00:00+08:00",
    "scope": "当前仓库与已提交部署资产",
    "authorized_test_level": "L0",
    "standards": ["OWASP ASVS 5.0 L2"],
    "limitations": ["未提供生产集群只读访问"],
    "verdict": "NO_GO | CONDITIONAL_GO | GO",
    "overall_risk": "CRITICAL | HIGH | MEDIUM | LOW"
  },
  "executive_summary": ["摘要事实"],
  "attack_surface": [
    {"surface": "Gateway", "assets": ["JWT"], "entry_points": ["/api/v1"], "trust_boundaries": ["Browser -> Gateway"], "notes": "..."}
  ],
  "coverage": [
    {"id": "authentication-session", "domain": "认证与会话", "status": "REVIEWED", "evidence": ["path:line"], "limitations": []}
  ],
  "checks": [
    {"id": "CHK-001", "domain": "依赖安全", "command": "govulncheck ./...", "status": "PASS", "result": "未发现可达漏洞", "evidence": []}
  ],
  "positive_controls": [
    {"title": "刷新令牌轮换", "evidence": ["path:line"], "notes": "已验证的控制边界"}
  ],
  "findings": [
    {
      "id": "SEC-001",
      "title": "生产管理端点公开",
      "severity": "HIGH",
      "confidence": "CONFIRMED",
      "category": "Exposure",
      "cwe": ["CWE-200"],
      "standards": ["OWASP ASVS"],
      "release_blocker": true,
      "affected_assets": ["Gateway"],
      "locations": [{"path": "smart-recruit-gateway/router/router.go", "line": 106, "symbol": "NewRouter"}],
      "evidence": ["无需认证即可注册 metrics 路由；未执行线上请求"],
      "attack_scenario": "外部攻击者枚举运行指标。",
      "impact": "泄露内部运行信息并辅助攻击。",
      "likelihood": "HIGH",
      "recommendation": "将管理端点迁移至内部监听面并增加网络策略。",
      "verification": ["公网入口返回 404/403", "内部监控仍可采集"],
      "status": "OPEN"
    }
  ],
  "residual_risks": ["..."],
  "remediation": {
    "strategy": ["先处理发布阻断项"],
    "tasks": []
  }
}
```

Required top-level arrays may be empty but must be present. Do not omit incomplete coverage; represent it explicitly.

Coverage must contain these stable IDs exactly once; display names may remain Chinese:

- `architecture-threat-model`
- `authentication-session`
- `authorization-tenancy`
- `gateway-api-browser`
- `grpc-service-mq`
- `resume-file-oss`
- `ai-llm-mcp`
- `billing-integrations`
- `data-privacy-recovery`
- `secrets-config-crypto`
- `supply-chain-cicd`
- `runtime-kubernetes-network`
- `availability-detection-ir`

## 2. Severity and confidence

### Severity

| Severity | Meaning |
|---|---|
| `CRITICAL` | Internet-reachable compromise, auth bypass, platform takeover, cross-tenant bulk disclosure, remote code execution, material payment compromise, or production secret compromise with immediate broad impact |
| `HIGH` | Practical privilege escalation, single/cross-tenant sensitive disclosure, serious SSRF/tool misuse, durable session revocation failure, exploitable dependency or unsafe deployment exposure |
| `MEDIUM` | Exploitation needs additional conditions or has bounded impact; meaningful defense, monitoring, privacy, or availability gap |
| `LOW` | Limited impact hardening issue with no practical exploit chain established |
| `INFO` | Positive control, observation, or improvement opportunity; normally keep positive controls outside findings |

### Confidence

| Confidence | Evidence requirement |
|---|---|
| `CONFIRMED` | Reproducible source/config/test evidence demonstrates the condition |
| `HIGH` | Strong evidence with a small unverified environmental dependency |
| `MEDIUM` | Plausible issue requiring one important verification step |
| `LOW` | Lead only; normally record as a limitation or follow-up instead of a formal finding |

Use `likelihood`: `HIGH`, `MEDIUM`, or `LOW`. Finding `status` is `OPEN`, `MITIGATED`, `ACCEPTED`, or `FALSE_POSITIVE`.

## 3. Release gate

Set `release_blocker: true` for any unresolved condition involving:

- `CRITICAL` severity;
- confirmed exploitable `HIGH` severity;
- authentication bypass or platform/tenant-admin privilege escalation;
- cross-tenant read/write/export;
- leaked live credentials or personal data;
- exposed production datastore, queue, internal gRPC, admin, debug, metrics, tracing, or configuration plane with meaningful impact;
- AI/MCP private-network access, credential exfiltration, or unauthorized high-impact Tool execution;
- payment signature, amount, replay, entitlement, or refund authorization failure;
- default/placeholder production credentials or insecure-development mode;
- absent recovery capability for critical data;
- untraceable/untrusted release artifacts;
- missing evidence for a critical trust boundary that prevents a responsible security conclusion.

The renderer enforces:

- `GO` cannot contain an open release blocker.
- `GO` requires every mandatory coverage domain to be `REVIEWED` or justified as `NOT_APPLICABLE`.
- `CONDITIONAL_GO` cannot contain an open `CRITICAL` finding.
- `NO_GO` must contain a blocker or explain a critical evidence gap in limitations.

## 4. Coverage and check status

Coverage status:

- `REVIEWED`: meaningful code/config/test evidence examined.
- `PARTIAL`: some evidence examined, but material paths or environment evidence are missing.
- `NOT_REVIEWED`: no meaningful assessment completed.
- `NOT_APPLICABLE`: justified non-applicability.

Check status:

- `PASS`, `FAIL`, `NOT_RUN`, `BLOCKED`, `NOT_APPLICABLE`.

Never convert `NOT_RUN` or `BLOCKED` into `PASS` in summary language.

## 5. Evidence quality

- Use repository-relative POSIX paths.
- Prefer exact line/symbol and concise command result.
- Do not include secret values, personal data, raw attack payloads against live systems, or excessive scanner output.
- State whether a route/config condition was verified statically, locally reproduced, or dynamically confirmed in staging.
- Describe false positives and accepted risks explicitly; do not silently delete their history from a rerun.
