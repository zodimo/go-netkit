package net

import (
	"errors"
	"fmt"
	"io"
)

// CloseError represents a close message.
type CloseError interface {
	error
	Code() int
	Reason() string
}

func AsCloseError(err error) CloseError {
	// EOF is the error returned by Read when no more input is available.
	// (Read must return EOF itself, not an error wrapping EOF,
	// because callers will test for EOF using ==.)
	// Functions should return EOF only to signal a graceful end of input.
	// If the EOF occurs unexpectedly in a structured data stream,
	// the appropriate error is either [ErrUnexpectedEOF] or some other error
	// giving more detail.
	// var EOF = errors.New("EOF")

	if errors.Is(err, io.EOF) {
		return NewCloseError(1000, "EOF")
	}
	if e, ok := err.(CloseError); ok {
		return e
	}
	return nil
}

func IsCloseError(err error) bool {
	if _, ok := err.(CloseError); ok {
		return true

	}
	return false
}

func IsUnexpectedCloseError(err error, expectedCodes ...int) bool {
	if e, ok := err.(CloseError); ok {
		for _, code := range expectedCodes {
			if e.Code() == code {
				return false
			}
		}
		return true
	}
	return false
}

type closeError struct {
	code   int
	reason string
}

func NewCloseError(code int, reason string) CloseError {
	return &closeError{
		code:   code,
		reason: reason,
	}
}

func (e *closeError) Code() int {
	return e.code
}

func (e *closeError) Reason() string {
	return e.reason
}

func (e *closeError) Error() string {
	return fmt.Sprintf("close error: %d, %s", e.code, e.reason)
}
