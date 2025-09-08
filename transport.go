package netkit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zodimo/go-netkit/cbio"
)

type OnOpenHandler func(conn cbio.WriteCloser)
type OnMessageHandler func(message []byte)
type OnCloseHandler func(code int, reason string)
type OnErrorHandler func(err error)

type TransportHandler interface {
	OnOpen(conn cbio.WriteCloser)
	OnMessage(message []byte)
	OnClose(code int, reason string)
	OnError(err error)
}

var _ TransportHandler = &transportHandler{}

type transportHandler struct {
	onOpen    OnOpenHandler
	onMessage OnMessageHandler
	onClose   OnCloseHandler
	onError   OnErrorHandler
	mu        sync.Mutex
}

type TransportReceiverFunc func(handler TransportHandler) io.Closer

func (f TransportReceiverFunc) Receive(handler TransportHandler) io.Closer {
	return f(handler)
}

type TransportReceiver interface {
	Receive(handler TransportHandler) io.Closer
}

func NewTransportHandler(onOpen OnOpenHandler, onMessage OnMessageHandler, onClose OnCloseHandler, onError OnErrorHandler) TransportHandler {
	return &transportHandler{
		onOpen:    onOpen,
		onMessage: onMessage,
		onClose:   onClose,
		onError:   onError,
	}
}

func (h *transportHandler) OnOpen(conn cbio.WriteCloser) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onOpen(conn)
}

func (h *transportHandler) OnMessage(message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onMessage(message)
}

func (h *transportHandler) OnClose(code int, reason string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onClose(code, reason)
}

func (h *transportHandler) OnError(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onError(err)
}

type RwcConfig struct {
	ReaderBufferSize int
	ReaderTimeout    time.Duration
}
type ConfigOption func(config *RwcConfig)

func WithReaderBufferSize(size int) ConfigOption {
	return func(config *RwcConfig) {
		config.ReaderBufferSize = size
	}
}

func WithReaderTimeout(timeout time.Duration) ConfigOption {
	return func(config *RwcConfig) {
		config.ReaderTimeout = timeout
	}
}

func DefaultRwcConfig() *RwcConfig {
	return &RwcConfig{
		ReaderBufferSize: 4096,
		ReaderTimeout:    10 * time.Second,
	}
}

var _ cbio.WriteCloser = (*transportActor)(nil)

type transportActor struct {
	ctx         context.Context
	cancel      context.CancelFunc
	rwc         io.ReadWriteCloser
	config      *RwcConfig
	closeSent   atomic.Bool
	mutex       sync.Mutex
	closed      atomic.Bool
	onCloseFunc OnCloseHandler
	done        chan struct{}
}

func (a *transportActor) setOnCloseFunc(onCloseFunc OnCloseHandler) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.onCloseFunc = onCloseFunc
}

func (a *transportActor) onClose(code int, reason string) {
	// Only send the close event once
	if !a.closeSent.Swap(true) {
		a.mutex.Lock()
		closeFunc := a.onCloseFunc
		a.mutex.Unlock()
		closeFunc(code, reason)
	}
}

func (a *transportActor) doClose() error {
	// Only close once
	if !a.closed.Swap(true) {
		// Send close event if not already sent
		if !a.closeSent.Load() {
			a.onClose(0, "connection closed")
		}

		// Close the underlying connection
		err := a.rwc.Close()

		// Signal that we're done
		close(a.done)

		return err
	}
	return nil
}

func (a *transportActor) Write(p []byte, handler cbio.WriterHandler, options ...cbio.WriterOption) (cbio.CbContext, error) {
	writerConfig := cbio.DefaultWriterConfig()
	for _, option := range options {
		option(writerConfig)
	}

	writeCtx, writeCancel := context.WithCancel(a.ctx)
	cbCtx := cbio.NewCbContext(cbio.CancelFunc(writeCancel), writeCtx.Done())

	// Check if already closed
	if a.closed.Load() {
		return cbCtx, io.ErrClosedPipe
	}

	// Lock only for the write operation
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Double-check closed state after acquiring lock
	if a.closed.Load() {
		return cbCtx, io.ErrClosedPipe
	}

	go func() {
		<-writeCtx.Done()
		if conn, ok := a.rwc.(interface{ SetWriteDeadline(time.Time) }); ok {
			conn.SetWriteDeadline(time.Now())
		}
	}()

	go func() {
		defer writeCancel()
		if conn, ok := a.rwc.(interface{ SetWriteDeadline(time.Time) }); ok {
			conn.SetWriteDeadline(time.Now().Add(writerConfig.Timeout))
		}
		n, err := a.rwc.Write(p)
		if err != nil {
			handler.OnError(err)
			return
		}
		handler.OnSuccess(n)
	}()

	return cbCtx, nil
}

func (a *transportActor) Close(options ...cbio.CloserOption) error {
	// Cancel context first to signal reader goroutine to stop
	a.cancel()

	// Close the connection and resources
	return a.doClose()
}

func (a *transportActor) start(handler TransportHandler) {
	// Wire up onClose to handler so it gets called exactly once
	// whether from read closed or context canceled
	a.setOnCloseFunc(handler.OnClose)

	// Always ensure we clean up resources
	defer a.doClose()

	for {
		// Check context before each read
		select {
		case <-a.ctx.Done():
			// Context was canceled, send close if not already sent
			a.onClose(-1, "context canceled")
			return
		default:
			// Continue with read operation
		}

		// Allocate a buffer with reasonable size
		buf := make([]byte, a.config.ReaderBufferSize)
		readCtx, readCancel := context.WithCancel(a.ctx)
		go func() {
			<-readCtx.Done()
			if readConn, ok := a.rwc.(interface{ SetReadDeadline(time.Time) }); ok {
				readConn.SetReadDeadline(time.Now())
			}
		}()
		go func() {
			if readConn, ok := a.rwc.(interface{ SetReadDeadline(time.Time) }); ok {
				readConn.SetReadDeadline(time.Now().Add(a.config.ReaderTimeout))
			}
			n, err := a.rwc.Read(buf)
			if err != nil {
				if errors.Is(err, io.EOF) {
					a.onClose(1000, "normal closure")
				} else {
					handler.OnError(err)
				}
				return
			}
			handler.OnMessage(buf[:n])
			readCancel()
		}()

		// Check context again after read completes
		select {
		case <-a.ctx.Done():
			// Context was canceled during read, prioritize this over read result
			a.onClose(-1, "context canceled")
			readCancel()
		case <-readCtx.Done():
			// read done or cancelled
		}

	}
}

var _ io.Closer = (*activeTransportHandler)(nil)

type activeTransportHandler struct {
	actor  *transportActor
	cancel context.CancelFunc
	wg     sync.WaitGroup
	closed atomic.Bool
}

func newTransportActor(ctx context.Context, rwc io.ReadWriteCloser, config *RwcConfig) *transportActor {
	ctx, cancel := context.WithCancel(ctx)
	onCloseFunc := func(code int, reason string) {}
	actor := &transportActor{
		ctx:         ctx,
		cancel:      cancel,
		rwc:         rwc,
		config:      config,
		onCloseFunc: onCloseFunc,
		done:        make(chan struct{}),
	}
	// Initialize atomic values
	actor.closeSent.Store(false)
	actor.closed.Store(false)
	return actor
}

func FromReaderWriteCloser(ctx context.Context, rwc io.ReadWriteCloser, options ...ConfigOption) TransportReceiver {
	// Apply configuration options
	config := DefaultRwcConfig()
	for _, option := range options {
		option(config)
	}

	// Create a derived context that can be canceled independently
	ctx, cancel := context.WithCancel(ctx)

	// Create the transport actor
	actor := newTransportActor(ctx, rwc, config)

	// Return the transport handler function
	return TransportReceiverFunc(func(handler TransportHandler) io.Closer {
		activeHandler := &activeTransportHandler{
			actor:  actor,
			cancel: cancel,
		}
		activeHandler.closed.Store(false)

		// Notify handler of new connection
		handler.OnOpen(actor)

		// Add to waitgroup before starting goroutine
		activeHandler.wg.Add(1)
		go func() {
			defer activeHandler.wg.Done()
			actor.start(handler)
		}()

		return activeHandler
	})
}

func (h *activeTransportHandler) Close() (err error) {
	// Only close once
	if h.closed.Swap(true) {
		return nil
	}

	// First cancel the context to signal the reader loop to stop
	h.cancel()

	// Then close the actor which will close the underlying connection
	err = h.actor.Close()

	// Wait for the actor to finish with timeout
	select {
	case <-h.actor.done:
		// Actor completed normally
	case <-time.After(1 * time.Second):
		// Short timeout for actor to finish
	}

	// Create a channel to signal completion of the goroutine
	done := make(chan struct{})

	// Wait for the goroutine to finish with timeout
	go func() {
		h.wg.Wait()
		close(done)
	}()

	// Wait with timeout (5 seconds should be enough for clean shutdown)
	select {
	case <-done:
		// Goroutine completed normally
	case <-time.After(5 * time.Second):
		// Timeout occurred
		if err == nil {
			err = fmt.Errorf("timeout waiting for goroutine to finish")
		} else {
			err = fmt.Errorf("timeout waiting for goroutine to finish: %w", err)
		}
	}

	return
}
