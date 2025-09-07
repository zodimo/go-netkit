package middlewares

import (
	"fmt"
	"io"

	"github.com/zodimo/go-netkit"
)

func NewLoggingMiddleware(prefix string) func(handler netkit.TransportHandler) netkit.TransportHandler {
	return func(handler netkit.TransportHandler) netkit.TransportHandler {
		return netkit.NewTransportHandler(
			func(conn io.WriteCloser) {
				fmt.Printf("[LOGGER %s] Connection opened \n", prefix)
				handler.OnOpen(conn)
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
