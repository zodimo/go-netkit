package cbio

import (
	"context"
	"io"
	"sync"
	"time"
)

// readWriteCloserWrapper wraps a standard io.ReadWriteCloser to implement cbio.ReadWriteCloser
type readWriteCloserWrapper struct {
	rwc   io.ReadWriteCloser
	mutex sync.RWMutex
}

// WrapReadWriteCloser wraps a standard io.ReadWriteCloser into a cbio.ReadWriteCloser.
// This enables seamless integration between standard Go I/O and callback-style cbio operations.
//
// The wrapper provides:
// - Non-blocking Read/Write operations using goroutines
// - Proper callback-based success/error handling
// - Cancellation support through CbContext
// - Configurable timeouts and options
// - Thread-safe concurrent operations
//
// Example usage:
//
//	conn, err := net.Dial("tcp", "example.com:80")
//	if err != nil {
//	    return err
//	}
//
//	cbioConn := cbio.WrapReadWriteCloser(conn)
//
//	// Use with callback-style operations
//	readHandler := cbio.NewReaderHandler(
//	    func(data []byte) {
//	        fmt.Printf("Received: %s\n", string(data))
//	    },
//	    func(err error) {
//	        fmt.Printf("Read error: %v\n", err)
//	    },
//	)
//
//	ctx, err := cbioConn.Read(readHandler, cbio.WithReadTimeout(5*time.Second))
//	if err != nil {
//	    return err
//	}
//
//	// Wait for completion or cancel
//	<-ctx.Done()
func WrapReadWriteCloser(rwc io.ReadWriteCloser) ReadWriteCloser {
	return &readWriteCloserWrapper{
		rwc: rwc,
	}
}

// Read implements cbio.Reader interface with non-blocking operation
func (w *readWriteCloserWrapper) Read(handler ReaderHandler, options ...ReaderOption) (CbContext, error) {
	if handler == nil {
		return nil, io.ErrShortBuffer // Use a standard error for invalid input
	}

	// Apply configuration options
	config := DefaultReaderConfig()
	for _, option := range options {
		option(config)
	}

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	if config.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), config.Timeout)
	}

	done := make(chan struct{})
	cbCtx := NewCbContext(CancelFunc(cancel), done)

	// Start read operation in goroutine
	go func() {
		defer close(done)
		defer cancel()

		// Create buffer for reading
		buffer := make([]byte, config.BufferSize)

		// Use a channel to handle the read operation result
		type readResult struct {
			data []byte
			err  error
		}

		resultChan := make(chan readResult, 1)

		// Perform read operation in another goroutine
		go func() {
			w.mutex.RLock()
			n, err := w.rwc.Read(buffer)
			w.mutex.RUnlock()

			if err != nil {
				resultChan <- readResult{nil, err}
				return
			}

			// Success - return the data that was actually read
			data := make([]byte, n)
			copy(data, buffer[:n])
			resultChan <- readResult{data, nil}
		}()

		// Wait for either the read to complete or context cancellation
		select {
		case <-ctx.Done():
			// Operation was cancelled or timed out
			if ctx.Err() == context.DeadlineExceeded {
				handler.OnError(context.DeadlineExceeded)
			} else {
				handler.OnError(ctx.Err())
			}
			return
		case result := <-resultChan:
			if result.err != nil {
				handler.OnError(result.err)
				return
			}
			handler.OnSuccess(result.data)
		}
	}()

	return cbCtx, nil
}

// Write implements cbio.Writer interface with non-blocking operation
func (w *readWriteCloserWrapper) Write(p []byte, handler WriterHandler, options ...WriterOption) (CbContext, error) {
	if handler == nil {
		return nil, io.ErrShortBuffer // Use a standard error for invalid input
	}
	if len(p) == 0 {
		// Handle empty write immediately
		go func() {
			handler.OnSuccess(0)
		}()
		done := make(chan struct{})
		close(done)
		return NewCbContext(func() {}, done), nil
	}

	// Apply configuration options
	config := DefaultWriterConfig()
	for _, option := range options {
		option(config)
	}

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	if config.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), config.Timeout)
	}

	done := make(chan struct{})
	cbCtx := NewCbContext(CancelFunc(cancel), done)

	// Start write operation in goroutine
	go func() {
		defer close(done)
		defer cancel()

		// Use a channel to handle the write operation result
		type writeResult struct {
			n   int
			err error
		}

		resultChan := make(chan writeResult, 1)

		// Perform write operation in another goroutine
		go func() {
			w.mutex.Lock()
			n, err := w.rwc.Write(p)
			w.mutex.Unlock()

			resultChan <- writeResult{n, err}
		}()

		// Wait for either the write to complete or context cancellation
		select {
		case <-ctx.Done():
			// Operation was cancelled or timed out
			if ctx.Err() == context.DeadlineExceeded {
				handler.OnError(context.DeadlineExceeded)
			} else {
				handler.OnError(ctx.Err())
			}
			return
		case result := <-resultChan:
			if result.err != nil {
				handler.OnError(result.err)
				return
			}
			handler.OnSuccess(result.n)
		}
	}()

	return cbCtx, nil
}

// Close implements cbio.Closer interface
func (w *readWriteCloserWrapper) Close(options ...CloserOption) error {
	// Apply configuration options
	config := &CloserConfig{
		Timeout: 0, // Default: no timeout
	}
	for _, option := range options {
		option(config)
	}

	// Create a channel for the close operation
	done := make(chan error, 1)

	// Perform close operation in goroutine to handle timeout
	go func() {
		w.mutex.Lock()
		err := w.rwc.Close()
		w.mutex.Unlock()
		done <- err
	}()

	// Handle timeout if configured
	if config.Timeout > 0 {
		select {
		case err := <-done:
			return err
		case <-time.After(config.Timeout):
			return context.DeadlineExceeded
		}
	}

	// No timeout - wait for completion
	return <-done
}
