package contextkeys

type CtxKey string

const (
	RequestID    CtxKey = "request_id"
	ClientIP     CtxKey = "client_ip"
	UserID       CtxKey = "auth_user_id"
	AccountType  CtxKey = "auth_account_type"
	TenantID     CtxKey = "auth_tenant_id"
	MembershipID CtxKey = "auth_membership_id"
	ClientApp    CtxKey = "auth_client_app"
	TraceID      CtxKey = "trace_id"
	SpanID       CtxKey = "span_id"
	Traceparent  CtxKey = "traceparent"
)
