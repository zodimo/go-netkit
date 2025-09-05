package net

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

type OnOpenHandler func(conn io.WriteCloser)
type OnMessageHandler func(message []byte)
type OnCloseHandler func(code int, reason string)
type OnErrorHandler func(err error)

type TransportHandler interface {
	OnOpen(conn io.WriteCloser)
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

type transportHandlerFunc func(handler TransportHandler) io.Closer

func NewTransportHandler(onOpen OnOpenHandler, onMessage OnMessageHandler, onClose OnCloseHandler, onError OnErrorHandler) TransportHandler {
	return &transportHandler{
		onOpen:    onOpen,
		onMessage: onMessage,
		onClose:   onClose,
		onError:   onError,
	}
}

func (h *transportHandler) OnOpen(conn io.WriteCloser) {
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
}
type ConfigOption func(config *RwcConfig)

func WithReaderBufferSize(size int) ConfigOption {
	return func(config *RwcConfig) {
		config.ReaderBufferSize = size
	}
}

func DefaultRwcConfig() *RwcConfig {
	return &RwcConfig{
		ReaderBufferSize: 4096,
	}
}

var _ io.WriteCloser = (*transportActor)(nil)

type transportActor struct {
	ctx       context.Context
	cancel    context.CancelFunc
	rwc       io.ReadWriteCloser
	config    *RwcConfig
	closeSent bool
	mutex     sync.Mutex
	closed    bool
}

func (a *transportActor) Write(p []byte) (n int, err error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.closed {
		return 0, io.ErrClosedPipe
	}
	return a.rwc.Write(p)
}

func (a *transportActor) Close() error {
	a.cancel()
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if !a.closed {
		a.closed = true
		return a.rwc.Close()
	}
	// already closed
	return nil
}

func (a *transportActor) start(handler TransportHandler) {
	defer a.rwc.Close()

	for {
		select {
		case <-a.ctx.Done():
			// not currently reading and should send the close if not sent yet
			a.mutex.Lock()
			if !a.closeSent {
				handler.OnClose(0, "context canceled")
				a.closeSent = true
			}
			a.mutex.Unlock()
			return
		default:
			// Allocate a buffer with reasonable size
			buf := make([]byte, a.config.ReaderBufferSize)
			// this is blocking until timeout or close
			n, err := a.rwc.Read(buf)
			if err != nil {
				if closedErr := AsCloseError(err); closedErr != nil {
					a.mutex.Lock()
					if !a.closeSent {
						handler.OnClose(closedErr.Code(), closedErr.Reason())
						a.closeSent = true
					}
					a.mutex.Unlock()
					return
				}
				handler.OnError(err)
				return
			}
			// Only pass the actual data that was read
			handler.OnMessage(buf[:n])
		}
	}

}

var _ io.Closer = (*activeTransportHandler)(nil)

type activeTransportHandler struct {
	actor  *transportActor
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newTransportActor(ctx context.Context, rwc io.ReadWriteCloser, config *RwcConfig) *transportActor {
	ctx, cancel := context.WithCancel(ctx)
	return &transportActor{
		ctx:    ctx,
		cancel: cancel,
		rwc:    rwc,
		config: config,
	}
}

func FromReaderWriterCloser(ctx context.Context, rwc io.ReadWriteCloser, options ...ConfigOption) transportHandlerFunc {
	config := DefaultRwcConfig()
	for _, option := range options {
		option(config)
	}
	ctx, cancel := context.WithCancel(ctx)
	actor := newTransportActor(ctx, rwc, config)

	//  close on actor should be the same as calling close on the activeTransportHandler
	writeCloser := actor

	return func(handler TransportHandler) io.Closer {
		activeHandler := &activeTransportHandler{
			actor:  actor,
			cancel: cancel,
		}

		handler.OnOpen(writeCloser)

		// Add to waitgroup before starting goroutine
		activeHandler.wg.Add(1)
		go func() {
			defer activeHandler.wg.Done()
			actor.start(handler)
		}()

		return activeHandler
	}
}

func (h *activeTransportHandler) Close() (err error) {
	// close the context to stop the reader loop
	h.cancel() // will cancel actor context

	// close the connection, which will propagate the close to the client
	err = h.actor.Close()

	// Create a channel to signal completion
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
		// Timeout occurred, log this situation
		// In production code, you might want to log this situation
		if err == nil {
			err = fmt.Errorf("timeout waiting for goroutine to finish")
		}
		err = fmt.Errorf("timeout waiting for goroutine to finish: %w", err)
	}

	return
}
