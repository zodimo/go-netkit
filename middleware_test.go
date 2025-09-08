package netkit

import (
	"errors"
	"testing"
	"time"

	"github.com/zodimo/go-netkit/cbio"
)

// TestMiddleware tests the middleware functionality
func TestMiddleware(t *testing.T) {
	// Create a recording handler to check the final results
	finalHandler := newRecordingTransportHandler()

	// Create middleware that counts calls
	var (
		openCount    int
		messageCount int
		closeCount   int
		errorCount   int
	)

	middleware := func(handler TransportHandler) TransportHandler {
		return NewTransportHandler(
			func(conn cbio.WriteCloser) {
				openCount++
				handler.OnOpen(conn)
			},
			func(message []byte) {
				messageCount++
				handler.OnMessage(message)
			},
			func(code int, reason string) {
				closeCount++
				handler.OnClose(code, reason)
			},
			func(err error) {
				errorCount++
				handler.OnError(err)
			},
		)
	}

	// Apply middleware to the handler
	handlerWithMiddleware := middleware(finalHandler)

	// Test the middleware
	conn := newMockCbioWriteCloser()
	message := []byte("test message")
	code := 1000
	reason := "normal closure"
	err := ErrTest

	// Call the handler methods
	handlerWithMiddleware.OnOpen(conn)
	handlerWithMiddleware.OnMessage(message)
	handlerWithMiddleware.OnClose(code, reason)
	handlerWithMiddleware.OnError(err)

	// Check that the middleware counted correctly
	if openCount != 1 {
		t.Errorf("Middleware open count is %d, want 1", openCount)
	}
	if messageCount != 1 {
		t.Errorf("Middleware message count is %d, want 1", messageCount)
	}
	if closeCount != 1 {
		t.Errorf("Middleware close count is %d, want 1", closeCount)
	}
	if errorCount != 1 {
		t.Errorf("Middleware error count is %d, want 1", errorCount)
	}

	// Check that the final handler was called with the correct values
	if !finalHandler.WasOnOpenCalled() {
		t.Error("Final handler OnOpen was not called")
	}
	if finalHandler.GetConn() != conn {
		t.Error("Final handler received wrong connection")
	}

	if !finalHandler.WasOnMessageCalled() {
		t.Error("Final handler OnMessage was not called")
	}
	if string(finalHandler.GetMessage()) != string(message) {
		t.Errorf("Final handler received wrong message: got %q, want %q",
			finalHandler.GetMessage(), message)
	}

	if !finalHandler.WasOnCloseCalled() {
		t.Error("Final handler OnClose was not called")
	}
	if finalHandler.GetCloseCode() != code {
		t.Errorf("Final handler received wrong close code: got %d, want %d",
			finalHandler.GetCloseCode(), code)
	}
	if finalHandler.GetCloseReason() != reason {
		t.Errorf("Final handler received wrong close reason: got %q, want %q",
			finalHandler.GetCloseReason(), reason)
	}

	if !finalHandler.WasOnErrorCalled() {
		t.Error("Final handler OnError was not called")
	}
	if finalHandler.GetError() != err {
		t.Errorf("Final handler received wrong error: got %v, want %v",
			finalHandler.GetError(), err)
	}
}

// TestMiddlewareChaining tests chaining multiple middleware together
func TestMiddlewareChaining(t *testing.T) {
	// Create a recording handler to check the final results
	finalHandler := newRecordingTransportHandler()

	// Create middleware that adds prefixes to messages
	addPrefix := func(prefix string) Middleware {
		return func(handler TransportHandler) TransportHandler {
			return NewTransportHandler(
				handler.OnOpen,
				func(message []byte) {
					// Add prefix to message
					newMessage := append([]byte(prefix), message...)
					handler.OnMessage(newMessage)
				},
				handler.OnClose,
				handler.OnError,
			)
		}
	}

	// Chain middleware together
	middleware1 := addPrefix("1:")
	middleware2 := addPrefix("2:")
	middleware3 := addPrefix("3:")

	// Apply middleware chain
	var handler TransportHandler = finalHandler
	handler = middleware1(handler) // Applied first (innermost)
	handler = middleware2(handler) // Applied second
	handler = middleware3(handler) // Applied last (outermost)

	// Test the middleware chain
	message := []byte("test")
	handler.OnMessage(message)

	// Check the result
	if !finalHandler.WasOnMessageCalled() {
		t.Error("Final handler OnMessage was not called")
	}

	// The message should have all prefixes in the correct order
	expected := "1:2:3:test"
	actual := string(finalHandler.GetMessage())
	if actual != expected {
		t.Errorf("Final handler received wrong message: got %q, want %q", actual, expected)
	}
}

// TestMiddlewareModifyingHandler tests middleware that completely changes handler behavior
func TestMiddlewareModifyingHandler(t *testing.T) {
	// Create a recording handler
	originalHandler := newRecordingTransportHandler()

	// Create middleware that ignores certain messages
	ignoreMiddleware := func(handler TransportHandler) TransportHandler {
		return NewTransportHandler(
			handler.OnOpen,
			func(message []byte) {
				// Only pass messages that don't start with "ignore:"
				if len(message) < 7 || string(message[:7]) != "ignore:" {
					handler.OnMessage(message)
				}
				// Otherwise ignore the message
			},
			handler.OnClose,
			handler.OnError,
		)
	}

	// Apply middleware
	handlerWithMiddleware := ignoreMiddleware(originalHandler)

	// Test with message that should be ignored
	ignoreMessage := []byte("ignore:this message")
	handlerWithMiddleware.OnMessage(ignoreMessage)

	// Check that the original handler was not called
	if originalHandler.WasOnMessageCalled() {
		t.Error("Original handler was called for ignored message")
	}

	// Test with message that should be passed through
	passMessage := []byte("pass this message")
	handlerWithMiddleware.OnMessage(passMessage)

	// Check that the original handler was called
	if !originalHandler.WasOnMessageCalled() {
		t.Error("Original handler was not called for pass-through message")
	}

	// Check that the correct message was passed
	if string(originalHandler.GetMessage()) != string(passMessage) {
		t.Errorf("Original handler received wrong message: got %q, want %q",
			originalHandler.GetMessage(), passMessage)
	}
}

// TestMiddlewareMux tests the MiddlewareMux functionality
func TestMiddlewareMux(t *testing.T) {
	mux := NewMiddlewareMux()
	finalHandler := newRecordingTransportHandler()

	var middlewareCalled bool
	middleware := func(handler TransportHandler) TransportHandler {
		return NewTransportHandler(
			func(conn cbio.WriteCloser) {
				middlewareCalled = true
				handler.OnOpen(conn)
			},
			func(message []byte) {
				handler.OnMessage(message)
			},
			func(code int, reason string) {
				handler.OnClose(code, reason)
			},
			func(err error) {
				handler.OnError(err)
			},
		)
	}

	mux.Use(middleware)
	mux.Handler(finalHandler)

	conn := newMockCbioWriteCloser()
	mux.OnOpen(conn)

	if !middlewareCalled {
		t.Error("Expected middleware to be called")
	}
	if !finalHandler.WasOnOpenCalled() {
		t.Error("Expected final handler to be called")
	}
}

// TestMiddlewarePanicOnLateMiddleware tests that adding middleware after setting handler panics
func TestMiddlewarePanicOnLateMiddleware(t *testing.T) {
	mux := NewMiddlewareMux()
	finalHandler := newRecordingTransportHandler()

	mux.Handler(finalHandler)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when adding middleware after handler")
		}
	}()

	mux.Use(func(handler TransportHandler) TransportHandler {
		return handler
	})
}

// TestStack tests the Stack interface
func TestStack(t *testing.T) {
	stack := NewStack()
	finalHandler := newRecordingTransportHandler()

	var middlewareCalled bool
	middleware := func(handler TransportHandler) TransportHandler {
		return NewTransportHandler(
			func(conn cbio.WriteCloser) {
				middlewareCalled = true
				handler.OnOpen(conn)
			},
			func(message []byte) {
				handler.OnMessage(message)
			},
			func(code int, reason string) {
				handler.OnClose(code, reason)
			},
			func(err error) {
				handler.OnError(err)
			},
		)
	}

	stack.Use(middleware)
	stack.Handler(finalHandler)

	conn := newMockCbioWriteCloser()
	stack.OnOpen(conn)

	if !middlewareCalled {
		t.Error("Expected middleware to be called")
	}
	if !finalHandler.WasOnOpenCalled() {
		t.Error("Expected final handler to be called")
	}
}

// TestMiddlewareCallbackHandling tests middleware with callback I/O operations
func TestMiddlewareCallbackHandling(t *testing.T) {
	finalHandler := newRecordingTransportHandler()

	// Create middleware that performs write operations with callbacks
	var writeCallbackSuccess bool
	var writeCallbackError error

	middleware := func(handler TransportHandler) TransportHandler {
		return NewTransportHandler(
			func(conn cbio.WriteCloser) {
				// Test writing to the connection using callbacks
				writeHandler := cbio.NewWriterHandler(
					func(n int) {
						writeCallbackSuccess = true
					},
					func(err error) {
						writeCallbackError = err
					},
				)

				testData := []byte("middleware test data")
				ctx, err := conn.Write(testData, writeHandler)
				if err != nil {
					t.Errorf("Unexpected error from Write: %v", err)
				}
				if ctx != nil {
					// Context should be available for cancellation
				}

				handler.OnOpen(conn)
			},
			func(message []byte) {
				handler.OnMessage(message)
			},
			func(code int, reason string) {
				handler.OnClose(code, reason)
			},
			func(err error) {
				handler.OnError(err)
			},
		)
	}

	handlerWithMiddleware := middleware(finalHandler)
	conn := newMockCbioWriteCloser()

	// Test OnOpen with callback operations
	handlerWithMiddleware.OnOpen(conn)

	// Give time for async callbacks to complete
	time.Sleep(10 * time.Millisecond)

	if !writeCallbackSuccess {
		t.Error("Expected write callback to succeed")
	}
	if writeCallbackError != nil {
		t.Errorf("Unexpected write callback error: %v", writeCallbackError)
	}
	if !finalHandler.WasOnOpenCalled() {
		t.Error("Expected final handler to be called")
	}

	// Verify that data was written to the mock connection
	writtenData := conn.GetWrittenData()
	expectedData := "middleware test data"
	if string(writtenData) != expectedData {
		t.Errorf("Expected written data %q, got %q", expectedData, string(writtenData))
	}
}

// TestMiddlewareErrorPropagation tests error propagation through middleware chain
func TestMiddlewareErrorPropagation(t *testing.T) {
	finalHandler := newRecordingTransportHandler()

	var middleware1Errors []error
	var middleware2Errors []error

	middleware1 := func(handler TransportHandler) TransportHandler {
		return NewTransportHandler(
			handler.OnOpen,
			handler.OnMessage,
			handler.OnClose,
			func(err error) {
				middleware1Errors = append(middleware1Errors, err)
				handler.OnError(err)
			},
		)
	}

	middleware2 := func(handler TransportHandler) TransportHandler {
		return NewTransportHandler(
			handler.OnOpen,
			handler.OnMessage,
			handler.OnClose,
			func(err error) {
				middleware2Errors = append(middleware2Errors, err)
				handler.OnError(err)
			},
		)
	}

	// Chain middlewares
	chainedHandler := middleware1(middleware2(finalHandler))

	// Test error propagation
	testErr := &testError{msg: "propagation test error"}
	chainedHandler.OnError(testErr)

	// Check that error propagated through all layers
	if len(middleware1Errors) != 1 || middleware1Errors[0] != testErr {
		t.Errorf("Expected middleware1 to receive error %v, got %v", testErr, middleware1Errors)
	}
	if len(middleware2Errors) != 1 || middleware2Errors[0] != testErr {
		t.Errorf("Expected middleware2 to receive error %v, got %v", testErr, middleware2Errors)
	}
	if !finalHandler.WasOnErrorCalled() || finalHandler.GetError() != testErr {
		t.Errorf("Expected final handler to receive error %v, got %v", testErr, finalHandler.GetError())
	}
}

// TestMiddlewareCallbackErrorHandling tests middleware handling of callback errors
func TestMiddlewareCallbackErrorHandling(t *testing.T) {
	finalHandler := newRecordingTransportHandler()

	var callbackErrorReceived error

	middleware := func(handler TransportHandler) TransportHandler {
		return NewTransportHandler(
			func(conn cbio.WriteCloser) {
				// Configure mock to return error on write
				mockConn := conn.(*mockCbioWriteCloser)
				mockConn.SetWriteError(errors.New("mock write error"))

				// Test writing with error handling
				writeHandler := cbio.NewWriterHandler(
					func(n int) {
						t.Error("Success callback should not be called on error")
					},
					func(err error) {
						callbackErrorReceived = err
					},
				)

				testData := []byte("test data")
				_, err := conn.Write(testData, writeHandler)
				if err == nil {
					t.Error("Expected error from Write operation")
				}

				handler.OnOpen(conn)
			},
			handler.OnMessage,
			handler.OnClose,
			handler.OnError,
		)
	}

	handlerWithMiddleware := middleware(finalHandler)
	conn := newMockCbioWriteCloser()

	handlerWithMiddleware.OnOpen(conn)

	// Give time for async callbacks to complete
	time.Sleep(10 * time.Millisecond)

	if callbackErrorReceived == nil {
		t.Error("Expected callback error to be received")
	}
	if callbackErrorReceived.Error() != "mock write error" {
		t.Errorf("Expected callback error 'mock write error', got %v", callbackErrorReceived)
	}
}

// TestMiddlewareCancellation tests middleware handling of context cancellation
func TestMiddlewareCancellation(t *testing.T) {
	finalHandler := newRecordingTransportHandler()

	var contextCancelled bool

	middleware := func(handler TransportHandler) TransportHandler {
		return NewTransportHandler(
			func(conn cbio.WriteCloser) {
				// Test writing with cancellation
				writeHandler := cbio.NewWriterHandler(
					func(n int) {
						// Should not be called if cancelled
					},
					func(err error) {
						// Should not be called if cancelled properly
					},
				)

				testData := []byte("test data for cancellation")
				ctx, err := conn.Write(testData, writeHandler)
				if err != nil {
					t.Errorf("Unexpected error from Write: %v", err)
				}

				// Cancel the context immediately
				if ctx != nil {
					ctx.Cancel()

					// Check if context is done
					select {
					case <-ctx.Done():
						contextCancelled = true
					case <-time.After(100 * time.Millisecond):
						t.Error("Context was not cancelled in time")
					}
				}

				handler.OnOpen(conn)
			},
			handler.OnMessage,
			handler.OnClose,
			handler.OnError,
		)
	}

	handlerWithMiddleware := middleware(finalHandler)
	conn := newMockCbioWriteCloser()

	handlerWithMiddleware.OnOpen(conn)

	if !contextCancelled {
		t.Error("Expected context to be cancelled")
	}
	if !finalHandler.WasOnOpenCalled() {
		t.Error("Expected final handler to be called")
	}
}
