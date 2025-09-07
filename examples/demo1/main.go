package main

import (
	"context"
	"fmt"
	"io"
	"net"

	netkit "github.com/zodimo/go-netkit"
)

func main() {

	// Create base handler
	baseHandler := netkit.NewTransportHandler(
		func(conn io.WriteCloser) { fmt.Println("Connection opened") },
		func(message []byte) { fmt.Printf("Received: %s\n", string(message)) },
		func(code int, reason string) { fmt.Printf("Closed: %d %s\n", code, reason) },
		func(err error) { fmt.Printf("Error: %v\n", err) },
	)

	// Create logging middleware
	loggingMiddleware := func(handler netkit.TransportHandler) netkit.TransportHandler {
		return netkit.NewTransportHandler(
			func(conn io.WriteCloser) {
				fmt.Println("[LOG] Connection opened")
				handler.OnOpen(conn)
			},
			func(message []byte) {
				fmt.Printf("[LOG] Message received: %d bytes\n", len(message))
				handler.OnMessage(message)
			},
			func(code int, reason string) {
				fmt.Printf("[LOG] Connection closed: %d %s\n", code, reason)
				handler.OnClose(code, reason)
			},
			func(err error) {
				fmt.Printf("[LOG] Error: %v\n", err)
				handler.OnError(err)
			},
		)
	}

	stack := netkit.NewStack()
	stack.Use(loggingMiddleware)
	stack.Handler(baseHandler)

	// Use the handler with middleware
	listener, _ := net.Listen("tcp", ":8080")
	fmt.Println("Listening on http://localhost:8080")
	conn, _ := listener.Accept()
	fmt.Println("Accepted connection")

	ctx := context.Background()
	receiverFunc := netkit.FromReaderWriterCloser(ctx, conn)
	closer := receiverFunc(stack)
	defer closer.Close()

	// Your application logic here...
}
