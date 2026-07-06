# ADR: MCP SDK Selection and Security Sandbox

## Date

2026-06-27

## Status

Accepted

## Context

The smart-recruit system needs to support the Model Context Protocol (MCP) to allow
dynamic registration and execution of external tools. This enables LLM agents to
interact with third-party services, databases, and APIs through a standardized
protocol.

Key requirements:
1. Support stdio, SSE, and HTTP transport for MCP servers
2. Dynamic tool discovery via `ListTools` RPC
3. Tool execution with argument passing and result retrieval
4. Timeout and error handling for all MCP operations
5. Audit logging of all tool calls
6. Security sandbox for external tool execution

## Decision

### SDK Selection: mark3labs/mcp-go

We select `github.com/mark3labs/mcp-go` as the MCP Go SDK. This is the most
widely adopted Go MCP implementation with active maintenance, supporting:

- Client and server implementations
- stdio and SSE transport
- JSON-RPC-based message protocol
- Tool/Resource/Prompt primitives

Alternative considered: `metoro-ai/mcp-golang`. This SDK was rejected because:
- Smaller community and less test coverage
- Tighter coupling to specific transport patterns
- Less flexibility for custom transport configuration

### Integration Strategy

The MCP SDK is used **only** in the `logic-grpc-service` (backend service layer).
It is not exposed to the HTTP gateway or frontend directly.

The `MCPService` gRPC service wraps the MCP Go SDK:

```
HTTP Gateway (web-gin-service)
    | gRPC
MCPService (logic-grpc-service)
    | mark3labs/mcp-go
MCP Servers (stdio/SSE/HTTP)
```

### Security Sandbox

1. **Connection Timeout**: All MCP client connections have a configurable timeout
   (default 30s). TestMCPConnection uses a shorter 10s timeout.

2. **Default Disabled**: New MCP servers are created with `is_enabled=false`.
   An admin must explicitly enable them after verification.

3. **Audit Logging**: Every tool call is logged to `mcp_tool_logs` with:
   - Tool name, arguments (desensitized), result (truncated/desensitized)
   - Duration, error (if any), caller HR ID, session ID

4. **Parameter Desensitization**: Tool arguments and results are desensitized
   before logging to avoid storing sensitive data (phone numbers, email, etc.).

### Coexistence with Existing Tools

Hardcoded tools (in `ai/hr_adk_tools.go` and `ai/candidate_adk_tools.go`) remain
unchanged. MCP tools are injected alongside them at the Agent level:

```
Agent Tool List = hardcoded tools + enabled MCP tools
```

This is done by reading the `agent_tool_bindings` table (which already supports
tool name binding) and adding MCP tools to the resolved tool set at runtime.

## Consequences

### Positive

- Standardized protocol for tool integration
- Dynamic tool discovery without code changes
- Audit trail for all tool executions
- Existing hardcoded tools are not affected

### Negative

- Additional dependency (mark3labs/mcp-go) in go.mod
- MCP server management adds operational complexity
- Tool calls through MCP have latency overhead vs. direct Go function calls

### Mitigations

- Timeouts prevent hanging on unresponsive MCP servers
- Default-disabled policy prevents accidental exposure
- Audit logs enable forensic analysis of tool usage
- Desensitization prevents sensitive data leakage in logs

## References

- MCP Specification: https://modelcontextprotocol.io/
- mark3labs/mcp-go: https://github.com/mark3labs/mcp-go
