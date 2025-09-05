package net

import (
	"context"
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
	h.onOpen(conn)
}

func (h *transportHandler) OnMessage(message []byte) {
	h.onMessage(message)
}

func (h *transportHandler) OnClose(code int, reason string) {
	h.onClose(code, reason)
}

func (h *transportHandler) OnError(err error) {
	h.onError(err)
}

type RwcConfig struct {
	Timeout      time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}
type ConfigOption func(config *RwcConfig)

func DefaultRwcConfig() *RwcConfig {
	return &RwcConfig{
		Timeout:      10 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
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
			var msg []byte
			// this is blocking until timeout or close
			_, err := a.rwc.Read(msg)
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
			handler.OnMessage(msg)
		}
	}

}

var _ io.Closer = (*activeTransportHandler)(nil)

type activeTransportHandler struct {
	actor  *transportActor
	cancel context.CancelFunc
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
		handler.OnOpen(writeCloser)
		go actor.start(handler)
		return &activeTransportHandler{
			actor:  actor,
			cancel: cancel,
		}
	}
}

func (h *activeTransportHandler) Close() error {
	// close the context to stop the reader loop
	h.cancel() // will cancel actor context
	// close the connection, which will propagate the close to the client
	return h.actor.Close()
}
