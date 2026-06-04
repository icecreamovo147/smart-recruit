package errs

import "errors"

const (
	OK                   = 0
	ErrBadRequest        = 400
	ErrUnauthorized      = 401
	ErrForbidden         = 403
	ErrConflict          = 409
	ErrInternal          = 500
	ErrProfileIncomplete = 4001
	ErrResumeNotFound    = 4002
	ErrDuplicateApply    = 4003
	ErrJobNotAvailable   = 4004
	ErrDuplicateResume   = 4005
	ErrNoPermission      = 4030
)

// Sentinel errors for typed error classification.
// Service-layer code wraps domain errors with these so the HTTP gateway
// can classify them via errors.Is instead of fragile string matching.
var (
	ErrAIService               = errors.New("ai_service_error")
	ErrOSSNotFound             = errors.New("oss_not_found")
	ErrBackendUnavailable      = errors.New("backend_unavailable")
	ErrTimeout                 = errors.New("operation_timeout")
	ErrDuplicateResumeSentinel = errors.New("duplicate_valid_resume")
)
