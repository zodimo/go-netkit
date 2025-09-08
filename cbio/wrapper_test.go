package cbio

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

// mockReadWriteCloser is a test implementation of io.ReadWriteCloser
type mockReadWriteCloser struct {
	readData    []byte
	readIndex   int
	readError   error
	writeError  error
	closeError  error
	readDelay   time.Duration
	writeDelay  time.Duration
	closeDelay  time.Duration
	writtenData []byte
	closed      bool
	mutex       sync.Mutex
}

func newMockReadWriteCloser() *mockReadWriteCloser {
	return &mockReadWriteCloser{
		readData:    []byte("test data"),
		writtenData: make([]byte, 0),
	}
}

func (m *mockReadWriteCloser) Read(p []byte) (n int, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.readDelay > 0 {
		time.Sleep(m.readDelay)
	}

	if m.readError != nil {
		return 0, m.readError
	}

	if m.readIndex >= len(m.readData) {
		return 0, io.EOF
	}

	n = copy(p, m.readData[m.readIndex:])
	m.readIndex += n
	return n, nil
}

func (m *mockReadWriteCloser) Write(p []byte) (n int, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.writeDelay > 0 {
		time.Sleep(m.writeDelay)
	}

	if m.writeError != nil {
		return 0, m.writeError
	}

	m.writtenData = append(m.writtenData, p...)
	return len(p), nil
}

func (m *mockReadWriteCloser) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.closeDelay > 0 {
		time.Sleep(m.closeDelay)
	}

	if m.closeError != nil {
		return m.closeError
	}

	m.closed = true
	return nil
}

func (m *mockReadWriteCloser) setReadError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.readError = err
}

func (m *mockReadWriteCloser) setWriteError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.writeError = err
}

func (m *mockReadWriteCloser) setCloseError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.closeError = err
}

func (m *mockReadWriteCloser) getWrittenData() []byte {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return append([]byte(nil), m.writtenData...)
}

func (m *mockReadWriteCloser) isClosed() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.closed
}

func TestWrapReadWriteCloser(t *testing.T) {
	mock := newMockReadWriteCloser()
	wrapper := WrapReadWriteCloser(mock)

	if wrapper == nil {
		t.Fatal("WrapReadWriteCloser returned nil")
	}

	// Verify it implements the correct interface
	var _ ReadWriteCloser = wrapper
}

func TestWrapper_Read_Success(t *testing.T) {
	mock := newMockReadWriteCloser()
	wrapper := WrapReadWriteCloser(mock)

	var receivedData []byte
	var receivedError error
	var wg sync.WaitGroup
	wg.Add(1)

	readHandler := NewReaderHandler(
		func(data []byte) {
			receivedData = append([]byte(nil), data...)
			wg.Done()
		},
		func(err error) {
			receivedError = err
			wg.Done()
		},
	)

	ctx, err := wrapper.Read(readHandler)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}

	wg.Wait()
	<-ctx.Done()

	if receivedError != nil {
		t.Fatalf("Expected no error, got: %v", receivedError)
	}

	expected := "test data"
	if string(receivedData) != expected {
		t.Fatalf("Expected data %q, got %q", expected, string(receivedData))
	}
}

func TestWrapper_Read_Error(t *testing.T) {
	mock := newMockReadWriteCloser()
	expectedError := errors.New("read error")
	mock.setReadError(expectedError)
	wrapper := WrapReadWriteCloser(mock)

	var receivedData []byte
	var receivedError error
	var wg sync.WaitGroup
	wg.Add(1)

	readHandler := NewReaderHandler(
		func(data []byte) {
			receivedData = data
			wg.Done()
		},
		func(err error) {
			receivedError = err
			wg.Done()
		},
	)

	ctx, err := wrapper.Read(readHandler)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}

	wg.Wait()
	<-ctx.Done()

	if receivedError != expectedError {
		t.Fatalf("Expected error %v, got %v", expectedError, receivedError)
	}

	if receivedData != nil {
		t.Fatalf("Expected no data, got %v", receivedData)
	}
}

func TestWrapper_Read_Timeout(t *testing.T) {
	mock := newMockReadWriteCloser()
	mock.readDelay = 200 * time.Millisecond
	wrapper := WrapReadWriteCloser(mock)

	var receivedError error
	var wg sync.WaitGroup
	wg.Add(1)

	readHandler := NewReaderHandler(
		func(data []byte) {
			t.Error("Should not receive success callback on timeout")
			wg.Done()
		},
		func(err error) {
			receivedError = err
			wg.Done()
		},
	)

	ctx, err := wrapper.Read(readHandler, WithReadTimeout(50*time.Millisecond))
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}

	wg.Wait()
	<-ctx.Done()

	if receivedError != context.DeadlineExceeded {
		t.Fatalf("Expected timeout error, got: %v", receivedError)
	}
}

func TestWrapper_Read_Cancellation(t *testing.T) {
	mock := newMockReadWriteCloser()
	mock.readDelay = 200 * time.Millisecond
	wrapper := WrapReadWriteCloser(mock)

	var receivedError error
	var wg sync.WaitGroup
	wg.Add(1)

	readHandler := NewReaderHandler(
		func(data []byte) {
			t.Error("Should not receive success callback on cancellation")
			wg.Done()
		},
		func(err error) {
			receivedError = err
			wg.Done()
		},
	)

	ctx, err := wrapper.Read(readHandler)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		ctx.Cancel()
	}()

	wg.Wait()
	<-ctx.Done()

	if receivedError != context.Canceled {
		t.Fatalf("Expected cancellation error, got: %v", receivedError)
	}
}

func TestWrapper_Read_NilHandler(t *testing.T) {
	mock := newMockReadWriteCloser()
	wrapper := WrapReadWriteCloser(mock)

	ctx, err := wrapper.Read(nil)
	if err == nil {
		t.Fatal("Expected error for nil handler")
	}
	if ctx != nil {
		t.Fatal("Expected nil context for nil handler")
	}
}

func TestWrapper_Write_Success(t *testing.T) {
	mock := newMockReadWriteCloser()
	wrapper := WrapReadWriteCloser(mock)

	testData := []byte("hello world")
	var bytesWritten int
	var receivedError error
	var wg sync.WaitGroup
	wg.Add(1)

	writeHandler := NewWriterHandler(
		func(n int) {
			bytesWritten = n
			wg.Done()
		},
		func(err error) {
			receivedError = err
			wg.Done()
		},
	)

	ctx, err := wrapper.Write(testData, writeHandler)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	wg.Wait()
	<-ctx.Done()

	if receivedError != nil {
		t.Fatalf("Expected no error, got: %v", receivedError)
	}

	if bytesWritten != len(testData) {
		t.Fatalf("Expected %d bytes written, got %d", len(testData), bytesWritten)
	}

	writtenData := mock.getWrittenData()
	if string(writtenData) != string(testData) {
		t.Fatalf("Expected written data %q, got %q", string(testData), string(writtenData))
	}
}

func TestWrapper_Write_Error(t *testing.T) {
	mock := newMockReadWriteCloser()
	expectedError := errors.New("write error")
	mock.setWriteError(expectedError)
	wrapper := WrapReadWriteCloser(mock)

	testData := []byte("hello world")
	var bytesWritten int
	var receivedError error
	var wg sync.WaitGroup
	wg.Add(1)

	writeHandler := NewWriterHandler(
		func(n int) {
			bytesWritten = n
			wg.Done()
		},
		func(err error) {
			receivedError = err
			wg.Done()
		},
	)

	ctx, err := wrapper.Write(testData, writeHandler)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	wg.Wait()
	<-ctx.Done()

	if receivedError != expectedError {
		t.Fatalf("Expected error %v, got %v", expectedError, receivedError)
	}

	if bytesWritten != 0 {
		t.Fatalf("Expected 0 bytes written on error, got %d", bytesWritten)
	}
}

func TestWrapper_Write_EmptyData(t *testing.T) {
	mock := newMockReadWriteCloser()
	wrapper := WrapReadWriteCloser(mock)

	var bytesWritten int
	var receivedError error
	var wg sync.WaitGroup
	wg.Add(1)

	writeHandler := NewWriterHandler(
		func(n int) {
			bytesWritten = n
			wg.Done()
		},
		func(err error) {
			receivedError = err
			wg.Done()
		},
	)

	ctx, err := wrapper.Write([]byte{}, writeHandler)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	wg.Wait()
	<-ctx.Done()

	if receivedError != nil {
		t.Fatalf("Expected no error, got: %v", receivedError)
	}

	if bytesWritten != 0 {
		t.Fatalf("Expected 0 bytes written for empty data, got %d", bytesWritten)
	}
}

func TestWrapper_Write_Timeout(t *testing.T) {
	mock := newMockReadWriteCloser()
	mock.writeDelay = 200 * time.Millisecond
	wrapper := WrapReadWriteCloser(mock)

	testData := []byte("hello world")
	var receivedError error
	var wg sync.WaitGroup
	wg.Add(1)

	writeHandler := NewWriterHandler(
		func(n int) {
			t.Error("Should not receive success callback on timeout")
			wg.Done()
		},
		func(err error) {
			receivedError = err
			wg.Done()
		},
	)

	ctx, err := wrapper.Write(testData, writeHandler, WithWriteTimeout(50*time.Millisecond))
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	wg.Wait()
	<-ctx.Done()

	if receivedError != context.DeadlineExceeded {
		t.Fatalf("Expected timeout error, got: %v", receivedError)
	}
}

func TestWrapper_Write_NilHandler(t *testing.T) {
	mock := newMockReadWriteCloser()
	wrapper := WrapReadWriteCloser(mock)

	ctx, err := wrapper.Write([]byte("test"), nil)
	if err == nil {
		t.Fatal("Expected error for nil handler")
	}
	if ctx != nil {
		t.Fatal("Expected nil context for nil handler")
	}
}

func TestWrapper_Close_Success(t *testing.T) {
	mock := newMockReadWriteCloser()
	wrapper := WrapReadWriteCloser(mock)

	err := wrapper.Close()
	if err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	if !mock.isClosed() {
		t.Fatal("Expected mock to be closed")
	}
}

func TestWrapper_Close_Error(t *testing.T) {
	mock := newMockReadWriteCloser()
	expectedError := errors.New("close error")
	mock.setCloseError(expectedError)
	wrapper := WrapReadWriteCloser(mock)

	err := wrapper.Close()
	if err != expectedError {
		t.Fatalf("Expected error %v, got %v", expectedError, err)
	}
}

func TestWrapper_Close_Timeout(t *testing.T) {
	mock := newMockReadWriteCloser()
	mock.closeDelay = 200 * time.Millisecond
	wrapper := WrapReadWriteCloser(mock)

	err := wrapper.Close(WithCloseTimeout(50 * time.Millisecond))
	if err != context.DeadlineExceeded {
		t.Fatalf("Expected timeout error, got: %v", err)
	}
}

func TestWrapper_ConcurrentOperations(t *testing.T) {
	mock := newMockReadWriteCloser()
	wrapper := WrapReadWriteCloser(mock)

	const numOperations = 10
	var wg sync.WaitGroup
	wg.Add(numOperations * 2) // read and write operations

	// Start concurrent read operations
	for i := 0; i < numOperations; i++ {
		go func() {
			defer wg.Done()
			readHandler := NewReaderHandler(
				func(data []byte) {},
				func(err error) {},
			)
			ctx, err := wrapper.Read(readHandler)
			if err != nil {
				t.Errorf("Read returned error: %v", err)
				return
			}
			<-ctx.Done()
		}()
	}

	// Start concurrent write operations
	for i := 0; i < numOperations; i++ {
		go func(i int) {
			defer wg.Done()
			writeHandler := NewWriterHandler(
				func(n int) {},
				func(err error) {},
			)
			data := []byte("data" + string(rune('0'+i)))
			ctx, err := wrapper.Write(data, writeHandler)
			if err != nil {
				t.Errorf("Write returned error: %v", err)
				return
			}
			<-ctx.Done()
		}(i)
	}

	// Wait for all operations to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All operations completed successfully
	case <-time.After(5 * time.Second):
		t.Fatal("Concurrent operations timed out")
	}
}

// mockCbioReadWriteCloser is a test implementation of cbio.ReadWriteCloser
type mockCbioReadWriteCloser struct {
	readData    []byte
	readIndex   int
	readError   error
	writeError  error
	closeError  error
	readDelay   time.Duration
	writeDelay  time.Duration
	closeDelay  time.Duration
	writtenData []byte
	closed      bool
	mutex       sync.Mutex
}

func newMockCbioReadWriteCloser() *mockCbioReadWriteCloser {
	return &mockCbioReadWriteCloser{
		readData:    []byte("test data"),
		writtenData: make([]byte, 0),
	}
}

func (m *mockCbioReadWriteCloser) Read(handler ReaderHandler, options ...ReaderOption) (CbContext, error) {
	if handler == nil {
		return nil, errors.New("handler cannot be nil")
	}

	done := make(chan struct{})
	ctx := NewCbContext(func() {}, done)

	go func() {
		defer close(done)

		m.mutex.Lock()
		defer m.mutex.Unlock()

		if m.readDelay > 0 {
			time.Sleep(m.readDelay)
		}

		if m.readError != nil {
			handler.OnError(m.readError)
			return
		}

		if m.readIndex >= len(m.readData) {
			handler.OnError(io.EOF)
			return
		}

		data := m.readData[m.readIndex:]
		m.readIndex = len(m.readData)
		handler.OnSuccess(data)
	}()

	return ctx, nil
}

func (m *mockCbioReadWriteCloser) Write(p []byte, handler WriterHandler, options ...WriterOption) (CbContext, error) {
	if handler == nil {
		return nil, errors.New("handler cannot be nil")
	}

	done := make(chan struct{})
	ctx := NewCbContext(func() {}, done)

	go func() {
		defer close(done)

		m.mutex.Lock()
		defer m.mutex.Unlock()

		if m.writeDelay > 0 {
			time.Sleep(m.writeDelay)
		}

		if m.writeError != nil {
			handler.OnError(m.writeError)
			return
		}

		m.writtenData = append(m.writtenData, p...)
		handler.OnSuccess(len(p))
	}()

	return ctx, nil
}

func (m *mockCbioReadWriteCloser) Close(options ...CloserOption) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.closeDelay > 0 {
		time.Sleep(m.closeDelay)
	}

	if m.closeError != nil {
		return m.closeError
	}

	m.closed = true
	return nil
}

func TestUnwrapReadWriteCloser(t *testing.T) {
	mock := newMockCbioReadWriteCloser()
	wrapper := UnwrapReadWriteCloser(mock)

	if wrapper == nil {
		t.Fatal("UnwrapReadWriteCloser returned nil")
	}
}

func TestUnwrapReadWriteCloser_Read_Success(t *testing.T) {
	mock := newMockCbioReadWriteCloser()
	wrapper := UnwrapReadWriteCloser(mock)

	buffer := make([]byte, 1024)
	n, err := wrapper.Read(buffer)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if n != len("test data") {
		t.Errorf("Expected %d bytes read, got %d", len("test data"), n)
	}

	if string(buffer[:n]) != "test data" {
		t.Errorf("Expected 'test data', got '%s'", string(buffer[:n]))
	}
}

func TestUnwrapReadWriteCloser_Read_Error(t *testing.T) {
	mock := newMockCbioReadWriteCloser()
	mock.readError = errors.New("read error")
	wrapper := UnwrapReadWriteCloser(mock)

	buffer := make([]byte, 1024)
	n, err := wrapper.Read(buffer)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if n != 0 {
		t.Errorf("Expected 0 bytes read, got %d", n)
	}

	if err.Error() != "read error" {
		t.Errorf("Expected 'read error', got '%s'", err.Error())
	}
}

func TestUnwrapReadWriteCloser_Read_EmptyBuffer(t *testing.T) {
	mock := newMockCbioReadWriteCloser()
	wrapper := UnwrapReadWriteCloser(mock)

	buffer := make([]byte, 0)
	n, err := wrapper.Read(buffer)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if n != 0 {
		t.Errorf("Expected 0 bytes read, got %d", n)
	}
}

func TestUnwrapReadWriteCloser_Write_Success(t *testing.T) {
	mock := newMockCbioReadWriteCloser()
	wrapper := UnwrapReadWriteCloser(mock)

	data := []byte("hello world")
	n, err := wrapper.Write(data)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected %d bytes written, got %d", len(data), n)
	}

	mock.mutex.Lock()
	if string(mock.writtenData) != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", string(mock.writtenData))
	}
	mock.mutex.Unlock()
}

func TestUnwrapReadWriteCloser_Write_Error(t *testing.T) {
	mock := newMockCbioReadWriteCloser()
	mock.writeError = errors.New("write error")
	wrapper := UnwrapReadWriteCloser(mock)

	data := []byte("hello world")
	n, err := wrapper.Write(data)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if n != 0 {
		t.Errorf("Expected 0 bytes written, got %d", n)
	}

	if err.Error() != "write error" {
		t.Errorf("Expected 'write error', got '%s'", err.Error())
	}
}

func TestUnwrapReadWriteCloser_Write_EmptyData(t *testing.T) {
	mock := newMockCbioReadWriteCloser()
	wrapper := UnwrapReadWriteCloser(mock)

	data := []byte{}
	n, err := wrapper.Write(data)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if n != 0 {
		t.Errorf("Expected 0 bytes written, got %d", n)
	}
}

func TestUnwrapReadWriteCloser_Close_Success(t *testing.T) {
	mock := newMockCbioReadWriteCloser()
	wrapper := UnwrapReadWriteCloser(mock)

	err := wrapper.Close()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	mock.mutex.Lock()
	if !mock.closed {
		t.Error("Expected mock to be closed")
	}
	mock.mutex.Unlock()
}

func TestUnwrapReadWriteCloser_Close_Error(t *testing.T) {
	mock := newMockCbioReadWriteCloser()
	mock.closeError = errors.New("close error")
	wrapper := UnwrapReadWriteCloser(mock)

	err := wrapper.Close()

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err.Error() != "close error" {
		t.Errorf("Expected 'close error', got '%s'", err.Error())
	}
}

func TestUnwrapReadWriteCloser_ConcurrentOperations(t *testing.T) {
	mock := newMockCbioReadWriteCloser()
	wrapper := UnwrapReadWriteCloser(mock)

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Concurrent reads
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buffer := make([]byte, 1024)
			_, err := wrapper.Read(buffer)
			if err != nil && err != io.EOF {
				errors <- err
			}
		}()
	}

	// Concurrent writes
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			data := []byte("data" + string(rune('0'+i)))
			_, err := wrapper.Write(data)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
	}
}

func TestWrapReadWriteCloser_FullDuplexConcurrency(t *testing.T) {
	mock := newMockReadWriteCloser()
	mock.readData = []byte("concurrent test data")
	wrapper := WrapReadWriteCloser(mock)

	var wg sync.WaitGroup
	readDone := make(chan bool, 1)
	writeDone := make(chan bool, 1)

	// Start a read operation
	wg.Add(1)
	go func() {
		defer wg.Done()
		readHandler := NewReaderHandler(
			func(data []byte) {
				readDone <- true
			},
			func(err error) {
				t.Errorf("Read error: %v", err)
				readDone <- false
			},
		)
		ctx, err := wrapper.Read(readHandler)
		if err != nil {
			t.Errorf("Failed to start read: %v", err)
			return
		}
		<-ctx.Done()
	}()

	// Start a write operation concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		writeHandler := NewWriterHandler(
			func(n int) {
				writeDone <- true
			},
			func(err error) {
				t.Errorf("Write error: %v", err)
				writeDone <- false
			},
		)
		data := []byte("concurrent write data")
		ctx, err := wrapper.Write(data, writeHandler)
		if err != nil {
			t.Errorf("Failed to start write: %v", err)
			return
		}
		<-ctx.Done()
	}()

	// Wait for both operations to complete
	readSuccess := <-readDone
	writeSuccess := <-writeDone

	if !readSuccess {
		t.Error("Read operation failed")
	}
	if !writeSuccess {
		t.Error("Write operation failed")
	}

	wg.Wait()
}

func TestUnwrapReadWriteCloser_FullDuplexConcurrency(t *testing.T) {
	mock := newMockCbioReadWriteCloser()
	mock.readData = []byte("concurrent test data")
	wrapper := UnwrapReadWriteCloser(mock)

	var wg sync.WaitGroup
	readResult := make(chan error, 1)
	writeResult := make(chan error, 1)

	// Start a read operation
	wg.Add(1)
	go func() {
		defer wg.Done()
		buffer := make([]byte, 1024)
		_, err := wrapper.Read(buffer)
		readResult <- err
	}()

	// Start a write operation concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		data := []byte("concurrent write data")
		_, err := wrapper.Write(data)
		writeResult <- err
	}()

	// Wait for both operations to complete
	readErr := <-readResult
	writeErr := <-writeResult

	if readErr != nil {
		t.Errorf("Read operation failed: %v", readErr)
	}
	if writeErr != nil {
		t.Errorf("Write operation failed: %v", writeErr)
	}

	wg.Wait()
}

func TestFullDuplexPerformance(t *testing.T) {
	mock := newMockReadWriteCloser()
	mock.readData = make([]byte, 1024) // 1KB of data
	wrapper := WrapReadWriteCloser(mock)

	const numOperations = 100
	var wg sync.WaitGroup

	start := time.Now()

	// Launch concurrent read and write operations
	for i := 0; i < numOperations; i++ {
		// Read operation
		wg.Add(1)
		go func() {
			defer wg.Done()
			readHandler := NewReaderHandler(
				func(data []byte) {},
				func(err error) {
					if err != io.EOF {
						t.Errorf("Read error: %v", err)
					}
				},
			)
			ctx, err := wrapper.Read(readHandler)
			if err != nil {
				t.Errorf("Failed to start read: %v", err)
				return
			}
			<-ctx.Done()
		}()

		// Write operation
		wg.Add(1)
		go func() {
			defer wg.Done()
			writeHandler := NewWriterHandler(
				func(n int) {},
				func(err error) {
					t.Errorf("Write error: %v", err)
				},
			)
			data := make([]byte, 1024) // 1KB of data
			ctx, err := wrapper.Write(data, writeHandler)
			if err != nil {
				t.Errorf("Failed to start write: %v", err)
				return
			}
			<-ctx.Done()
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	// With proper full-duplex support, operations should complete much faster
	// than if they were serialized
	t.Logf("Full-duplex performance: %d operations completed in %v", numOperations*2, duration)

	// This is just a sanity check - actual timing will depend on the system
	if duration > 5*time.Second {
		t.Errorf("Operations took too long (%v), possible serialization issue", duration)
	}
}
