package net

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"
)

func TestNewTransportHandler(t *testing.T) {
	var (
		openCalled    bool
		messageCalled bool
		closeCalled   bool
		errorCalled   bool
	)

	handler := NewTransportHandler(
		func(conn io.WriteCloser) { openCalled = true },
		func(message []byte) { messageCalled = true },
		func(code int, reason string) { closeCalled = true },
		func(err error) { errorCalled = true },
	)

	// Test that the handler calls the correct functions
	handler.OnOpen(nil)
	if !openCalled {
		t.Error("OnOpen did not call the handler function")
	}

	handler.OnMessage([]byte("test"))
	if !messageCalled {
		t.Error("OnMessage did not call the handler function")
	}

	handler.OnClose(0, "test")
	if !closeCalled {
		t.Error("OnClose did not call the handler function")
	}

	handler.OnError(ErrTest)
	if !errorCalled {
		t.Error("OnError did not call the handler function")
	}
}

func TestTransportActorWrite(t *testing.T) {
	ctx := context.Background()
	rwc := newMockReadWriteCloser()
	config := DefaultRwcConfig()

	actor := newTransportActor(ctx, rwc, config)

	// Test normal write
	data := []byte("test data")
	n, err := actor.Write(data)
	if err != nil {
		t.Errorf("Write returned error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned wrong length: got %d, want %d", n, len(data))
	}

	writtenData := rwc.GetWrittenData()
	if string(writtenData) != string(data) {
		t.Errorf("Write wrote wrong data: got %q, want %q", writtenData, data)
	}

	// Test write error
	expectedErr := newTestError("write error")
	rwc.SetWriteError(expectedErr)
	_, err = actor.Write(data)
	if err != expectedErr {
		t.Errorf("Write did not return expected error: got %v, want %v", err, expectedErr)
	}

	// Test write after close
	actor.Close()
	_, err = actor.Write(data)
	if err != io.ErrClosedPipe {
		t.Errorf("Write after close did not return ErrClosedPipe: got %v", err)
	}
}

func TestTransportActorClose(t *testing.T) {
	ctx := context.Background()
	rwc := newMockReadWriteCloser()
	config := DefaultRwcConfig()

	actor := newTransportActor(ctx, rwc, config)

	// Test normal close
	err := actor.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}
	if !rwc.IsClosed() {
		t.Error("Close did not close the underlying ReadWriteCloser")
	}

	// Test double close
	err = actor.Close()
	if err != nil {
		t.Errorf("Second Close returned error: %v", err)
	}

	// Test close error
	rwc = newMockReadWriteCloser()
	actor = newTransportActor(ctx, rwc, config)
	expectedErr := newTestError("close error")
	rwc.SetCloseError(expectedErr)
	err = actor.Close()
	if err != expectedErr {
		t.Errorf("Close did not return expected error: got %v, want %v", err, expectedErr)
	}
}

func TestFromReaderWriterCloser(t *testing.T) {
	ctx := context.Background()
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
		return 0, io.EOF
	})

	// Create handler
	handler := newRecordingTransportHandler()

	// Create transport function
	transportFunc := FromReaderWriterCloser(ctx, rwc)

	// Start transport
	closer := transportFunc(handler)

	// Give time for goroutine to run
	time.Sleep(50 * time.Millisecond)

	// Check that OnOpen was called
	if !handler.WasOnOpenCalled() {
		t.Error("OnOpen was not called")
	}

	// Check that OnMessage was called with correct data
	if !handler.WasOnMessageCalled() {
		t.Error("OnMessage was not called")
	}
	message := handler.GetMessage()
	expectedData := "test message"
	if string(message) != expectedData {
		t.Errorf("OnMessage received wrong data: got %q, want %q", message, expectedData)
	}

	// Test Close
	err := closer.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}

	// Give time for goroutine to complete
	time.Sleep(100 * time.Millisecond)

	// Check that OnClose was called
	if !handler.WasOnCloseCalled() {
		t.Error("OnClose was not called")
	}
}

func TestTransportActorStartWithError(t *testing.T) {
	ctx := context.Background()
	rwc := newMockReadWriteCloser()
	config := DefaultRwcConfig()

	// Set up read error
	expectedErr := newTestError("read error")
	rwc.SetReadError(expectedErr)

	// Create handler
	handler := newRecordingTransportHandler()

	// Create and start actor
	actor := newTransportActor(ctx, rwc, config)
	go actor.start(handler)

	// Give time for goroutine to run
	time.Sleep(50 * time.Millisecond)

	// Check that OnError was called with correct error
	if !handler.WasOnErrorCalled() {
		t.Error("OnError was not called")
	}
	err := handler.GetError()
	if err != expectedErr {
		t.Errorf("OnError received wrong error: got %v, want %v", err, expectedErr)
	}
}

func TestTransportActorStartWithCloseError(t *testing.T) {
	ctx := context.Background()
	rwc := newMockReadWriteCloser()
	config := DefaultRwcConfig()

	// Set up close error
	closeErr := newMockCloseError(1000, "normal closure")
	rwc.SetReadError(closeErr)

	// Create handler
	handler := newRecordingTransportHandler()

	// Create and start actor
	actor := newTransportActor(ctx, rwc, config)
	go actor.start(handler)

	// Give time for goroutine to run
	time.Sleep(50 * time.Millisecond)

	// Check that OnClose was called with correct code and reason
	if !handler.WasOnCloseCalled() {
		t.Error("OnClose was not called")
	}
	code := handler.GetCloseCode()
	reason := handler.GetCloseReason()
	if code != 1000 || reason != "normal closure" {
		t.Errorf("OnClose received wrong code/reason: got %d/%s, want 1000/normal closure", code, reason)
	}
}

func TestTransportActorStartWithContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	rwc := newMockReadWriteCloser()
	config := DefaultRwcConfig()

	// Make read block
	rwc.SetReadDelay(1 * time.Second)

	// Create handler
	handler := newRecordingTransportHandler()

	// Create and start actor
	actor := newTransportActor(ctx, rwc, config)
	go actor.start(handler)

	// Cancel context
	cancel()

	// Give time for goroutine to run
	time.Sleep(50 * time.Millisecond)

	// Check that OnClose was called with context canceled
	if !handler.WasOnCloseCalled() {
		t.Error("OnClose was not called")
	}
	reason := handler.GetCloseReason()
	if reason != "context canceled" {
		t.Errorf("OnClose received wrong reason: got %s, want 'context canceled'", reason)
	}
}

func TestActiveTransportHandlerClose(t *testing.T) {
	ctx := context.Background()
	rwc := newMockReadWriteCloser()

	// Create handler
	handler := newRecordingTransportHandler()

	// Create transport function
	transportFunc := FromReaderWriterCloser(ctx, rwc)

	// Start transport
	closer := transportFunc(handler)

	// Close with potential timeout
	err := closer.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

func TestConcurrentWrites(t *testing.T) {
	ctx := context.Background()
	rwc := newMockReadWriteCloser()
	config := DefaultRwcConfig()

	actor := newTransportActor(ctx, rwc, config)

	// Perform concurrent writes
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			data := []byte{byte(i)}
			actor.Write(data)
		}(i)
	}

	wg.Wait()

	// Check that all writes were performed
	writtenData := rwc.GetWrittenData()
	if len(writtenData) != 10 {
		t.Errorf("Expected 10 bytes written, got %d", len(writtenData))
	}
}

func TestWithReaderBufferSize(t *testing.T) {
	config := DefaultRwcConfig()

	// Default buffer size should be 4096
	if config.ReaderBufferSize != 4096 {
		t.Errorf("Default ReaderBufferSize is %d, want 4096", config.ReaderBufferSize)
	}

	// Test custom buffer size
	customSize := 8192
	option := WithReaderBufferSize(customSize)
	option(config)

	if config.ReaderBufferSize != customSize {
		t.Errorf("Custom ReaderBufferSize is %d, want %d", config.ReaderBufferSize, customSize)
	}
}
