package noteapp

import "fmt"

type ErrorKind string

const (
	ErrorInvalid  ErrorKind = "VALIDATION_ERROR"
	ErrorNotFound ErrorKind = "NOT_FOUND"
	ErrorInternal ErrorKind = "INTERNAL"
)

type UseCaseError struct {
	Kind    ErrorKind
	Message string
	Cause   error
}

func (err *UseCaseError) Error() string {
	if err.Cause == nil {
		return err.Message
	}
	return fmt.Sprintf("%s: %v", err.Message, err.Cause)
}

func (err *UseCaseError) Unwrap() error {
	return err.Cause
}

func invalid(message string) error {
	return &UseCaseError{Kind: ErrorInvalid, Message: message}
}

func notFound(message string) error {
	return &UseCaseError{Kind: ErrorNotFound, Message: message}
}

func internal(message string, cause error) error {
	return &UseCaseError{Kind: ErrorInternal, Message: message, Cause: cause}
}

type ConflictError struct {
	ServerUpdatedAt string `json:"server_updated_at"`
	ServerSnapshot  Note   `json:"server_snapshot"`
}

func (err *ConflictError) Error() string {
	return "note has been modified by another client"
}
