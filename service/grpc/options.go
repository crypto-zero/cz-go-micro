package grpc

import (
	"crypto/tls"

	gc "c-z.dev/go-micro/client/grpc"
	gs "c-z.dev/go-micro/server/grpc"
	"c-z.dev/go-micro/service"
)

// WithTLS sets the TLS config for the service
func WithTLS(t *tls.Config) service.Option {
	return func(o *service.Options) {
		o.Client.Init(
			gc.AuthTLS(t),
		)
		o.Server.Init(
			gs.AuthTLS(t),
		)
	}
}
