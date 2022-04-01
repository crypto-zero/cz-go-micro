// Package source is the interface for sources
package source

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"time"

	"c-z.dev/go-micro/client"
	"c-z.dev/go-micro/config/encoder"
	"c-z.dev/go-micro/config/encoder/json"
)

// ErrWatcherStopped is returned when source watcher has been stopped
var ErrWatcherStopped = errors.New("watcher stopped")

// Source is the source from which config is loaded
type Source interface {
	Read() (*ChangeSet, error)
	Write(*ChangeSet) error
	Watch() (Watcher, error)
	String() string
}

// Watcher watches a source for changes
type Watcher interface {
	Next() (*ChangeSet, error)
	Stop() error
}

// ChangeSet represents a set of changes from a source
type ChangeSet struct {
	Data      []byte
	Checksum  string
	Format    string
	Source    string
	Timestamp time.Time
}

// Sum returns the md5 checksum of the ChangeSet data
func (c *ChangeSet) Sum() string {
	h := md5.New()
	h.Write(c.Data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

type Options struct {
	// Encoder
	Encoder encoder.Encoder

	// for alternative data
	Context context.Context

	// Client to use for RPC
	Client client.Client
}

type Option func(o *Options)

func NewOptions(opts ...Option) Options {
	options := Options{
		Encoder: json.NewEncoder(),
		Context: context.Background(),
		Client:  client.DefaultClient,
	}

	for _, o := range opts {
		o(&options)
	}

	return options
}

// WithEncoder sets the source encoder
func WithEncoder(e encoder.Encoder) Option {
	return func(o *Options) {
		o.Encoder = e
	}
}

// WithClient sets the source client
func WithClient(c client.Client) Option {
	return func(o *Options) {
		o.Client = c
	}
}
