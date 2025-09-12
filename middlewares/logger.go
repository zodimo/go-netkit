package middlewares

import (
	"fmt"

	"github.com/zodimo/go-netkit"
	"github.com/zodimo/go-netkit/cbio"
)

var _ cbio.WriteCloser = (*logWriteCloser)(nil)

type logWriteCloser struct {
	cbio.WriteCloser
	prefix string
}

func NewLogWriteCloser(writeCloser cbio.WriteCloser, prefix string) cbio.WriteCloser {
	return &logWriteCloser{WriteCloser: writeCloser, prefix: prefix}
}

func (l *logWriteCloser) Write(p []byte, handler cbio.WriterHandler, options ...cbio.WriterOption) (cbio.CbContext, error) {
	fmt.Printf("[LOGGER %s] Writing %d bytes\n", l.prefix, len(p))
	return l.WriteCloser.Write(p, handler, options...)
}

func (l *logWriteCloser) Close(options ...cbio.CloserOption) error {
	fmt.Printf("[LOGGER %s] Closing\n", l.prefix)
	return l.WriteCloser.Close(options...)
}

func NewLoggingMiddleware(prefix string) func(handler netkit.TransportHandler) netkit.TransportHandler {
	return func(handler netkit.TransportHandler) netkit.TransportHandler {
		return netkit.NewTransportHandler(
			func(conn cbio.WriteCloser) {
				fmt.Printf("[LOGGER %s] Connection opened \n", prefix)
				handler.OnOpen(NewLogWriteCloser(conn, prefix))
			},
			func(message []byte) {
				fmt.Printf("[LOGGER %s] Message received: %s\n", prefix, string(message))
				handler.OnMessage(message)
			},
			func(code int, reason string) {
				fmt.Printf("[LOGGER %s] Connection closed: %d %s\n", prefix, code, reason)
				handler.OnClose(code, reason)
			},
			func(err error) {
				fmt.Printf("[LOGGER %s] Error: %v\n", prefix, err)
				handler.OnError(err)
			},
		)
	}
}
