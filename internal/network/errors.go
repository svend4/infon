package network

import (
	"errors"
	"fmt"
	"net"
	"syscall"
)

var (
	// ErrTimeout indicates a network operation timed out
	ErrTimeout = errors.New("network operation timed out")
	// ErrConnectionReset indicates connection was reset by peer
	ErrConnectionReset = errors.New("connection reset by peer")
	// ErrNetworkUnreachable indicates network is unreachable
	ErrNetworkUnreachable = errors.New("network unreachable")
	// ErrTemporary indicates a temporary network error
	ErrTemporary = errors.New("temporary network error")
)

// IsTemporaryError checks if an error is temporary and can be retried
func IsTemporaryError(err error) bool {
	if err == nil {
		return false
	}

	// Check for net.Error with Temporary() method
	if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
		return true
	}

	// Check for specific errno values
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		var sysErr syscall.Errno
		if errors.As(opErr.Err, &sysErr) {
			switch sysErr {
			case syscall.ECONNREFUSED, syscall.ETIMEDOUT,
				syscall.EHOSTUNREACH, syscall.ENETUNREACH:
				return true
			}
		}
	}

	return false
}

// IsTimeoutError checks if an error is a timeout error
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	return errors.Is(err, ErrTimeout)
}

// IsConnectionReset checks if connection was reset
func IsConnectionReset(err error) bool {
	if err == nil {
		return false
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		var sysErr syscall.Errno
		if errors.As(opErr.Err, &sysErr) {
			return sysErr == syscall.ECONNRESET
		}
	}

	return errors.Is(err, ErrConnectionReset)
}

// WrapError wraps an error with additional context
func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", operation, err)
}

// WrapErrorf wraps an error with formatted context
func WrapErrorf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}

	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s: %w", msg, err)
}

// RetryableError indicates an operation can be retried
type RetryableError struct {
	Err     error
	Retries int
	MaxRetries int
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable error (attempt %d/%d): %v", e.Retries, e.MaxRetries, e.Err)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// ShouldRetry checks if the error should be retried
func (e *RetryableError) ShouldRetry() bool {
	return e.Retries < e.MaxRetries && IsTemporaryError(e.Err)
}

// NewRetryableError creates a new retryable error
func NewRetryableError(err error, maxRetries int) *RetryableError {
	return &RetryableError{
		Err:        err,
		Retries:    0,
		MaxRetries: maxRetries,
	}
}

// ErrorStats tracks error statistics
type ErrorStats struct {
	TotalErrors    uint64
	TimeoutErrors  uint64
	ResetErrors    uint64
	TempErrors     uint64
	PermanentErrors uint64
}

// RecordError records an error in statistics
func (s *ErrorStats) RecordError(err error) {
	if err == nil {
		return
	}

	s.TotalErrors++

	if IsTimeoutError(err) {
		s.TimeoutErrors++
	} else if IsConnectionReset(err) {
		s.ResetErrors++
	} else if IsTemporaryError(err) {
		s.TempErrors++
	} else {
		s.PermanentErrors++
	}
}

// GetErrorRate returns the error rate as a fraction (0.0 to 1.0)
func (s *ErrorStats) GetErrorRate(totalOperations uint64) float64 {
	if totalOperations == 0 {
		return 0.0
	}
	return float64(s.TotalErrors) / float64(totalOperations)
}
