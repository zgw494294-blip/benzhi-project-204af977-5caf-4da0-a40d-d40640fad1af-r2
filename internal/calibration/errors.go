package calibration

import "fmt"

type Error struct {
	Code    string
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func invalid(code, message string) error {
	return Error{Code: code, Message: message}
}

func notFound(entity string) error {
	return Error{Code: "not_found", Message: entity + "不存在"}
}
