package cbio

type CancelFunc func()

func (f CancelFunc) Cancel() {
	f()
}

type Canceler interface {
	Cancel()
}
