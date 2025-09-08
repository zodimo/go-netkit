# CBIO - Callback-Style I/O Interfaces

The `cbio` package provides callback-style alternatives to Go's standard I/O interfaces, enabling asynchronous operations with proper error handling, cancellation support, and configurable options.

## Overview

Traditional Go I/O operations are synchronous and block until completion. The `cbio` package transforms these operations into asynchronous, callback-driven patterns that provide:

- **Non-blocking operations**: I/O operations return immediately with a `CbContext`
- **Structured error handling**: Separate immediate errors from operation errors
- **Cancellation support**: All operations can be cancelled mid-flight via `CbContext`
- **Configurable behavior**: Timeouts, buffer sizes, and other options per operation
- **Composable interfaces**: ReadCloser, WriteCloser, ReadWriteCloser combinations

## Core Interfaces

### Reader

```go
type Reader interface {
    Read(handler ReaderHandler, options ...ReaderOption) (CbContext, error)
}

type ReaderHandler interface {
    OnSuccess(p []byte)  // Called when data is successfully read
    OnError(err error)   // Called when read operation fails
}
```

**Options:**
- `WithReadTimeout(duration)`: Set read operation timeout
- `WithReaderBufferSize(size)`: Configure read buffer size

### Writer

```go
type Writer interface {
    Write(p []byte, handler WriterHandler, options ...WriterOption) (CbContext, error)
}

type WriterHandler interface {
    OnSuccess(n int)     // Called when data is successfully written (n = bytes written)
    OnError(err error)   // Called when write operation fails
}
```

**Options:**
- `WithWriteTimeout(duration)`: Set write operation timeout

### Closer

```go
type Closer interface {
    Close(options ...CloserOption) error
}
```

**Options:**
- `WithCloseTimeout(duration)`: Set close operation timeout

### Composite Interfaces

```go
type ReadCloser interface {
    Reader
    Closer
}

type WriteCloser interface {
    Writer
    Closer
}

type ReadWriteCloser interface {
    Reader
    Writer
    Closer
}
```

## Context and Cancellation

The `CbContext` interface provides cancellation and completion signaling:

```go
type CbContext interface {
    Done() <-chan struct{}  // Channel that closes when operation completes or is cancelled
    Cancel()                // Cancel the operation
}
```

## Usage Examples

### Basic Reader Usage

```go
import (
    "fmt"
    "time"
    "github.com/zodimo/go-netkit/cbio"
)

func readExample(reader cbio.Reader) {
    // Create handler for read callbacks
    readHandler := cbio.NewReaderHandler(
        func(data []byte) {
            fmt.Printf("Read %d bytes: %s\n", len(data), string(data))
        },
        func(err error) {
            fmt.Printf("Read failed: %v\n", err)
        },
    )
    
    // Start read operation with timeout
    ctx, err := reader.Read(
        readHandler,
        cbio.WithReadTimeout(5*time.Second),
        cbio.WithReaderBufferSize(8192),
    )
    
    if err != nil {
        fmt.Printf("Failed to start read: %v\n", err)
        return
    }
    
    // Can cancel the read operation
    // ctx.Cancel()
    
    // Wait for completion
    <-ctx.Done()
}
```

### Basic Writer Usage

```go
func writeExample(writer cbio.Writer) {
    data := []byte("Hello, callback world!")
    
    writeHandler := cbio.NewWriterHandler(
        func(n int) {
            fmt.Printf("Successfully wrote %d bytes\n", n)
        },
        func(err error) {
            fmt.Printf("Write failed: %v\n", err)
        },
    )
    
    ctx, err := writer.Write(
        data,
        writeHandler,
        cbio.WithWriteTimeout(3*time.Second),
    )
    
    if err != nil {
        fmt.Printf("Failed to start write: %v\n", err)
        return
    }
    
    // Wait for operation completion
    <-ctx.Done()
}
```

### ReadWriteCloser Usage

```go
func readWriteExample(rwc cbio.ReadWriteCloser) {
    // Write some data
    writeHandler := cbio.NewWriterHandler(
        func(n int) {
            fmt.Printf("Wrote %d bytes, now starting read...\n", n)
            
            // After successful write, start reading
            readHandler := cbio.NewReaderHandler(
                func(data []byte) {
                    fmt.Printf("Read response: %s\n", string(data))
                    
                    // Close when done
                    err := rwc.Close(cbio.WithCloseTimeout(1*time.Second))
                    if err != nil {
                        fmt.Printf("Close failed: %v\n", err)
                    }
                },
                func(err error) {
                    fmt.Printf("Read failed: %v\n", err)
                },
            )
            
            readCtx, err := rwc.Read(readHandler)
            if err != nil {
                fmt.Printf("Failed to start read: %v\n", err)
                return
            }
            _ = readCtx
        },
        func(err error) {
            fmt.Printf("Write failed: %v\n", err)
        },
    )
    
    writeCtx, err := rwc.Write([]byte("ping"), writeHandler)
    if err != nil {
        fmt.Printf("Failed to start write: %v\n", err)
        return
    }
    
    // Wait for write completion
    <-writeCtx.Done()
}
```

## Error Handling Patterns

### Immediate vs. Operation Errors

```go
ctx, err := writer.Write(data, handler)
if err != nil {
    // Immediate error - operation couldn't be started
    // Examples: invalid parameters, resource unavailable
    fmt.Printf("Immediate error: %v\n", err)
    return
}

// Operation started successfully
// Any runtime errors will be delivered to handler.OnError()
// Wait for completion or cancellation
<-ctx.Done()
```

### Timeout Handling

```go
writeHandler := cbio.NewWriterHandler(
    func(n int) {
        fmt.Println("Write completed successfully")
    },
    func(err error) {
        if err == context.DeadlineExceeded {
            fmt.Println("Write operation timed out")
        } else {
            fmt.Printf("Write failed: %v\n", err)
        }
    },
)

ctx, err := writer.Write(
    data,
    writeHandler,
    cbio.WithWriteTimeout(5*time.Second), // Will timeout after 5 seconds
)
```

## Cancellation Patterns

### Immediate Cancellation

```go
ctx, err := reader.Read(handler)
if err != nil {
    return
}

// Cancel immediately
ctx.Cancel()
```

### Conditional Cancellation

```go
ctx, err := reader.Read(handler)
if err != nil {
    return
}

// Cancel after some condition
go func() {
    time.Sleep(2*time.Second)
    fmt.Println("Cancelling read operation...")
    ctx.Cancel()
}()

// Wait for completion or cancellation
<-ctx.Done()
```

## Configuration Options

### Default Configurations

```go
// Reader defaults
readerConfig := cbio.DefaultReaderConfig()
// Timeout: 0 (no timeout)
// BufferSize: 4096

// Writer defaults
writerConfig := cbio.DefaultWriterConfig()
// Timeout: 0 (no timeout)

// Closer defaults - no explicit default config function
// Timeout: 0 (no timeout)
```

### Custom Configurations

```go
// Read with custom configuration
ctx, err := reader.Read(
    handler,
    cbio.WithReadTimeout(10*time.Second),
    cbio.WithReaderBufferSize(16384),
)

// Write with timeout
ctx, err := writer.Write(
    data,
    handler, 
    cbio.WithWriteTimeout(5*time.Second),
)

// Close with timeout
err := closer.Close(cbio.WithCloseTimeout(2*time.Second))
```

## I/O Wrapper Functions

### WrapReadWriteCloser

The `WrapReadWriteCloser` function provides seamless interoperability between standard Go I/O and callback-style cbio operations:

```go
func WrapReadWriteCloser(rwc io.ReadWriteCloser) ReadWriteCloser
```

This function wraps a standard `io.ReadWriteCloser` into a `cbio.ReadWriteCloser`, enabling:
- Non-blocking Read/Write operations using goroutines
- Proper callback-based success/error handling
- Cancellation support through CbContext
- Configurable timeouts and options
- Thread-safe concurrent operations with full-duplex support
- True concurrent read/write operations without blocking each other

#### Example Usage

```go
import (
    "net"
    "time"
    "github.com/zodimo/go-netkit/cbio"
)

// Wrap a standard network connection
conn, err := net.Dial("tcp", "example.com:80")
if err != nil {
    return err
}

cbioConn := cbio.WrapReadWriteCloser(conn)

// Use with callback-style operations
readHandler := cbio.NewReaderHandler(
    func(data []byte) {
        fmt.Printf("Received: %s\n", string(data))
    },
    func(err error) {
        fmt.Printf("Read error: %v\n", err)
    },
)

ctx, err := cbioConn.Read(readHandler, cbio.WithReadTimeout(5*time.Second))
if err != nil {
    return err
}

// Wait for completion or cancel if needed
<-ctx.Done()

// Close when done
err = cbioConn.Close(cbio.WithCloseTimeout(2*time.Second))
```

#### Migration Pattern

The wrapper is particularly useful for migrating existing code:

```go
// Before: Standard I/O
func handleConnection(conn io.ReadWriteCloser) {
    buffer := make([]byte, 1024)
    n, err := conn.Read(buffer)
    if err != nil {
        log.Printf("Read error: %v", err)
        return
    }
    
    _, err = conn.Write([]byte("response"))
    if err != nil {
        log.Printf("Write error: %v", err)
    }
}

// After: Callback-style with wrapper
func handleConnection(conn io.ReadWriteCloser) {
    cbioConn := cbio.WrapReadWriteCloser(conn)
    
    readHandler := cbio.NewReaderHandler(
        func(data []byte) {
            // Handle successful read
            writeHandler := cbio.NewWriterHandler(
                func(n int) {
                    log.Printf("Wrote %d bytes", n)
                },
                func(err error) {
                    log.Printf("Write error: %v", err)
                },
            )
            
            ctx, err := cbioConn.Write([]byte("response"), writeHandler)
            if err != nil {
                log.Printf("Failed to start write: %v", err)
                return
            }
            <-ctx.Done()
        },
        func(err error) {
            log.Printf("Read error: %v", err)
        },
    )
    
    ctx, err := cbioConn.Read(readHandler, cbio.WithReadTimeout(5*time.Second))
    if err != nil {
        log.Printf("Failed to start read: %v", err)
        return
    }
    <-ctx.Done()
}
```

### UnwrapReadWriteCloser

The `UnwrapReadWriteCloser` function provides the reverse interoperability, converting callback-style cbio operations back to standard Go I/O:

```go
func UnwrapReadWriteCloser(rwc ReadWriteCloser) io.ReadWriteCloser
```

This function unwraps a `cbio.ReadWriteCloser` into a standard `io.ReadWriteCloser`, enabling:
- Synchronous blocking Read/Write operations using channels to wait for callbacks
- Standard io interface compliance with `(n int, err error)` return patterns
- Thread-safe concurrent operations with full-duplex support
- True concurrent read/write operations without blocking each other
- Proper error handling and resource cleanup
- Integration with standard library code and third-party packages

#### Example Usage

```go
import (
    "io"
    "fmt"
    "github.com/zodimo/go-netkit/cbio"
)

// Assuming you have a cbio.ReadWriteCloser instance
var cbioConn cbio.ReadWriteCloser = getSomeCbioConnection()

// Unwrap to standard io.ReadWriteCloser
ioConn := cbio.UnwrapReadWriteCloser(cbioConn)

// Use with standard I/O operations
buffer := make([]byte, 1024)
n, err := ioConn.Read(buffer)
if err != nil {
    return err
}
fmt.Printf("Read %d bytes: %s\n", n, string(buffer[:n]))

// Write data
data := []byte("Hello, world!")
n, err = ioConn.Write(data)
if err != nil {
    return err
}
fmt.Printf("Wrote %d bytes\n", n)

// Close the connection
err = ioConn.Close()
if err != nil {
    return err
}
```

#### Integration Pattern

The unwrapper is particularly useful for integrating cbio instances with standard library code:

```go
// Use cbio instances with standard library functions
func processWithStandardLib(cbioConn cbio.ReadWriteCloser) error {
    // Unwrap to standard io interface
    ioConn := cbio.UnwrapReadWriteCloser(cbioConn)
    
    // Use with io.Copy
    var buf bytes.Buffer
    n, err := io.Copy(&buf, ioConn)
    if err != nil {
        return err
    }
    
    fmt.Printf("Copied %d bytes: %s\n", n, buf.String())
    return nil
}

// Use with third-party libraries expecting io.ReadWriteCloser
func useWithThirdParty(cbioConn cbio.ReadWriteCloser) error {
    ioConn := cbio.UnwrapReadWriteCloser(cbioConn)
    
    // Pass to any function expecting io.ReadWriteCloser
    return someThirdPartyFunction(ioConn)
}
```

#### Bidirectional Conversion

Both wrapper functions can be used together for complete interoperability:

```go
// Start with standard io
conn, err := net.Dial("tcp", "example.com:80")
if err != nil {
    return err
}

// Convert to cbio for async operations
cbioConn := cbio.WrapReadWriteCloser(conn)

// Perform some async operations...
// ...

// Convert back to io for standard library usage
ioConn := cbio.UnwrapReadWriteCloser(cbioConn)

// Use with standard library functions
_, err = io.Copy(os.Stdout, ioConn)
```

### Full-Duplex Concurrency

Both wrapper functions support true full-duplex operation with separate synchronization for read and write operations:

**Synchronization Design:**
- **Read operations**: Use dedicated read mutex, allowing concurrent reads with writes
- **Write operations**: Use dedicated write mutex, allowing concurrent writes with reads  
- **Close operations**: Coordinate with both read and write operations to ensure clean shutdown

**Benefits:**
- Read and write operations can execute simultaneously without blocking each other
- Maximum throughput for bidirectional data transfer
- Proper coordination with close operations to prevent race conditions
- Thread-safe for concurrent use by multiple goroutines

**Example of concurrent operations:**
```go
conn, _ := net.Dial("tcp", "example.com:80")
cbioConn := cbio.WrapReadWriteCloser(conn)

// These operations can run concurrently
go func() {
    // Read operation
    readHandler := cbio.NewReaderHandler(
        func(data []byte) { /* handle data */ },
        func(err error) { /* handle error */ },
    )
    ctx, _ := cbioConn.Read(readHandler)
    <-ctx.Done()
}()

go func() {
    // Write operation (concurrent with read)
    writeHandler := cbio.NewWriterHandler(
        func(n int) { /* handle success */ },
        func(err error) { /* handle error */ },
    )
    ctx, _ := cbioConn.Write([]byte("data"), writeHandler)
    <-ctx.Done()
}()
```

## Handler Creation

The package provides convenient constructor functions for creating handlers:

### Reader Handler

```go
// Using constructor function
readHandler := cbio.NewReaderHandler(
    func(data []byte) {
        // Handle successful read
    },
    func(err error) {
        // Handle read error
    },
)
```

### Writer Handler

```go
// Using constructor function
writeHandler := cbio.NewWriterHandler(
    func(n int) {
        // Handle successful write (n = bytes written)
    },
    func(err error) {
        // Handle write error
    },
)
```

## Best Practices

1. **Always handle immediate errors**: Check the error returned from I/O operations
2. **Implement both success and error callbacks**: Don't leave handlers incomplete
3. **Use timeouts appropriately**: Set reasonable timeouts for network operations
4. **Store context when needed**: Keep references to cancel long-running operations
5. **Proper resource cleanup**: Ensure Close() is called when appropriate
6. **Error context**: Provide meaningful error handling in callbacks
7. **Wait for completion**: Use `<-ctx.Done()` when you need to wait for operation completion

## Migration from Standard I/O

| Standard I/O Pattern | CBIO Pattern |
|---------------------|--------------|
| `n, err := r.Read(buf)` | `ctx, err := r.Read(handler, options...)` |
| `n, err := w.Write(data)` | `ctx, err := w.Write(data, handler, options...)` |
| `err := c.Close()` | `err := c.Close(options...)` |
| Synchronous, blocking | Asynchronous, non-blocking |
| Single error return | Immediate + callback errors |
| No cancellation | Built-in cancellation via CbContext |
| No configuration | Flexible options |

The callback-style approach provides much more flexibility and control over I/O operations while maintaining the familiar interface patterns from standard Go I/O.

## Implementation Details

### Context Implementation

The `CbContext` interface is implemented by the internal `cbContext` struct:

```go
type cbContext struct {
    done   <-chan struct{}
    cancel CancelFunc
}
```

This provides a lightweight context implementation specifically designed for callback-style operations, offering both completion signaling and cancellation capabilities.

### Function Types

The package defines function types for convenience:

```go
type CloseFunc func(options ...CloserOption) error

// CloseFunc implements the Closer interface
func (f CloseFunc) Close(options ...CloserOption) error {
    return f(options...)
}
```

This allows functions to directly implement the `Closer` interface, providing flexibility in implementation patterns.
