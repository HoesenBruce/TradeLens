// Package providererrors keeps untrusted provider diagnostics out of responses, status rows and logs.
package providererrors

import (
	"context"
	"errors"
	"fmt"
	"net"
)

type safeError struct {
	cause   error
	message string
}

func (e *safeError) Error() string { return e.message }
func (e *safeError) Unwrap() error { return e.cause }

// Debug formatting must not reflect the retained provider cause.
func (e *safeError) Format(f fmt.State, verb rune) { _, _ = f.Write([]byte(e.message)) }

type statusError int

func (e statusError) Error() string { return fmt.Sprintf("external service returned HTTP %d", e) }

func HTTPStatus(code int) error { return statusError(code) }

// Safe preserves error identity for retries/cancellation without formatting provider text or URLs.
func Safe(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*safeError); ok {
		return err
	}
	message := "external service request failed"
	if status, ok := errors.AsType[statusError](err); ok {
		message = status.Error()
	} else if errors.Is(err, context.Canceled) {
		message = "external service request canceled"
	} else if timeout, ok := errors.AsType[net.Error](err); errors.Is(err, context.DeadlineExceeded) || ok && timeout.Timeout() {
		message = "external service request timed out"
	}
	return &safeError{cause: err, message: message}
}
