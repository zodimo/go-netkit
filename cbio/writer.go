package cbio

import "time"

var _ WriterHandler = (*writerHandler)(nil)

type WriterConfig struct {
	Timeout time.Duration
}

type WriterOption func(config *WriterConfig)

func WithWriteTimeout(timeout time.Duration) WriterOption {
	return func(config *WriterConfig) {
		config.Timeout = timeout
	}
}

func DefaultWriterConfig() *WriterConfig {
	return &WriterConfig{
		Timeout: 0,
	}
}

func WithDefaultWriterConfig() WriterOption {
	return func(config *WriterConfig) {
		config.Timeout = 0
	}
}

type WriterHandler interface {
	OnSuccess(n int)
	OnError(err error)
}

type writerHandler struct {
	onSuccess OnWriteSuccessHandler
	onError   OnWriteErrorHandler
}

func (h *writerHandler) OnSuccess(n int) {
	h.onSuccess(n)
}

func (h *writerHandler) OnError(err error) {
	h.onError(err)
}

type OnWriteSuccessHandler func(n int)
type OnWriteErrorHandler func(err error)

func NewWriterHandler(onSuccess OnWriteSuccessHandler, onError OnWriteErrorHandler) WriterHandler {
	return &writerHandler{
		onSuccess: onSuccess,
		onError:   onError,
	}
}

type Writer interface {
	Write(p []byte, handler WriterHandler, options ...WriterOption) (CbContext, error)
}
