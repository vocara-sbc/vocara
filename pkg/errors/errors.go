// Package errors provides business error code encapsulation.
// It defines a custom business error type containing an error code, error message, and HTTP status code.
package errors

import (
	"errors"
	"fmt"
)

// BizError is a business error.
type BizError struct {
	Code       int    `json:"code"`              // Error code
	Message    string `json:"message"`           // Error message
	Detail     string `json:"detail,omitempty"`  // Detailed information
	HTTPStatus int    `json:"-"`                 // HTTP status code
	cause      error  // Original error
}

// Error implements the error interface.
func (e *BizError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Detail)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap supports errors.Unwrap.
func (e *BizError) Unwrap() error {
	return e.cause
}

// WithDetail sets detailed information and returns the receiver.
func (e *BizError) WithDetail(detail string) *BizError {
	e.Detail = detail
	return e
}

// WithDetailf sets formatted detailed information and returns the receiver.
func (e *BizError) WithDetailf(format string, args ...interface{}) *BizError {
	e.Detail = fmt.Sprintf(format, args...)
	return e
}

// WithCause sets the original error and returns the receiver.
func (e *BizError) WithCause(err error) *BizError {
	e.cause = err
	if e.Detail == "" && err != nil {
		e.Detail = err.Error()
	}
	return e
}

// NewBizError creates a business error.
func NewBizError(code int, msg string) *BizError {
	return &BizError{
		Code:       code,
		Message:    msg,
		HTTPStatus: codeToHTTPStatus(code),
	}
}

// NewBizErrorWithStatus creates a business error with an HTTP status code.
func NewBizErrorWithStatus(code int, msg string, httpStatus int) *BizError {
	return &BizError{
		Code:       code,
		Message:    msg,
		HTTPStatus: httpStatus,
	}
}

// Wrap wraps a regular error as a business system error.
func Wrap(err error) *BizError {
	if err == nil {
		return nil
	}
	// Return the error directly if it is already a BizError
	var bizErr *BizError
	if errors.As(err, &bizErr) {
		return bizErr
	}
	return &BizError{
		Code:       CodeSystemError,
		Message:    MsgSystemError,
		Detail:     err.Error(),
		HTTPStatus: 500,
		cause:      err,
	}
}

// WrapWithCode wraps a regular error as a business error with the specified error code.
func WrapWithCode(err error, code int, msg string) *BizError {
	if err == nil {
		return nil
	}
	return &BizError{
		Code:       code,
		Message:    msg,
		Detail:     err.Error(),
		HTTPStatus: codeToHTTPStatus(code),
		cause:      err,
	}
}

// IsBizError reports whether an error is a business error and returns it.
func IsBizError(err error) (*BizError, bool) {
	if err == nil {
		return nil, false
	}
	var bizErr *BizError
	if errors.As(err, &bizErr) {
		return bizErr, true
	}
	return nil, false
}

// IsCode reports whether the error code matches.
func (e *BizError) IsCode(code int) bool {
	return e.Code == code
}

// CodeToHTTPStatus infers an HTTP status code from an error code.
// Error code rules:
//   0        -> 200 OK
//   40000-40099 -> 400 Bad Request
//   40100-40199 -> 401 Unauthorized
//   40300-40399 -> 403 Forbidden
//   40400-40499 -> 404 Not Found
//   40900-40999 -> 409 Conflict
//   42900-42999 -> 429 Too Many Requests
//   50000-59999 -> 500 Internal Server Error
//   70001-70003 -> 402 Payment Required
//   70004-70007 -> 403 Forbidden
//   Others   -> 200 OK
func CodeToHTTPStatus(code int) int {
	return codeToHTTPStatus(code)
}

func codeToHTTPStatus(code int) int {
	switch {
	case code == 0:
		return 200
	case code >= 40000 && code < 40100:
		return 400
	case code >= 40100 && code < 40200:
		return 401
	case code >= 40300 && code < 40400:
		return 403
	case code >= 40400 && code < 40500:
		return 404
	case code >= 40900 && code < 41000:
		return 409
	case code >= 42900 && code < 43000:
		return 429
	case code >= 50000 && code < 60000:
		return 500
	case code >= 70001 && code < 70004:
		return 402
	case code >= 70004 && code < 70008:
		return 403
	default:
		return 200
	}
}
