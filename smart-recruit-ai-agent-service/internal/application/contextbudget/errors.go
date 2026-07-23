package contextbudget

import "errors"

type GuardError struct{ Code string }

func (e *GuardError) Error() string { return e.Code }

func ErrorCode(err error) string {
	var target *GuardError
	if errors.As(err, &target) {
		return target.Code
	}
	return ""
}
