package apperrors

import "errors"

type Kind string

const (
	KindBadRequest   Kind = "bad_request"
	KindUnauthorized Kind = "unauthorized"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
	KindInternal     Kind = "internal"
)

type Code string

const (
	CodeInvalidRequest     Code = "invalid_request"
	CodeInvalidCredentials Code = "invalid_credentials"
	CodeInitiativeNotFound Code = "initiative_not_found"
	CodeEmailAlreadyExists Code = "email_already_exists"
	CodeInternalError      Code = "internal_error"
)

type Error struct {
	Kind Kind
	Code Code
	Err  error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return string(e.Code)
	}

	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}

func BadRequest(code Code, err error) *Error {
	return &Error{Kind: KindBadRequest, Code: code, Err: err}
}

func Unauthorized(code Code, err error) *Error {
	return &Error{Kind: KindUnauthorized, Code: code, Err: err}
}

func NotFound(code Code, err error) *Error {
	return &Error{Kind: KindNotFound, Code: code, Err: err}
}

func Conflict(code Code, err error) *Error {
	return &Error{Kind: KindConflict, Code: code, Err: err}
}

func Internal(err error) *Error {
	return &Error{Kind: KindInternal, Code: CodeInternalError, Err: err}
}

func From(err error) (*Error, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}

	return nil, false
}
