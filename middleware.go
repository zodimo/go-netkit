package net

type Middleware func(handler TransportHandler) TransportHandler
