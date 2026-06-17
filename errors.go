package bulutklinik

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for use with errors.Is. Every API failure also matches ErrAPI;
// transport failures match ErrTransport.
var (
	ErrTransport      = errors.New("bulutklinik: transport error")
	ErrAPI            = errors.New("bulutklinik: api error")
	ErrValidation     = errors.New("bulutklinik: validation error")
	ErrAuthentication = errors.New("bulutklinik: authentication error")
	ErrAuthorization  = errors.New("bulutklinik: authorization error")
	ErrNotFound       = errors.New("bulutklinik: not found")
	ErrRateLimit      = errors.New("bulutklinik: rate limited")
)

// TransportError is returned when no usable HTTP response was received
// (network failure, timeout, DNS or TLS error). It matches [ErrTransport].
type TransportError struct {
	Message string
	Err     error
}

func (e *TransportError) Error() string        { return e.Message }
func (e *TransportError) Unwrap() error        { return e.Err }
func (e *TransportError) Is(target error) bool { return target == ErrTransport }

// APIError is returned when an HTTP response was received but the call was not
// successful. Use errors.As to read its fields, or errors.Is against a sentinel
// (it matches both [ErrAPI] and its specific kind, e.g. [ErrNotFound]).
type APIError struct {
	Message    string
	HTTPStatus int
	ResultType *int
	// ErrorType is the raw envelope value: a string label or a numeric code.
	ErrorType  any
	Data       json.RawMessage
	Method     string
	Path       string
	RetryAfter *int

	kind error
}

func (e *APIError) Error() string {
	return fmt.Sprintf("bulutklinik: %s %s: %s (HTTP %d)", e.Method, e.Path, e.Message, e.HTTPStatus)
}

func (e *APIError) Is(target error) bool { return target == ErrAPI || target == e.kind }

func newAPIError(method, path, message string, status int, resultType *int, errorType any, data json.RawMessage, retryAfter *int) *APIError {
	kind := ErrAPI
	switch {
	case resultType != nil && *resultType == 2:
		kind = ErrAuthentication
	case isValidationErrorType(errorType) || status == 422:
		kind = ErrValidation
	case status == 401:
		kind = ErrAuthentication
	case status == 403:
		kind = ErrAuthorization
	case status == 404:
		kind = ErrNotFound
	case status == 429:
		kind = ErrRateLimit
	}
	return &APIError{
		Message:    message,
		HTTPStatus: status,
		ResultType: resultType,
		ErrorType:  errorType,
		Data:       data,
		Method:     method,
		Path:       path,
		RetryAfter: retryAfter,
		kind:       kind,
	}
}

func isValidationErrorType(errorType any) bool {
	s, ok := errorType.(string)
	return ok && strings.EqualFold(s, "validation")
}
