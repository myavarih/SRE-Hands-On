package exceptions

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrNotFound       = errors.New("resource not found")
	ErrAlreadyExists  = errors.New("resource already exists")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrConflict       = errors.New("conflict / insufficient stock")
	ErrInternal       = errors.New("internal server error")
)

type DomainError struct {
	Code    codes.Code
	Message string
	Err     error
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

func NewNotFoundError(message string) error {
	return &DomainError{Code: codes.NotFound, Message: message, Err: ErrNotFound}
}

func NewAlreadyExistsError(message string) error {
	return &DomainError{Code: codes.AlreadyExists, Message: message, Err: ErrAlreadyExists}
}

func NewUnauthorizedError(message string) error {
	return &DomainError{Code: codes.Unauthenticated, Message: message, Err: ErrUnauthorized}
}

func NewForbiddenError(message string) error {
	return &DomainError{Code: codes.PermissionDenied, Message: message, Err: ErrForbidden}
}

func NewValidationError(message string) error {
	return &DomainError{Code: codes.InvalidArgument, Message: message, Err: ErrInvalidArgument}
}

func NewConflictError(message string) error {
	return &DomainError{Code: codes.FailedPrecondition, Message: message, Err: ErrConflict}
}

func NewInternalError(message string, err error) error {
	return &DomainError{Code: codes.Internal, Message: message, Err: err}
}

// ToGRPCError converts a domain error to a gRPC status error.
func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	var dErr *DomainError
	if errors.As(err, &dErr) {
		return status.Error(dErr.Code, dErr.Message)
	}

	return status.Error(codes.Internal, err.Error())
}

// FromGRPCError maps a gRPC error back into a standard domain error.
func FromGRPCError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	switch st.Code() {
	case codes.NotFound:
		return NewNotFoundError(st.Message())
	case codes.AlreadyExists:
		return NewAlreadyExistsError(st.Message())
	case codes.Unauthenticated:
		return NewUnauthorizedError(st.Message())
	case codes.PermissionDenied:
		return NewForbiddenError(st.Message())
	case codes.InvalidArgument:
		return NewValidationError(st.Message())
	case codes.FailedPrecondition, codes.Aborted:
		return NewConflictError(st.Message())
	default:
		return NewInternalError(st.Message(), err)
	}
}
