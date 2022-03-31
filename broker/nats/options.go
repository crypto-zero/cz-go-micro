package nats

import (
	"c-z.dev/go-micro/broker"

	"github.com/nats-io/nats.go"
)

type (
	optionsKey         struct{}
	drainConnectionKey struct{}
)

// Options accepts nats.Options
func Options(opts nats.Options) broker.Option {
	return setBrokerOption(optionsKey{}, opts)
}

// DrainConnection will drain subscription on close
func DrainConnection() broker.Option {
	return setBrokerOption(drainConnectionKey{}, struct{}{})
}
