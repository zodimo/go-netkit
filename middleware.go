package netkit

import (
	"io"
)

var _ TransportHandler = (*MiddlewareMux)(nil)

var _ Stack = (*MiddlewareMux)(nil)

func NewStack() Stack {
	return NewMiddlewareMux()
}

func NewMiddlewareMux() *MiddlewareMux {
	return &MiddlewareMux{}
}

// Serves as the router in the http analogy
type Stack interface {
	TransportHandler

	Use(middlewares ...func(TransportHandler) TransportHandler)
	Handler(handler TransportHandler)
}

type Middleware = func(TransportHandler) TransportHandler

type Middlewares []Middleware

func (mx Middlewares) Handler(handler TransportHandler) TransportHandler {
	for _, middleware := range mx {
		handler = middleware(handler)
	}
	return handler
}

type MiddlewareMux struct {
	handler     TransportHandler
	middlewares []Middleware
}

func (mx *MiddlewareMux) getChainedHandler() TransportHandler {
	return Chain(mx.middlewares...).Handler(mx.handler)
}

func (mx *MiddlewareMux) Use(middlewares ...Middleware) {
	if mx.handler != nil {
		panic("netkit: all middlewares must be defined before connecting on a mux")
	}
	mx.middlewares = append(mx.middlewares, middlewares...)
}

func (mx *MiddlewareMux) OnOpen(conn io.WriteCloser) {
	mx.getChainedHandler().OnOpen(conn)
}

func (mx *MiddlewareMux) OnMessage(message []byte) {
	mx.getChainedHandler().OnMessage(message)
}

func (mx *MiddlewareMux) OnClose(code int, reason string) {
	mx.getChainedHandler().OnClose(code, reason)
}

func (mx *MiddlewareMux) OnError(err error) {
	mx.getChainedHandler().OnError(err)
}

func (mx *MiddlewareMux) Handler(handler TransportHandler) {
	mx.handler = handler
}
