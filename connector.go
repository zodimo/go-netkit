package netkit

import (
	"context"
)

type Connector interface {
	// this can be a blocking operation and can be cancelled
	Connect(ctx context.Context) (TransportReceiver, error)
}
