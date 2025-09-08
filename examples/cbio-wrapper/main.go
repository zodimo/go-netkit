package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/zodimo/go-netkit/cbio"
)

// simpleReadWriteCloser is a simple implementation for demonstration
type simpleReadWriteCloser struct {
	*bytes.Buffer
	closed bool
}

func (s *simpleReadWriteCloser) Close() error {
	s.closed = true
	fmt.Println("Connection closed")
	return nil
}

func main() {
	fmt.Println("=== CBIO Wrapper Example ===")

	// Create a simple io.ReadWriteCloser for demonstration
	buffer := bytes.NewBufferString("Hello from standard I/O!")
	conn := &simpleReadWriteCloser{Buffer: buffer}

	// Wrap it with cbio
	cbioConn := cbio.WrapReadWriteCloser(conn)

	fmt.Println("\n1. Reading data with callback-style I/O:")

	// Create read handler
	readHandler := cbio.NewReaderHandler(
		func(data []byte) {
			fmt.Printf("✅ Successfully read %d bytes: %q\n", len(data), string(data))
		},
		func(err error) {
			if err == io.EOF {
				fmt.Println("📄 Reached end of data")
			} else {
				fmt.Printf("❌ Read error: %v\n", err)
			}
		},
	)

	// Perform async read with timeout
	readCtx, err := cbioConn.Read(readHandler, cbio.WithReadTimeout(2*time.Second))
	if err != nil {
		log.Fatalf("Failed to start read: %v", err)
	}

	// Wait for read completion
	<-readCtx.Done()

	fmt.Println("\n2. Writing data with callback-style I/O:")

	// Create write handler
	writeHandler := cbio.NewWriterHandler(
		func(n int) {
			fmt.Printf("✅ Successfully wrote %d bytes\n", n)
		},
		func(err error) {
			fmt.Printf("❌ Write error: %v\n", err)
		},
	)

	// Perform async write
	writeData := []byte("Response from cbio wrapper!")
	writeCtx, err := cbioConn.Write(writeData, writeHandler)
	if err != nil {
		log.Fatalf("Failed to start write: %v", err)
	}

	// Wait for write completion
	<-writeCtx.Done()

	fmt.Println("\n3. Reading the written data:")

	// Read again to see what we wrote
	readHandler2 := cbio.NewReaderHandler(
		func(data []byte) {
			fmt.Printf("✅ Read written data: %q\n", string(data))
		},
		func(err error) {
			if err == io.EOF {
				fmt.Println("📄 No more data to read")
			} else {
				fmt.Printf("❌ Read error: %v\n", err)
			}
		},
	)

	readCtx2, err := cbioConn.Read(readHandler2)
	if err != nil {
		log.Fatalf("Failed to start second read: %v", err)
	}

	<-readCtx2.Done()

	fmt.Println("\n4. Demonstrating timeout:")

	// Create a mock that will simulate a slow read by not having data immediately available
	slowBuffer := bytes.NewBuffer(nil) // Empty buffer to simulate no data
	slowConn := &simpleReadWriteCloser{Buffer: slowBuffer}
	slowCbioConn := cbio.WrapReadWriteCloser(slowConn)

	timeoutHandler := cbio.NewReaderHandler(
		func(data []byte) {
			fmt.Printf("❌ Should not receive data on timeout: %q\n", string(data))
		},
		func(err error) {
			fmt.Printf("✅ Read timed out as expected: %v\n", err)
		},
	)

	timeoutCtx, err := slowCbioConn.Read(timeoutHandler, cbio.WithReadTimeout(100*time.Millisecond))
	if err != nil {
		log.Fatalf("Failed to start timeout read: %v", err)
	}

	fmt.Println("⏳ Waiting for timeout...")
	<-timeoutCtx.Done()

	fmt.Println("\n5. Demonstrating manual cancellation:")

	// Create another slow connection for cancellation demo
	slowBuffer2 := bytes.NewBuffer(nil)
	slowConn2 := &simpleReadWriteCloser{Buffer: slowBuffer2}
	slowCbioConn2 := cbio.WrapReadWriteCloser(slowConn2)

	cancelHandler := cbio.NewReaderHandler(
		func(data []byte) {
			fmt.Printf("❌ Should not receive data on cancellation: %q\n", string(data))
		},
		func(err error) {
			fmt.Printf("✅ Read was cancelled as expected: %v\n", err)
		},
	)

	cancelCtx, err := slowCbioConn2.Read(cancelHandler)
	if err != nil {
		log.Fatalf("Failed to start cancellable read: %v", err)
	}

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		fmt.Println("🚫 Manually cancelling read operation...")
		cancelCtx.Cancel()
	}()

	<-cancelCtx.Done()

	fmt.Println("\n6. Closing the connection:")

	// Close with timeout
	err = cbioConn.Close(cbio.WithCloseTimeout(1 * time.Second))
	if err != nil {
		fmt.Printf("❌ Close error: %v\n", err)
	} else {
		fmt.Println("✅ Connection closed successfully")
	}

	fmt.Println("\n=== Example completed ===")
}
