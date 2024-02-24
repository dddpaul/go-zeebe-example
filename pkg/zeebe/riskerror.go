package zeebe

import (
	"fmt"
)

// ZeebeError represents a constant error type with a code and a message.
type ZeebeError struct {
	Code    string
	Message string
}

func (z ZeebeError) Error() string {
	return fmt.Sprintf("%s: %s", z.Code, z.Message)
}

// Predefined errors
var (
	RiskLevelError = ZeebeError{
		Code:    "RISK_LEVEL_ERROR",
		Message: "Some error just happened: %s",
	}
)

// ZeebeBpmnError represents an error that can be thrown within a Zeebe BPMN process.
type ZeebeBpmnError struct {
	Code    string
	Message string
	// In Go, we generally don't use empty interfaces, but this is here to mimic the Java usage.
	Variables map[string]interface{}
}

func (z ZeebeBpmnError) Error() string {
	return fmt.Sprintf("%s: %s", z.Code, z.Message)
}

// NewZeebeBpmnError constructs a new ZeebeBpmnError
func NewZeebeBpmnError(z ZeebeError, variables map[string]interface{}) ZeebeBpmnError {
	return ZeebeBpmnError{
		Code:      z.Code,
		Message:   z.Message,
		Variables: variables,
	}
}
