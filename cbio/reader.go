package cbio

import "time"

var _ ReaderHandler = (*readerHandler)(nil)

type ReaderConfig struct {
	Timeout    time.Duration
	BufferSize int
}

type ReaderOption func(config *ReaderConfig)

func WithReadTimeout(timeout time.Duration) ReaderOption {
	return func(config *ReaderConfig) {
		config.Timeout = timeout
	}
}

func WithReaderBufferSize(size int) ReaderOption {
	return func(config *ReaderConfig) {
		config.BufferSize = size
	}
}

func DefaultReaderConfig() *ReaderConfig {
	return &ReaderConfig{
		Timeout:    0,
		BufferSize: 4096,
	}
}

type OnReadSuccessHandler func(p []byte)
type OnReadErrorHandler func(err error)

type ReaderHandler interface {
	OnSuccess(p []byte)
	OnError(err error)
}

type readerHandler struct {
	onSuccess OnReadSuccessHandler
	onError   OnReadErrorHandler
}

func NewReaderHandler(onSuccess OnReadSuccessHandler, onError OnReadErrorHandler) ReaderHandler {
	return &readerHandler{
		onSuccess: onSuccess,
		onError:   onError,
	}
}

func (h *readerHandler) OnSuccess(p []byte) {
	h.onSuccess(p)
}

func (h *readerHandler) OnError(err error) {
	h.onError(err)
}

// @TODO maybe we sould return a waiter alongside the canceler
type Reader interface {
	Read(handler ReaderHandler, options ...ReaderOption) (CbContext, error)
}
