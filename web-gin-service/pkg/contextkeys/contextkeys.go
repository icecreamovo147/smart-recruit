package contextkeys

type CtxKey string

const (
	RequestID   CtxKey = "request_id"
	ClientIP    CtxKey = "client_ip"
	UserID      CtxKey = "auth_user_id"
	AccountType CtxKey = "auth_account_type"
	TraceID     CtxKey = "trace_id"
	SpanID      CtxKey = "span_id"
	Traceparent CtxKey = "traceparent"
)
