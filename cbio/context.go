package cbio

type CancelFunc func()

type CbContext interface {
	Done() <-chan struct{}
	Cancel()
}

type cbContext struct {
	done   <-chan struct{}
	cancel CancelFunc
}

func (c *cbContext) Done() <-chan struct{} {
	return c.done
}

func (c *cbContext) Cancel() {
	c.cancel()
}

func NewCbContext(cancel CancelFunc, done <-chan struct{}) CbContext {
	return &cbContext{
		done:   done,
		cancel: cancel,
	}
}
