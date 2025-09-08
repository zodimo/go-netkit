package main

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/zodimo/go-netkit/cbio"
)

func main() {
	fmt.Println("CBIO Unwrap Example - Bidirectional Conversion")
	fmt.Println("==============================================")

	// Example 1: Simple unwrapping demonstration
	fmt.Println("\n1. Basic Unwrapping Example:")
	basicUnwrapExample()

	// Example 2: Bidirectional conversion
	fmt.Println("\n2. Bidirectional Conversion Example:")
	bidirectionalExample()

	// Example 3: Integration with standard library
	fmt.Println("\n3. Standard Library Integration Example:")
	standardLibraryExample()
}

// basicUnwrapExample demonstrates basic unwrapping of cbio to io
func basicUnwrapExample() {
	// Create a mock connection (in real use, this would be a network connection)
	mockConn := &mockConnection{
		data: "Hello from cbio!",
	}

	// Wrap standard io to cbio
	cbioConn := cbio.WrapReadWriteCloser(mockConn)
	fmt.Printf("  Created cbio connection from mock: %T\n", cbioConn)

	// Unwrap cbio back to standard io
	ioConn := cbio.UnwrapReadWriteCloser(cbioConn)
	fmt.Printf("  Unwrapped to standard io: %T\n", ioConn)

	// Use with standard io operations
	buffer := make([]byte, 1024)
	n, err := ioConn.Read(buffer)
	if err != nil {
		log.Printf("  Read error: %v", err)
		return
	}
	fmt.Printf("  Read %d bytes: %q\n", n, string(buffer[:n]))

	// Write response
	response := []byte("Response from unwrapped connection")
	n, err = ioConn.Write(response)
	if err != nil {
		log.Printf("  Write error: %v", err)
		return
	}
	fmt.Printf("  Wrote %d bytes\n", n)

	// Close
	err = ioConn.Close()
	if err != nil {
		log.Printf("  Close error: %v", err)
		return
	}
	fmt.Println("  Connection closed successfully")
}

// bidirectionalExample demonstrates converting back and forth
func bidirectionalExample() {
	mockConn := &mockConnection{
		data: "Bidirectional test data",
	}

	fmt.Println("  Starting with standard io.ReadWriteCloser")

	// Step 1: io -> cbio
	cbioConn := cbio.WrapReadWriteCloser(mockConn)
	fmt.Println("  Wrapped to cbio.ReadWriteCloser")

	// Step 2: cbio -> io
	ioConn := cbio.UnwrapReadWriteCloser(cbioConn)
	fmt.Println("  Unwrapped back to io.ReadWriteCloser")

	// Step 3: Use the unwrapped connection
	buffer := make([]byte, 1024)
	n, err := ioConn.Read(buffer)
	if err != nil {
		log.Printf("  Read error: %v", err)
		return
	}
	fmt.Printf("  Successfully read %d bytes: %q\n", n, string(buffer[:n]))

	// Close the connection
	err = ioConn.Close()
	if err != nil {
		log.Printf("  Close error: %v", err)
		return
	}
	fmt.Println("  Connection closed successfully")
}

// standardLibraryExample shows integration with standard library functions
func standardLibraryExample() {
	mockConn := &mockConnection{
		data: "Data for standard library integration",
	}

	// Create cbio connection
	cbioConn := cbio.WrapReadWriteCloser(mockConn)

	// Unwrap for standard library use
	ioConn := cbio.UnwrapReadWriteCloser(cbioConn)

	// Use with io.Copy
	var result strings.Builder
	n, err := io.Copy(&result, ioConn)
	if err != nil {
		log.Printf("  io.Copy error: %v", err)
		return
	}
	fmt.Printf("  io.Copy transferred %d bytes: %q\n", n, result.String())

	// Close
	err = ioConn.Close()
	if err != nil {
		log.Printf("  Close error: %v", err)
		return
	}
	fmt.Println("  Standard library integration successful")
}

// mockConnection implements io.ReadWriteCloser for demonstration
type mockConnection struct {
	data     string
	position int
	written  []byte
	closed   bool
}

func (m *mockConnection) Read(p []byte) (n int, err error) {
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	if m.position >= len(m.data) {
		return 0, io.EOF
	}

	remaining := m.data[m.position:]
	n = copy(p, remaining)
	m.position += n
	return n, nil
}

func (m *mockConnection) Write(p []byte) (n int, err error) {
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	m.written = append(m.written, p...)
	return len(p), nil
}

func (m *mockConnection) Close() error {
	m.closed = true
	return nil
}
