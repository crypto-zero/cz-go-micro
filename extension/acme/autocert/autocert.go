// Package autocert is the ACME provider from golang.org/x/crypto/acme/autocert
// This provider does not take any config.
package autocert

import (
	"crypto/tls"
	"net"
	"os"

	"c-z.dev/go-micro/api/server/acme"
	"c-z.dev/go-micro/logger"

	"golang.org/x/crypto/acme/autocert"
)

// autoCertACME is the ACME provider from golang.org/x/crypto/acme/autocert
type autoCertProvider struct{}

// Listen implements acme.Provider
func (a *autoCertProvider) Listen(hosts ...string) (net.Listener, error) {
	return autocert.NewListener(hosts...), nil
}

// TLSConfig returns a new tls config
func (a *autoCertProvider) TLSConfig(hosts ...string) (*tls.Config, error) {
	// create a new manager
	m := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
	}
	if len(hosts) > 0 {
		m.HostPolicy = autocert.HostWhitelist(hosts...)
	}
	dir := cacheDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		if logger.V(logger.InfoLevel, logger.DefaultLogger) {
			logger.Infof("warning: autocert not using a cache: %v", err)
		}
	} else {
		m.Cache = autocert.DirCache(dir)
	}
	return m.TLSConfig(), nil
}

// NewProvider new returns an autocert acme.Provider
func NewProvider() acme.Provider {
	return &autoCertProvider{}
}
