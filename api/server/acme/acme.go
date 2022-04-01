// Package acme abstracts away various ACME libraries
package acme

import (
	"crypto/tls"
	"errors"
	"net"
)

// ErrProviderNotImplemented can be returned when attempting to
// instantiate an unimplemented provider
var ErrProviderNotImplemented = errors.New("provider not implemented")

// Provider is a ACME provider interface
type Provider interface {
	// Listen returns a new listener
	Listen(...string) (net.Listener, error)
	// TLSConfig returns a tls config
	TLSConfig(...string) (*tls.Config, error)
}

// The Let's Encrypt ACME endpoints
const (
	LetsEncryptStagingCA    = "https://acme-staging-v02.api.letsencrypt.org/directory"
	LetsEncryptProductionCA = "https://acme-v02.api.letsencrypt.org/directory"
)
