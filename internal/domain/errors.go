package domain

import "fmt"

type RuleError struct {
	Code    string
	Message string
}

func (e RuleError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func Invalid(code, message string) error {
	return RuleError{Code: code, Message: message}
}
