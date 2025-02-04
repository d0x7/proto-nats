package protonats

import (
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
	"github.com/pkg/errors"
)

// Predefined Errors
var (
	ErrMarshallingFailed   = errors.New("Failed to marshal proto message")
	ErrUnmarshallingFailed = errors.New("Failed to unmarshal proto message")
)

// ServiceError is returned when the server returns an error instead of the expected message.
type ServiceError struct {
	Code, Description, Details string
}

func (e ServiceError) Error() string {
	if e.Details == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Description)
	}
	return fmt.Sprintf("%s: %s (%s)", e.Code, e.Description, e.Details)
}

func (e ServiceError) Is(target error) bool {
	var se ServiceError
	ok := errors.As(target, &se)
	return ok
}

func IsServiceError(err error) bool {
	var se ServiceError
	ok := errors.As(err, &se)
	return ok
}

func AsServiceError(err error) (ServiceError, bool) {
	var se ServiceError
	ok := errors.As(err, &se)
	return se, ok
}

// ServerError is returned from functions when the server implementation returns an ServerError.
// It contains a code, description, and an optional wrapped error.
type ServerError struct {
	Code, Description string
	Wrapped           error
	Headers           map[string][]string
}

func (n ServerError) Error() string {
	if n.Wrapped != nil {
		return n.Description + ": " + n.Wrapped.Error()
	}
	return n.Description
}

func (n ServerError) Cause() error {
	return n.Wrapped
}

// GetWrapped returns the wrapped error as a byte slice, or nil if there is no wrapped error.
// It's therefore safe to be used directly in a NATS response, for example, like this:
// ```request.Error(natsErr.code, natsErr.description, natsErr.GetWrapped())```
func (n ServerError) GetWrapped() []byte {
	if n.Wrapped != nil {
		return []byte(n.Wrapped.Error())
	}
	return nil
}

func (n ServerError) ensureHeader() {
	if n.Headers == nil {
		n.Headers = make(nats.Header)
	}
}

func (n ServerError) GetOptHeaders() micro.RespondOpt {
	if n.Headers == nil || len(n.Headers) == 0 {
		return func(m *nats.Msg) {}
	}

	return func(m *nats.Msg) {
		if m.Header == nil {
			m.Header = n.Headers
			return
		}

		for k, v := range n.Headers {
			m.Header[k] = v
		}
	}
}

func (n ServerError) AddHeader(header, value string) ServerError {
	n.ensureHeader()
	n.Headers[header] = append(n.Headers[header], value)
	return n
}

func (n ServerError) SetHeader(header, value string) ServerError {
	n.ensureHeader()
	n.Headers[header] = []string{value}
	return n
}

func (n ServerError) GetHeaders() micro.Headers {
	n.ensureHeader()
	return n.Headers
}

func (n ServerError) WithHeaders(headers map[string][]string) error {
	n.Headers = headers
	return n
}

// NewServerErr creates a new ServerError with the given code and description.
// To be used within the server implementation of your service, when an error occurs.
func NewServerErr(code, description string) ServerError {
	return WrapServerErr(nil, code, description)
}

// WrapServerErr wraps an existing error into a ServerError with the given code and description.
func WrapServerErr(err error, code, description string) ServerError {
	return ServerError{
		Code:        code,
		Description: description,
		Wrapped:     err,
	}
}
