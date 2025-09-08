package cbio

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
