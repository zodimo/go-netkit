package middlewares

import (
	"fmt"

	"github.com/zodimo/go-netkit"
	"github.com/zodimo/go-netkit/cbio"
)

func NewLoggingMiddleware(prefix string) func(handler netkit.TransportHandler) netkit.TransportHandler {
	return func(handler netkit.TransportHandler) netkit.TransportHandler {
		return netkit.NewTransportHandler(
			func(conn cbio.WriteCloser) {
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
