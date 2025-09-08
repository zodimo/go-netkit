package cbio

import "time"

type CloserConfig struct {
	Timeout time.Duration
}

type CloserOption func(config *CloserConfig)

func WithCloseTimeout(timeout time.Duration) CloserOption {
	return func(config *CloserConfig) {
		config.Timeout = timeout
	}
}

type Closer interface {
	Close(options ...CloserOption) error
}

type CloseFunc func(options ...CloserOption) error

func (f CloseFunc) Close(options ...CloserOption) error {
	return f(options...)
}
