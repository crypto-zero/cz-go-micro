// Package mucp provides an mucp server
package mucp

import (
	"c-z.dev/go-micro/server"
)

// NewServer returns a micro server interface
func NewServer(opts ...server.Option) server.Server {
	return server.NewServer(opts...)
}
