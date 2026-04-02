package errors

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	CodeInternalError      = "INTERNAL_ERROR"
	CodeAlreadyExists      = "ALREADY_EXISTS"
	CodeNotFound           = "NOT_FOUND"
	CodeBadRequest         = "BAD_REQUEST"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeForbidden          = "FORBIDDEN"
	CodeInvalidInput       = "INVALID_INPUT"
	CodeTimeout            = "TIMEOUT"
	CodeConflict           = "CONFLICT"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	CodeRateLimitExceeded  = "RATE_LIMIT_EXCEEDED"
)

type AppError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	HTTPStatus int16                  `json:"-"`
	GRPCStatus codes.Code             `json:"-"`
}

// Assertion error interface
func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Avoid race logic
func (e *AppError) Clone() *AppError {
	c := *e
	if e.Details != nil {
		c.Details = make(map[string]interface{}, len(e.Details))
		for k, v := range e.Details {
			c.Details[k] = v
		}
	}
	return &c
}

// Custom error constructor
func NewAppError(code, message string, details map[string]interface{}, httpStatus int16, gRPCStatus codes.Code) *AppError {
	if details == nil {
		details = make(map[string]interface{})
	}

	return &AppError{
		Code:       code,
		Message:    message,
		Details:    details,
		HTTPStatus: httpStatus,
		GRPCStatus: gRPCStatus,
	}
}

func WrapError(err error, code, message string) *AppError {
	appErr, ok := err.(*AppError)
	if ok {
		return appErr.Clone().WithMessage(message)
	}

	return NewAppError(code, message, map[string]interface{}{"original_error": err}, 500, codes.Internal)
}

func (e *AppError) WithMessage(message string) *AppError {
	e.Message = message
	return e
}

func (e *AppError) WithCode(code string) *AppError {
	e.Code = code
	return e
}

func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	if details == nil {
		details = make(map[string]interface{})
	}
	e.Details = details
	return e
}

func (e *AppError) ToGRPCError() error {
	return status.Error(e.GRPCStatus, e.Message)
}

func FromGRPCError(err error) *AppError {
	st, ok := status.FromError(err)
	if !ok {
		return ErrInternalError.Clone()
	}
	switch st.Code() {
	case codes.AlreadyExists:
		return ErrAlreadyExists.Clone().WithMessage(st.Message())
	case codes.NotFound:
		return ErrNotFound.Clone().WithMessage(st.Message())
	case codes.InvalidArgument:
		return ErrBadRequest.Clone().WithMessage(st.Message())
	case codes.Unauthenticated:
		return ErrUnauthorized.Clone().WithMessage(st.Message())
	case codes.PermissionDenied:
		return ErrForbidden.Clone().WithMessage(st.Message())
	case codes.DeadlineExceeded:
		return ErrTimeout.Clone().WithMessage(st.Message())
	case codes.Unavailable:
		return ErrServiceUnavailable.Clone().WithMessage(st.Message())
	case codes.ResourceExhausted:
		return ErrRateLimitExceeded.Clone().WithMessage(st.Message())
	case codes.Aborted:
		return ErrConflict.Clone().WithMessage(st.Message())
	default:
		return ErrInternalError.Clone().WithMessage(st.Message())
	}
}

/* CONSTANT ERROR MODELS */
var (
	ErrInternalError      = NewAppError(CodeInternalError, "Internal server error", nil, 500, codes.Internal)
	ErrAlreadyExists      = NewAppError(CodeAlreadyExists, "Resource already exists", nil, 409, codes.AlreadyExists)
	ErrNotFound           = NewAppError(CodeNotFound, "Resource not found", nil, 404, codes.NotFound)
	ErrBadRequest         = NewAppError(CodeBadRequest, "Bad request", nil, 400, codes.InvalidArgument)
	ErrUnauthorized       = NewAppError(CodeUnauthorized, "Unauthorized", nil, 401, codes.Unauthenticated)
	ErrForbidden          = NewAppError(CodeForbidden, "Forbidden", nil, 403, codes.PermissionDenied)
	ErrInvalidInput       = NewAppError(CodeInvalidInput, "Invalid input", nil, 422, codes.InvalidArgument)
	ErrTimeout            = NewAppError(CodeTimeout, "Request timeout", nil, 504, codes.DeadlineExceeded)
	ErrConflict           = NewAppError(CodeConflict, "Conflict", nil, 409, codes.Aborted)
	ErrServiceUnavailable = NewAppError(CodeServiceUnavailable, "Service unavailable", nil, 503, codes.Unavailable)
	ErrRateLimitExceeded  = NewAppError(CodeRateLimitExceeded, "Rate limit exceeded", nil, 429, codes.ResourceExhausted)
)
