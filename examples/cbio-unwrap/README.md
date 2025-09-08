# CBIO Unwrap Example

This example demonstrates how to use the `cbio.UnwrapReadWriteCloser` function to convert callback-style `cbio.ReadWriteCloser` instances back into standard `io.ReadWriteCloser` interfaces.

## Features Demonstrated

1. **Basic Unwrapping**: Converting cbio instances to standard io interfaces
2. **Bidirectional Conversion**: Converting back and forth between io and cbio
3. **Standard Library Integration**: Using unwrapped connections with standard library functions like `io.Copy`

## Key Concepts

### UnwrapReadWriteCloser Function

```go
func UnwrapReadWriteCloser(rwc cbio.ReadWriteCloser) io.ReadWriteCloser
```

This function converts a callback-style `cbio.ReadWriteCloser` into a standard `io.ReadWriteCloser` that:
- Provides synchronous blocking operations
- Returns standard `(n int, err error)` results
- Works with any standard library function expecting `io.ReadWriteCloser`
- Maintains thread safety for concurrent operations

### Usage Patterns

1. **Direct Unwrapping**: Convert cbio instances for use with standard I/O code
2. **Library Integration**: Pass cbio instances to functions expecting standard interfaces
3. **Bidirectional Workflow**: Convert between io and cbio as needed in your application

## Running the Example

```bash
cd examples/cbio-unwrap
go run main.go
```

## Example Output

```
CBIO Unwrap Example - Bidirectional Conversion
==============================================

1. Basic Unwrapping Example:
  Created cbio connection from mock: *cbio.readWriteCloserWrapper
  Unwrapped to standard io: *cbio.ioReadWriteCloserWrapper
  Read 17 bytes: "Hello from cbio!"
  Wrote 33 bytes
  Connection closed successfully

2. Bidirectional Conversion Example:
  Starting with standard io.ReadWriteCloser
  Wrapped to cbio.ReadWriteCloser
  Unwrapped back to io.ReadWriteCloser
  Successfully read 24 bytes: "Bidirectional test data"
  Connection closed successfully

3. Standard Library Integration Example:
  io.Copy transferred 38 bytes: "Data for standard library integration"
  Standard library integration successful
```

## Use Cases

- **Legacy Code Integration**: Use cbio instances with existing code that expects standard I/O
- **Third-Party Libraries**: Pass cbio connections to libraries expecting `io.ReadWriteCloser`
- **Standard Library Functions**: Use cbio instances with functions like `io.Copy`, `io.ReadAll`, etc.
- **Testing**: Create test scenarios that work with both cbio and standard I/O patterns

## Related Examples

- [CBIO Wrapper Example](../cbio-wrapper/): Shows the reverse conversion (io to cbio)
- [SockJS Client Example](../sockjs-client/): Real-world usage of cbio interfaces
