// Package mucp provides an mucp client
package mucp

import (
	"c-z.dev/go-micro/client"
)

// NewClient returns a new micro client interface
func NewClient(opts ...client.Option) client.Client {
	return client.NewClient(opts...)
}
