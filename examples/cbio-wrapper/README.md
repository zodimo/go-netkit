# CBIO Wrapper Example

This example demonstrates how to use the `cbio.WrapReadWriteCloser` function to wrap standard Go `io.ReadWriteCloser` instances into callback-style `cbio.ReadWriteCloser` interfaces.

## What This Example Shows

1. **Basic Wrapping**: How to wrap a standard `io.ReadWriteCloser` with `cbio.WrapReadWriteCloser`
2. **Callback-Style Reading**: Using `cbio.Reader` with success/error handlers
3. **Callback-Style Writing**: Using `cbio.Writer` with success/error handlers  
4. **Timeout Handling**: Demonstrating read operations with configurable timeouts
5. **Manual Cancellation**: Showing how to cancel operations using `CbContext.Cancel()`
6. **Resource Cleanup**: Proper closing of wrapped connections with timeout support

## Key Features Demonstrated

### Asynchronous Operations
All I/O operations return immediately with a `CbContext` for tracking completion and cancellation.

### Callback-Based Error Handling
Separate success and error callbacks provide clear handling of different outcomes:
```go
readHandler := cbio.NewReaderHandler(
    func(data []byte) {
        // Handle successful read
    },
    func(err error) {
        // Handle read error
    },
)
```

### Timeout Support
Operations can be configured with timeouts:
```go
ctx, err := cbioConn.Read(handler, cbio.WithReadTimeout(2*time.Second))
```

### Cancellation Support
Operations can be cancelled manually:
```go
ctx, err := cbioConn.Read(handler)
// Cancel after some condition
ctx.Cancel()
```

## Running the Example

```bash
cd examples/cbio-wrapper
go run main.go
```

## Expected Output

The example will demonstrate:
- ✅ Successful read and write operations
- ⏳ Timeout behavior when no data is available
- 🚫 Manual cancellation of operations
- 🔒 Proper resource cleanup

## Use Cases

This wrapper is particularly useful for:

1. **Migration from Standard I/O**: Gradually moving existing code to callback-style I/O
2. **Network Operations**: Wrapping network connections for asynchronous handling
3. **File Operations**: Converting file I/O to non-blocking callback patterns
4. **Testing**: Creating callback-style interfaces from mock implementations

## Integration with Go-NetKit

The wrapper integrates seamlessly with the broader go-netkit ecosystem, allowing standard I/O implementations to work with:
- Transport layers
- Middleware chains
- Context-aware operations
- Timeout and cancellation patterns
