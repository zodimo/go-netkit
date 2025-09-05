package netkit

type Middleware func(handler TransportHandler) TransportHandler
