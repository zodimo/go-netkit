package net

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"
)

// TestFullTransportFlow tests the complete flow of a transport with middleware
func TestFullTransportFlow(t *testing.T) {
	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a mock that won't close after reading
	rwc := newMockReadWriteCloser()

	// Configure the mock to not close after reading
	rwc.SetOnRead(func(p []byte) (int, error) {
		// Only read once and then return EOF on subsequent reads
		// without closing the connection
		if rwc.GetReadCallCount() == 1 {
			testData := []byte("test message")
			n := copy(p, testData)
			return n, nil
		}
		<-rwc.ClosedChan
		return 0, fmt.Errorf("read error")
	})

	// Create recording handler to verify results
	finalHandler := newRecordingTransportHandler()

	// Create logging middleware
	var logs []string
	var logsMutex sync.Mutex

	loggingMiddleware := func(handler TransportHandler) TransportHandler {
		return NewTransportHandler(
			func(conn io.WriteCloser) {
				logsMutex.Lock()
				logs = append(logs, "connection opened")
				logsMutex.Unlock()
				handler.OnOpen(conn)
			},
			func(message []byte) {
				logsMutex.Lock()
				logs = append(logs, "message received: "+string(message))
				logsMutex.Unlock()
				handler.OnMessage(message)
			},
			func(code int, reason string) {
				logsMutex.Lock()
				logs = append(logs, "connection closed: "+reason)
				logsMutex.Unlock()
				handler.OnClose(code, reason)
			},
			func(err error) {
				logsMutex.Lock()
				logs = append(logs, "error: "+err.Error())
				logsMutex.Unlock()
				handler.OnError(err)
			},
		)
	}

	// Apply middleware
	handlerWithMiddleware := loggingMiddleware(finalHandler)

	// Create transport with custom buffer size
	transportFunc := FromReaderWriterCloser(ctx, rwc, WithReaderBufferSize(8192))

	// Start transport
	closer := transportFunc(handlerWithMiddleware)

	// Give time for goroutine to run
	time.Sleep(50 * time.Millisecond)

	// Check that OnOpen was called
	if !finalHandler.WasOnOpenCalled() {
		t.Error("OnOpen was not called")
	}

	// Check that OnMessage was called with correct data
	if !finalHandler.WasOnMessageCalled() {
		t.Error("OnMessage was not called")
	}
	message := finalHandler.GetMessage()
	expectedData := "test message"
	if string(message) != expectedData {
		t.Errorf("OnMessage received wrong data: got %q, want %q", message, expectedData)
	}

	// Check logs
	logsMutex.Lock()
	if len(logs) < 2 {
		t.Errorf("Expected at least 2 log entries, got %d", len(logs))
	}
	if len(logs) > 0 && logs[0] != "connection opened" {
		t.Errorf("First log entry is %q, want 'connection opened'", logs[0])
	}
	if len(logs) > 1 && logs[1] != "message received: "+expectedData {
		t.Errorf("Second log entry is %q, want 'message received: %s'", logs[1], expectedData)
	}
	logsMutex.Unlock()

	// Test writing data
	writeData := []byte("response data")
	conn := finalHandler.GetConn()
	n, err := conn.Write(writeData)
	if err != nil {
		t.Errorf("Write returned error: %v", err)
	}
	if n != len(writeData) {
		t.Errorf("Write returned wrong length: got %d, want %d", n, len(writeData))
	}

	// Check that data was written to the mock
	writtenData := rwc.GetWrittenData()
	if string(writtenData) != string(writeData) {
		t.Errorf("Write wrote wrong data: got %q, want %q", writtenData, writeData)
	}

	// Close the connection
	err = closer.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}

	// Give time for goroutine to complete
	time.Sleep(50 * time.Millisecond)

	// Check that OnClose was called
	if !finalHandler.WasOnCloseCalled() {
		t.Error("OnClose was not called")
	}

	// Check that the connection is closed
	if !rwc.IsClosed() {
		t.Error("Connection was not closed")
	}
}

// TestMultipleMiddleware tests stacking multiple middleware
func TestMultipleMiddleware(t *testing.T) {
	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create mock connection
	rwc := newMockReadWriteCloser()

	// Set up test data
	testData := []byte("test message")
	rwc.SetReadData(testData)

	// Create recording handler to verify results
	finalHandler := newRecordingTransportHandler()

	// Create middleware that adds a prefix to messages
	prefixMiddleware := func(prefix string) Middleware {
		return func(handler TransportHandler) TransportHandler {
			return NewTransportHandler(
				handler.OnOpen,
				func(message []byte) {
					newMessage := append([]byte(prefix), message...)
					handler.OnMessage(newMessage)
				},
				handler.OnClose,
				handler.OnError,
			)
		}
	}

	// Create middleware that converts messages to uppercase
	uppercaseMiddleware := func(handler TransportHandler) TransportHandler {
		return NewTransportHandler(
			handler.OnOpen,
			func(message []byte) {
				// Convert to uppercase
				upper := make([]byte, len(message))
				for i, b := range message {
					if b >= 'a' && b <= 'z' {
						upper[i] = b - ('a' - 'A')
					} else {
						upper[i] = b
					}
				}
				handler.OnMessage(upper)
			},
			handler.OnClose,
			handler.OnError,
		)
	}

	// Apply middleware (order matters)
	var handler TransportHandler = finalHandler
	handler = uppercaseMiddleware(handler)          // Applied first (innermost)
	handler = prefixMiddleware("PREFIX: ")(handler) // Applied second (outermost)

	// Create transport
	transportFunc := FromReaderWriterCloser(ctx, rwc)

	// Start transport
	closer := transportFunc(handler)
	defer closer.Close()

	// Give time for goroutine to run
	time.Sleep(50 * time.Millisecond)

	// Check that OnMessage was called
	if !finalHandler.WasOnMessageCalled() {
		t.Error("OnMessage was not called")
	}

	// Check that the message was transformed correctly
	// First uppercase, then prefix
	expected := "PREFIX: TEST MESSAGE"
	actual := string(finalHandler.GetMessage())
	if actual != expected {
		t.Errorf("Message transformation wrong: got %q, want %q", actual, expected)
	}
}

// TestErrorHandling tests error propagation through the transport
func TestErrorHandling(t *testing.T) {
	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create mock connection
	rwc := newMockReadWriteCloser()

	// Set up read error
	expectedErr := newTestError("read error")
	rwc.SetReadError(expectedErr)

	// Create recording handler
	handler := newRecordingTransportHandler()

	// Create error handling middleware
	var errorHandled bool
	var errorHandledMutex sync.Mutex

	errorMiddleware := func(h TransportHandler) TransportHandler {
		return NewTransportHandler(
			h.OnOpen,
			h.OnMessage,
			h.OnClose,
			func(err error) {
				// Mark that we handled the error
				errorHandledMutex.Lock()
				errorHandled = true
				errorHandledMutex.Unlock()

				// Pass to next handler
				h.OnError(err)
			},
		)
	}

	// Apply middleware
	handlerWithMiddleware := errorMiddleware(handler)

	// Create transport
	transportFunc := FromReaderWriterCloser(ctx, rwc)

	// Start transport
	closer := transportFunc(handlerWithMiddleware)
	defer closer.Close()

	// Give time for goroutine to run
	time.Sleep(50 * time.Millisecond)

	// Check that OnError was called
	if !handler.WasOnErrorCalled() {
		t.Error("OnError was not called")
	}

	// Check that the error was passed correctly
	err := handler.GetError()
	if err != expectedErr {
		t.Errorf("OnError received wrong error: got %v, want %v", err, expectedErr)
	}

	// Check that middleware handled the error
	errorHandledMutex.Lock()
	if !errorHandled {
		t.Error("Error middleware did not handle the error")
	}
	errorHandledMutex.Unlock()
}

// TestCloseErrorHandling tests handling of close errors
func TestCloseErrorHandling(t *testing.T) {
	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create mock connection
	rwc := newMockReadWriteCloser()

	// Set up close error
	closeErr := NewCloseError(1001, "going away")
	rwc.SetReadError(closeErr)

	// Create recording handler
	handler := newRecordingTransportHandler()

	// Create transport
	transportFunc := FromReaderWriterCloser(ctx, rwc)

	// Start transport
	closer := transportFunc(handler)
	defer closer.Close()

	// Give time for goroutine to run
	time.Sleep(50 * time.Millisecond)

	// Check that OnClose was called
	if !handler.WasOnCloseCalled() {
		t.Error("OnClose was not called")
	}

	// Check that the close code and reason were passed correctly
	code := handler.GetCloseCode()
	reason := handler.GetCloseReason()
	if code != 1001 || reason != "going away" {
		t.Errorf("OnClose received wrong code/reason: got %d/%s, want 1001/going away", code, reason)
	}

	// Check that OnError was not called
	if handler.WasOnErrorCalled() {
		t.Error("OnError was called for a close error")
	}
}
