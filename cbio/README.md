# CBIO - Callback-Style I/O Interfaces

The `cbio` package provides callback-style alternatives to Go's standard I/O interfaces, enabling asynchronous operations with proper error handling, cancellation support, and configurable options.

## Overview

Traditional Go I/O operations are synchronous and block until completion. The `cbio` package transforms these operations into asynchronous, callback-driven patterns that provide:

- **Non-blocking operations**: I/O operations return immediately with a `Canceler`
- **Structured error handling**: Separate immediate errors from operation errors
- **Cancellation support**: All operations can be cancelled mid-flight
- **Configurable behavior**: Timeouts, buffer sizes, and other options per operation
- **Composable interfaces**: ReadCloser, WriteCloser, ReadWriteCloser combinations

## Core Interfaces

### Reader

```go
type Reader interface {
    Read(handler ReaderHandler, options ...ReaderOption) (Canceler, error)
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
    Write(p []byte, handler WriterHandler, options ...WriterOption) (Canceler, error)
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
    canceler, err := reader.Read(
        readHandler,
        cbio.WithReadTimeout(5*time.Second),
        cbio.WithReaderBufferSize(8192),
    )
    
    if err != nil {
        fmt.Printf("Failed to start read: %v\n", err)
        return
    }
    
    // Can cancel the read operation
    // canceler.Cancel()
    
    _ = canceler
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
    
    canceler, err := writer.Write(
        data,
        writeHandler,
        cbio.WithWriteTimeout(3*time.Second),
    )
    
    if err != nil {
        fmt.Printf("Failed to start write: %v\n", err)
        return
    }
    
    // Operation is now running asynchronously
    _ = canceler
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
            
            canceler, err := rwc.Read(readHandler)
            if err != nil {
                fmt.Printf("Failed to start read: %v\n", err)
            }
            _ = canceler
        },
        func(err error) {
            fmt.Printf("Write failed: %v\n", err)
        },
    )
    
    canceler, err := rwc.Write([]byte("ping"), writeHandler)
    if err != nil {
        fmt.Printf("Failed to start write: %v\n", err)
        return
    }
    
    _ = canceler
}
```

## Error Handling Patterns

### Immediate vs. Operation Errors

```go
canceler, err := writer.Write(data, handler)
if err != nil {
    // Immediate error - operation couldn't be started
    // Examples: invalid parameters, resource unavailable
    fmt.Printf("Immediate error: %v\n", err)
    return
}

// Operation started successfully
// Any runtime errors will be delivered to handler.OnError()
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

canceler, err := writer.Write(
    data,
    writeHandler,
    cbio.WithWriteTimeout(5*time.Second), // Will timeout after 5 seconds
)
```

## Cancellation Patterns

### Immediate Cancellation

```go
canceler, err := reader.Read(handler)
if err != nil {
    return
}

// Cancel immediately
canceler.Cancel()
```

### Conditional Cancellation

```go
canceler, err := reader.Read(handler)
if err != nil {
    return
}

// Cancel after some condition
go func() {
    time.Sleep(2*time.Second)
    fmt.Println("Cancelling read operation...")
    canceler.Cancel()
}()
```

## Configuration Options

### Default Configurations

```go
// Reader defaults
readerConfig := cbio.DefaultReaderConfig()
// Timeout: 0 (no timeout)
// BufferSize: 4096

// Writer defaults - no default config function yet
// Timeout: 0 (no timeout)

// Closer defaults - no default config function yet  
// Timeout: 0 (no timeout)
```

### Custom Configurations

```go
// Read with custom configuration
canceler, err := reader.Read(
    handler,
    cbio.WithReadTimeout(10*time.Second),
    cbio.WithReaderBufferSize(16384),
)

// Write with timeout
canceler, err := writer.Write(
    data,
    handler, 
    cbio.WithWriteTimeout(5*time.Second),
)

// Close with timeout
err := closer.Close(cbio.WithCloseTimeout(2*time.Second))
```

## Best Practices

1. **Always handle immediate errors**: Check the error returned from I/O operations
2. **Implement both success and error callbacks**: Don't leave handlers incomplete
3. **Use timeouts appropriately**: Set reasonable timeouts for network operations
4. **Store cancelers when needed**: Keep references to cancel long-running operations
5. **Proper resource cleanup**: Ensure Close() is called when appropriate
6. **Error context**: Provide meaningful error handling in callbacks

## Migration from Standard I/O

| Standard I/O Pattern | CBIO Pattern |
|---------------------|--------------|
| `n, err := r.Read(buf)` | `canceler, err := r.Read(handler, options...)` |
| `n, err := w.Write(data)` | `canceler, err := w.Write(data, handler, options...)` |
| `err := c.Close()` | `err := c.Close(options...)` |
| Synchronous, blocking | Asynchronous, non-blocking |
| Single error return | Immediate + callback errors |
| No cancellation | Built-in cancellation |
| No configuration | Flexible options |

The callback-style approach provides much more flexibility and control over I/O operations while maintaining the familiar interface patterns from standard Go I/O.
