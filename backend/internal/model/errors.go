package model

import "fmt"

// API 业务错误码。HTTP 层映射见 api.WriteError。
const (
	CodeOK              = 0
	CodeBadRequest      = 40000
	CodeValidation      = 40001
	CodeDimMismatch     = 40002
	CodeFilterInvalid   = 40003
	CodeUploadInvalid   = 40004
	CodeBudgetExceeded  = 40005
	CodeUnauthorized    = 40100
	CodeNotFound        = 40400
	CodeConflict        = 40900
	CodeUnimplemented   = 40006
	CodeInternal        = 50000
	CodeProvider        = 50200
)

type Error struct {
	Code    int
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Cause }

func NewError(code int, msg string) *Error {
	return &Error{Code: code, Message: msg}
}

func Wrap(code int, msg string, err error) *Error {
	return &Error{Code: code, Message: msg, Cause: err}
}

func IsCode(err error, code int) bool {
	if err == nil {
		return false
	}
	e, ok := err.(*Error)
	return ok && e.Code == code
}
