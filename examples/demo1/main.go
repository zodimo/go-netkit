package main

import (
	"context"
	"fmt"
	"net"
	"time"

	netkit "github.com/zodimo/go-netkit"
	"github.com/zodimo/go-netkit/cbio"
)

func main() {
	fmt.Println("=== Go-NetKit CBIO Demo ===")

	// Create base handler with callback-style I/O demonstrations
	baseHandler := netkit.NewTransportHandler(
		// OnOpen - demonstrates cbio.WriteCloser usage
		func(conn cbio.WriteCloser) {
			fmt.Println("Connection opened - demonstrating callback-style writes")

			// Demonstrate callback-style write with success/error handling
			writeHandler := cbio.NewWriterHandler(
				func(n int) {
					fmt.Printf("✅ Successfully wrote %d bytes\n", n)
				},
				func(err error) {
					fmt.Printf("❌ Write failed: %v\n", err)
				},
			)

			// Write with timeout configuration
			ctx, err := conn.Write(
				[]byte("Hello from callback-style I/O!"),
				writeHandler,
				cbio.WithWriteTimeout(5*time.Second),
			)

			if err != nil {
				fmt.Printf("Failed to start write: %v\n", err)
				return
			}

			// Demonstrate cancellation (optional)
			go func() {
				time.Sleep(100 * time.Millisecond)
				// Uncomment to test cancellation:
				// fmt.Println("Cancelling write operation...")
				// ctx.Cancel()
			}()

			// Wait for write completion
			<-ctx.Done()

			// Demonstrate close with timeout
			err = conn.Close(cbio.WithCloseTimeout(2 * time.Second))
			if err != nil {
				fmt.Printf("Close failed: %v\n", err)
			} else {
				fmt.Println("Connection closed successfully")
			}
		},
		// OnMessage - handles incoming messages
		func(message []byte) {
			fmt.Printf("📨 Received: %s\n", string(message))
		},
		// OnClose - handles connection closure
		func(code int, reason string) {
			fmt.Printf("🔌 Connection closed: %d %s\n", code, reason)
		},
		// OnError - handles errors
		func(err error) {
			fmt.Printf("⚠️  Error: %v\n", err)
		},
	)

	// Create logging middleware that also uses cbio interfaces
	loggingMiddleware := func(handler netkit.TransportHandler) netkit.TransportHandler {
		return netkit.NewTransportHandler(
			func(conn cbio.WriteCloser) {
				fmt.Println("[LOG] Connection opened with cbio.WriteCloser")

				// Demonstrate middleware can also use callback-style operations
				writeHandler := cbio.NewWriterHandler(
					func(n int) {
						fmt.Printf("[LOG] Middleware wrote %d bytes\n", n)
						// Pass to next handler after successful write
						handler.OnOpen(conn)
					},
					func(err error) {
						fmt.Printf("[LOG] Middleware write failed: %v\n", err)
						// Still pass to handler even if middleware write fails
						handler.OnOpen(conn)
					},
				)

				// Middleware writes a log message first
				ctx, err := conn.Write(
					[]byte("[LOG] Middleware initialized\n"),
					writeHandler,
					cbio.WithWriteTimeout(3*time.Second),
				)

				if err != nil {
					fmt.Printf("[LOG] Failed to start middleware write: %v\n", err)
					handler.OnOpen(conn)
					return
				}

				// Don't block the middleware
				go func() {
					<-ctx.Done()
				}()
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

	// Build the middleware stack
	stack := netkit.NewStack()
	stack.Use(loggingMiddleware)
	stack.Handler(baseHandler)

	// Set up a simple TCP server for demonstration
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
		return
	}
	defer listener.Close()

	fmt.Println("🚀 Server listening on http://localhost:8080")
	fmt.Println("📋 This demo shows:")
	fmt.Println("   • cbio.WriteCloser usage in handlers")
	fmt.Println("   • Callback-style write operations")
	fmt.Println("   • Error handling with success/error callbacks")
	fmt.Println("   • Configuration options (timeouts)")
	fmt.Println("   • Cancellation support")
	fmt.Println("   • Proper resource cleanup")
	fmt.Println("\n💡 Connect with: telnet localhost 8080")
	fmt.Println("   Or use netcat: nc localhost 8080")

	// Accept a single connection for demo purposes
	conn, err := listener.Accept()
	if err != nil {
		fmt.Printf("Failed to accept connection: %v\n", err)
		return
	}
	fmt.Println("✅ Accepted connection from", conn.RemoteAddr())

	// Create transport receiver with callback-style I/O
	ctx := context.Background()
	receiver := netkit.FromReaderWriteCloser(ctx, conn)
	closer := receiver.Receive(stack)
	defer closer.Close()

	// Keep the demo running for a bit to show the callback operations
	fmt.Println("\n⏳ Demo running for 10 seconds...")
	time.Sleep(10 * time.Second)

	fmt.Println("\n🎯 Demo completed successfully!")
	fmt.Println("   All operations used callback-style I/O patterns")
}
