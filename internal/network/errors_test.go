package network

import (
	"errors"
	"net"
	"syscall"
	"testing"
)

func TestIsTemporaryError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "timeout error",
			err:      &timeoutError{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTemporaryError(tt.err)
			if result != tt.expected {
				t.Errorf("IsTemporaryError() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "timeout error",
			err:      &timeoutError{},
			expected: true,
		},
		{
			name:     "ErrTimeout",
			err:      ErrTimeout,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTimeoutError(tt.err)
			if result != tt.expected {
				t.Errorf("IsTimeoutError() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		operation string
		expected  string
	}{
		{
			name:      "nil error",
			err:       nil,
			operation: "test",
			expected:  "",
		},
		{
			name:      "regular error",
			err:       errors.New("original error"),
			operation: "send packet",
			expected:  "send packet: original error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapError(tt.err, tt.operation)
			if result == nil && tt.expected != "" {
				t.Errorf("WrapError() = nil, expected error")
				return
			}
			if result != nil && result.Error() != tt.expected {
				t.Errorf("WrapError() = %q, expected %q", result.Error(), tt.expected)
			}
		})
	}
}

func TestWrapErrorf(t *testing.T) {
	err := errors.New("original error")
	result := WrapErrorf(err, "operation failed for peer %s", "peer1")

	expected := "operation failed for peer peer1: original error"
	if result.Error() != expected {
		t.Errorf("WrapErrorf() = %q, expected %q", result.Error(), expected)
	}

	// Test with nil error
	result = WrapErrorf(nil, "test")
	if result != nil {
		t.Errorf("WrapErrorf(nil) = %v, expected nil", result)
	}
}

func TestRetryableError(t *testing.T) {
	originalErr := errors.New("temporary failure")
	retryErr := NewRetryableError(originalErr, 3)

	if retryErr.Retries != 0 {
		t.Errorf("Initial retries = %d, expected 0", retryErr.Retries)
	}

	if retryErr.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, expected 3", retryErr.MaxRetries)
	}

	expectedMsg := "retryable error (attempt 0/3): temporary failure"
	if retryErr.Error() != expectedMsg {
		t.Errorf("Error() = %q, expected %q", retryErr.Error(), expectedMsg)
	}

	// Test unwrap
	if errors.Unwrap(retryErr) != originalErr {
		t.Error("Unwrap() did not return original error")
	}
}

func TestRetryableError_ShouldRetry(t *testing.T) {
	tempErr := &timeoutError{}
	permErr := errors.New("permanent error")

	tests := []struct {
		name     string
		err      error
		retries  int
		max      int
		expected bool
	}{
		{
			name:     "temporary error within limit",
			err:      tempErr,
			retries:  0,
			max:      3,
			expected: true,
		},
		{
			name:     "temporary error at limit",
			err:      tempErr,
			retries:  3,
			max:      3,
			expected: false,
		},
		{
			name:     "permanent error",
			err:      permErr,
			retries:  0,
			max:      3,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryErr := &RetryableError{
				Err:        tt.err,
				Retries:    tt.retries,
				MaxRetries: tt.max,
			}

			result := retryErr.ShouldRetry()
			if result != tt.expected {
				t.Errorf("ShouldRetry() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestErrorStats(t *testing.T) {
	stats := &ErrorStats{}

	// Record various errors
	stats.RecordError(nil) // Should not increment
	stats.RecordError(&timeoutError{})
	stats.RecordError(&timeoutError{})
	stats.RecordError(errors.New("permanent error"))

	if stats.TotalErrors != 3 {
		t.Errorf("TotalErrors = %d, expected 3", stats.TotalErrors)
	}

	if stats.TimeoutErrors != 2 {
		t.Errorf("TimeoutErrors = %d, expected 2", stats.TimeoutErrors)
	}

	if stats.PermanentErrors != 1 {
		t.Errorf("PermanentErrors = %d, expected 1", stats.PermanentErrors)
	}
}

func TestErrorStats_GetErrorRate(t *testing.T) {
	stats := &ErrorStats{
		TotalErrors: 10,
	}

	tests := []struct {
		name       string
		totalOps   uint64
		expected   float64
	}{
		{
			name:     "zero operations",
			totalOps: 0,
			expected: 0.0,
		},
		{
			name:     "100 operations",
			totalOps: 100,
			expected: 0.1,
		},
		{
			name:     "50 operations",
			totalOps: 50,
			expected: 0.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stats.GetErrorRate(tt.totalOps)
			if result != tt.expected {
				t.Errorf("GetErrorRate() = %f, expected %f", result, tt.expected)
			}
		})
	}
}

// Helper types for testing

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

func TestIsConnectionReset(t *testing.T) {
	// Create a connection reset error
	opErr := &net.OpError{
		Op:  "read",
		Net: "tcp",
		Err: syscall.ECONNRESET,
	}

	if !IsConnectionReset(opErr) {
		t.Error("IsConnectionReset() = false for ECONNRESET, expected true")
	}

	// Test with non-reset error
	regularErr := errors.New("regular error")
	if IsConnectionReset(regularErr) {
		t.Error("IsConnectionReset() = true for regular error, expected false")
	}

	// Test with nil
	if IsConnectionReset(nil) {
		t.Error("IsConnectionReset() = true for nil, expected false")
	}
}

func BenchmarkIsTemporaryError(b *testing.B) {
	err := &timeoutError{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsTemporaryError(err)
	}
}

func BenchmarkWrapError(b *testing.B) {
	err := errors.New("test error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WrapError(err, "operation")
	}
}

func BenchmarkErrorStats_RecordError(b *testing.B) {
	stats := &ErrorStats{}
	err := &timeoutError{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats.RecordError(err)
	}
}
