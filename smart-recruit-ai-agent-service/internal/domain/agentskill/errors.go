package agentskill

import (
	"errors"
	"fmt"
)

const (
	CodePackageInvalid          = "AGENT_SKILL_PACKAGE_INVALID"
	CodeCoreBudgetExceeded      = "AGENT_SKILL_CORE_BUDGET_EXCEEDED"
	CodeSectionInvalid          = "AGENT_SKILL_SECTION_INVALID"
	CodeCompositionConflict     = "AGENT_SKILL_COMPOSITION_CONFLICT"
	CodeSelectionOutsideRelease = "AGENT_SKILL_SELECTION_OUTSIDE_RELEASE"
	CodeConfirmationRequired    = "AGENT_SKILL_CONFIRMATION_REQUIRED"
	CodeConfirmationExpired     = "AGENT_SKILL_CONFIRMATION_EXPIRED"
	CodeV2Disabled              = "AGENT_SKILL_V2_DISABLED"
	CodeOutputInvalid           = "AGENT_SKILL_OUTPUT_INVALID"
)

type CompileError struct {
	Code    string
	Field   string
	Message string
}

func (e *CompileError) Error() string {
	if e.Field == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s: %s", e.Code, e.Field, e.Message)
}

func ErrorCode(err error) string {
	var target *CompileError
	if errors.As(err, &target) {
		return target.Code
	}
	return ""
}

func compileError(code, field, message string) error {
	return &CompileError{Code: code, Field: field, Message: message}
}
