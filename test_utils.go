package netkit

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/zodimo/go-netkit/cbio"
)

// mockReadWriteCloser implements io.ReadWriteCloser for testing
type mockReadWriteCloser struct {
	readData       []byte
	readError      error
	writtenData    []byte
	writeError     error
	closeError     error
	closed         bool
	ClosedChan     chan struct{}
	mu             sync.Mutex
	readLock       sync.Mutex
	writeLock      sync.Mutex
	closeLock      sync.Mutex
	readDelay      time.Duration
	writeDelay     time.Duration
	closeDelay     time.Duration
	readCallCount  int
	writeCallCount int
	closeCallCount int
	onRead         func(p []byte) (int, error)
	onWrite        func(p []byte) (int, error)
	onClose        func() error
	lockedFrom     string
	debug          bool
}

// newMockReadWriteCloser creates a new mock ReadWriteCloser for testing
func newMockReadWriteCloser() *mockReadWriteCloser {
	return &mockReadWriteCloser{
		readData:    []byte{},
		ClosedChan:  make(chan struct{}),
		writtenData: []byte{},
	}
}

// SetReadData sets the data that will be returned by Read
func (m *mockReadWriteCloser) SetReadData(data []byte) {
	m.Lock("SetReadData")
	defer m.Unlock()
	m.readData = data
}

// SetReadError sets the error that will be returned by Read
func (m *mockReadWriteCloser) SetReadError(err error) {
	m.Lock("SetReadError")
	defer m.Unlock()
	m.readError = err
}

// SetWriteError sets the error that will be returned by Write
func (m *mockReadWriteCloser) SetWriteError(err error) {
	m.Lock("SetWriteError")
	defer m.Unlock()
	m.writeError = err
}

// SetCloseError sets the error that will be returned by Close
func (m *mockReadWriteCloser) SetCloseError(err error) {
	m.Lock("SetCloseError")
	defer m.Unlock()
	m.closeError = err
}

// SetReadDelay sets a delay before Read returns
func (m *mockReadWriteCloser) SetReadDelay(delay time.Duration) {
	m.Lock("SetReadDelay")
	defer m.Unlock()
	m.readDelay = delay
}

// SetWriteDelay sets a delay before Write returns
func (m *mockReadWriteCloser) SetWriteDelay(delay time.Duration) {
	m.Lock("SetWriteDelay")
	defer m.Unlock()
	m.writeDelay = delay
}

// SetCloseDelay sets a delay before Close returns
func (m *mockReadWriteCloser) SetCloseDelay(delay time.Duration) {
	m.Lock("SetCloseDelay")
	defer m.Unlock()
	m.closeDelay = delay
}

// SetOnRead sets a custom function to be called on Read
func (m *mockReadWriteCloser) SetOnRead(fn func(p []byte) (int, error)) {
	m.Lock("SetOnRead")
	defer m.Unlock()
	m.onRead = fn
}

// SetOnWrite sets a custom function to be called on Write
func (m *mockReadWriteCloser) SetOnWrite(fn func(p []byte) (int, error)) {
	m.Lock("SetOnWrite")
	defer m.Unlock()
	m.onWrite = fn
}

// SetOnClose sets a custom function to be called on Close
func (m *mockReadWriteCloser) SetOnClose(fn func() error) {
	m.Lock("SetOnClose")
	defer m.Unlock()
	m.onClose = fn
}

// GetWrittenData returns the data written to the mock
func (m *mockReadWriteCloser) GetWrittenData() []byte {
	m.Lock("GetWrittenData")
	defer m.Unlock()
	return m.writtenData
}

// IsClosed returns whether Close has been called
func (m *mockReadWriteCloser) IsClosed() bool {
	m.Lock("IsClosed")
	defer m.Unlock()
	return m.closed
}

// GetReadCallCount returns the number of times Read was called
func (m *mockReadWriteCloser) GetReadCallCount() int {
	m.Lock("GetReadCallCount")
	defer m.Unlock()
	return m.readCallCount
}

// GetWriteCallCount returns the number of times Write was called
func (m *mockReadWriteCloser) GetWriteCallCount() int {
	m.Lock("GetWriteCallCount")
	defer m.Unlock()
	return m.writeCallCount
}

// GetCloseCallCount returns the number of times Close was called
func (m *mockReadWriteCloser) GetCloseCallCount() int {
	m.Lock("GetCloseCallCount")
	defer m.Unlock()
	return m.closeCallCount
}

// Read implements io.Reader
func (m *mockReadWriteCloser) Read(p []byte) (n int, err error) {
	if m.debug {
		fmt.Println("Locking Read lock")
	}
	m.readLock.Lock()
	defer func() {
		if m.debug {
			fmt.Println("Unlocking Read lock")
		}
		m.readLock.Unlock()
	}()

	m.readCallCount++

	if m.readDelay > 0 {
		time.Sleep(m.readDelay)
	}

	if m.onRead != nil {
		return m.onRead(p)
	}

	if m.readError != nil {
		return 0, m.readError
	}

	if m.closed {
		return 0, io.ErrClosedPipe
	}

	if len(m.readData) == 0 {
		return 0, io.EOF
	}

	n = copy(p, m.readData)
	m.readData = m.readData[n:]
	return n, nil
}

// Write implements io.Writer
func (m *mockReadWriteCloser) Write(p []byte) (n int, err error) {
	if m.debug {
		fmt.Println("Locking Write lock")
	}
	m.writeLock.Lock()
	defer func() {
		if m.debug {
			fmt.Println("Unlocking Write lock")
		}
		m.writeLock.Unlock()
	}()

	m.writeCallCount++

	if m.writeDelay > 0 {
		time.Sleep(m.writeDelay)
	}

	if m.onWrite != nil {
		return m.onWrite(p)
	}

	if m.writeError != nil {
		return 0, m.writeError
	}

	if m.closed {
		return 0, io.ErrClosedPipe
	}

	m.writtenData = append(m.writtenData, p...)
	return len(p), nil
}

// Close implements io.Closer
func (m *mockReadWriteCloser) Close() error {
	if m.debug {
		fmt.Println("Locking Close lock")
	}
	m.closeLock.Lock()
	defer func() {
		if m.debug {
			fmt.Println("Unlocking Close lock")
		}
		m.closeLock.Unlock()
	}()
	m.closeCallCount++

	if m.closeDelay > 0 {
		time.Sleep(m.closeDelay)
	}

	if m.onClose != nil {
		return m.onClose()
	}

	if m.closeError != nil {
		return m.closeError
	}

	m.closed = true
	close(m.ClosedChan)
	return nil
}

func (m *mockReadWriteCloser) Unlock() {
	if m.debug {
		fmt.Println("Unlocking mutex from", m.lockedFrom)
	}
	m.lockedFrom = ""
	m.mu.Unlock()
}

func (m *mockReadWriteCloser) Lock(from string) {
	if m.debug {
		fmt.Println("Trying to Lock mutex from", from)
	}
	m.mu.Lock()
	if m.debug {
		fmt.Println("Locked mutex from", from)
	}
	m.lockedFrom = from

}

// mockCloseError implements CloseError for testing
type mockCloseError struct {
	code   int
	reason string
}

func newMockCloseError(code int, reason string) *mockCloseError {
	return &mockCloseError{
		code:   code,
		reason: reason,
	}
}

func (e *mockCloseError) Code() int {
	return e.code
}

func (e *mockCloseError) Reason() string {
	return e.reason
}

func (e *mockCloseError) Error() string {
	return "mock close error"
}

var _ TransportHandler = (*recordingTransportHandler)(nil)

// recordingTransportHandler records handler calls for testing
type recordingTransportHandler struct {
	onOpenCalled    bool
	onMessageCalled bool
	onCloseCalled   bool
	onErrorCalled   bool
	conn            cbio.WriteCloser
	message         []byte
	closeCode       int
	closeReason     string
	err             error
	mu              sync.Mutex
}

func newRecordingTransportHandler() *recordingTransportHandler {
	return &recordingTransportHandler{}
}

func (h *recordingTransportHandler) OnOpen(conn cbio.WriteCloser) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onOpenCalled = true
	h.conn = conn
}

func (h *recordingTransportHandler) OnMessage(message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onMessageCalled = true
	h.message = append([]byte{}, message...) // Make a copy
}

func (h *recordingTransportHandler) OnClose(code int, reason string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onCloseCalled = true
	h.closeCode = code
	h.closeReason = reason
}

func (h *recordingTransportHandler) OnError(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onErrorCalled = true
	h.err = err
}

func (h *recordingTransportHandler) WasOnOpenCalled() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.onOpenCalled
}

func (h *recordingTransportHandler) WasOnMessageCalled() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.onMessageCalled
}

func (h *recordingTransportHandler) WasOnCloseCalled() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.onCloseCalled
}

func (h *recordingTransportHandler) WasOnErrorCalled() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.onErrorCalled
}

func (h *recordingTransportHandler) GetConn() cbio.WriteCloser {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.conn
}

func (h *recordingTransportHandler) GetMessage() []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.message
}

func (h *recordingTransportHandler) GetCloseCode() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.closeCode
}

func (h *recordingTransportHandler) GetCloseReason() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.closeReason
}

func (h *recordingTransportHandler) GetError() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.err
}

// testError is a simple error implementation for testing
type testError struct {
	msg string
}

func newTestError(msg string) error {
	return &testError{msg: msg}
}

func (e *testError) Error() string {
	return e.msg
}

// ErrTest is a predefined test error
var ErrTest = errors.New("test error")
