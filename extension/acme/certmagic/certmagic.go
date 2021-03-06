// Package certmagic is the ACME provider from github.com/caddyserver/certmagic
package certmagic

import (
	"crypto/tls"
	"math/rand"
	"net"
	"time"

	"c-z.dev/go-micro/api/server/acme"
	"c-z.dev/go-micro/logger"

	"github.com/caddyserver/certmagic"
)

type certMagicProvider struct {
	opts acme.Options
}

// TODO: set self-contained options
func (c *certMagicProvider) setup() {
	certmagic.DefaultACME.CA = c.opts.CA
	if c.opts.ChallengeProvider != nil {
		// Enabling DNS Challenge disables the other challenges
	}
	if c.opts.OnDemand {
		certmagic.Default.OnDemand = new(certmagic.OnDemandConfig)
	}
	if c.opts.Cache != nil {
		// already validated by new()
		certmagic.Default.Storage = c.opts.Cache.(certmagic.Storage)
	}
	// If multiple instances of the provider are running, inject some
	// randomness, so they don't collide
	// RenewalWindowRatio [0.33 - 0.50)
	rand.Seed(time.Now().UnixNano())
	randomRatio := float64(rand.Intn(17)+33) * 0.01
	certmagic.Default.RenewalWindowRatio = randomRatio
}

func (c *certMagicProvider) Listen(hosts ...string) (net.Listener, error) {
	c.setup()
	return certmagic.Listen(hosts)
}

func (c *certMagicProvider) TLSConfig(hosts ...string) (*tls.Config, error) {
	c.setup()
	return certmagic.TLS(hosts)
}

// NewProvider returns a cert-magic provider
func NewProvider(options ...acme.Option) acme.Provider {
	opts := acme.DefaultOptions()

	for _, o := range options {
		o(&opts)
	}

	if opts.Cache != nil {
		if _, ok := opts.Cache.(certmagic.Storage); !ok {
			logger.Fatal("ACME: cache provided doesn't implement certmagic's Storage interface")
		}
	}

	return &certMagicProvider{
		opts: opts,
	}
}
