package netkit

import (
	"io"
	"testing"
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
			func(conn io.WriteCloser) {
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
	conn := newMockReadWriteCloser()
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
