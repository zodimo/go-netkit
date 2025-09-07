package netkit

// Chain returns a Middlewares type from a slice of middleware handlers.
func Chain(middlewares ...Middleware) Middlewares {
	return Middlewares(middlewares)
}
