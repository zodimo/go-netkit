package netkit

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewCloseError(t *testing.T) {
	code := 1000
	reason := "normal closure"

	err := NewCloseError(code, reason)

	// Check that the error implements CloseError
	if _, ok := err.(CloseError); !ok {
		t.Error("NewCloseError did not return a CloseError")
	}

	// Check that the error has the correct code and reason
	if err.Code() != code {
		t.Errorf("CloseError has wrong code: got %d, want %d", err.Code(), code)
	}
	if err.Reason() != reason {
		t.Errorf("CloseError has wrong reason: got %s, want %s", err.Reason(), reason)
	}

	// Check the error message
	expected := fmt.Sprintf("close error: %d, %s", code, reason)
	if err.Error() != expected {
		t.Errorf("CloseError has wrong error message: got %s, want %s", err.Error(), expected)
	}
}

func TestAsCloseError(t *testing.T) {
	// Test with a CloseError
	closeErr := NewCloseError(1000, "normal closure")
	result := AsCloseError(closeErr)
	if result == nil {
		t.Error("AsCloseError returned nil for a CloseError")
	}
	if result.Code() != 1000 || result.Reason() != "normal closure" {
		t.Errorf("AsCloseError returned wrong error: got %d/%s, want 1000/normal closure",
			result.Code(), result.Reason())
	}

	// Test with a non-CloseError
	regularErr := errors.New("regular error")
	result = AsCloseError(regularErr)
	if result != nil {
		t.Errorf("AsCloseError did not return nil for a non-CloseError: got %v", result)
	}

	// Test with nil
	result = AsCloseError(nil)
	if result != nil {
		t.Errorf("AsCloseError did not return nil for nil: got %v", result)
	}
}

func TestIsCloseError(t *testing.T) {
	// Test with a CloseError
	closeErr := NewCloseError(1000, "normal closure")
	if !IsCloseError(closeErr) {
		t.Error("IsCloseError returned false for a CloseError")
	}

	// Test with a non-CloseError
	regularErr := errors.New("regular error")
	if IsCloseError(regularErr) {
		t.Error("IsCloseError returned true for a non-CloseError")
	}

	// Test with nil
	if IsCloseError(nil) {
		t.Error("IsCloseError returned true for nil")
	}
}

func TestIsUnexpectedCloseError(t *testing.T) {
	// Create close errors with different codes
	normalErr := NewCloseError(1000, "normal closure")
	goingAwayErr := NewCloseError(1001, "going away")
	protocolErr := NewCloseError(1002, "protocol error")

	// Test with expected codes
	if IsUnexpectedCloseError(normalErr, 1000, 1001) {
		t.Error("IsUnexpectedCloseError returned true for an expected code 1000")
	}
	if IsUnexpectedCloseError(goingAwayErr, 1000, 1001) {
		t.Error("IsUnexpectedCloseError returned true for an expected code 1001")
	}

	// Test with unexpected code
	if !IsUnexpectedCloseError(protocolErr, 1000, 1001) {
		t.Error("IsUnexpectedCloseError returned false for an unexpected code 1002")
	}

	// Test with non-CloseError
	regularErr := errors.New("regular error")
	if IsUnexpectedCloseError(regularErr, 1000, 1001) {
		t.Error("IsUnexpectedCloseError returned true for a non-CloseError")
	}

	// Test with nil
	if IsUnexpectedCloseError(nil, 1000, 1001) {
		t.Error("IsUnexpectedCloseError returned true for nil")
	}

	// Test with no expected codes
	if !IsUnexpectedCloseError(normalErr) {
		t.Error("IsUnexpectedCloseError returned false for any code when no expected codes were provided")
	}
}

func TestCloseErrorImplementsError(t *testing.T) {
	var err error = NewCloseError(1000, "normal closure")

	// Check that the error can be used as an error
	if err.Error() == "" {
		t.Error("CloseError does not implement Error() correctly")
	}
}

func TestCloseErrorNilCheck(t *testing.T) {
	var err error

	// Test AsCloseError with nil
	if AsCloseError(err) != nil {
		t.Error("AsCloseError did not return nil for nil error")
	}

	// Test IsCloseError with nil
	if IsCloseError(err) {
		t.Error("IsCloseError returned true for nil error")
	}

	// Test IsUnexpectedCloseError with nil
	if IsUnexpectedCloseError(err, 1000) {
		t.Error("IsUnexpectedCloseError returned true for nil error")
	}
}
