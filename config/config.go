// Package config is an interface for dynamic configuration.
package config

import (
	"context"

	"c-z.dev/go-micro/config/loader"
	"c-z.dev/go-micro/config/reader"
	"c-z.dev/go-micro/config/source"
	"c-z.dev/go-micro/config/source/file"
)

// Config is an interface abstraction for dynamic configuration
type Config interface {
	// Values provide the reader.Values interface
	reader.Values
	// Init the config
	Init(opts ...Option) error
	// Options in the config
	Options() Options
	// Close stop the config loader/watcher
	Close() error
	// Load config sources
	Load(source ...source.Source) error
	// Sync force a source change-set sync
	Sync() error
	// Watch a value for changes
	Watch(path ...string) (Watcher, error)
}

// Watcher is the config watcher
type Watcher interface {
	Next() (reader.Value, error)
	Stop() error
}

type Options struct {
	Loader loader.Loader
	Reader reader.Reader
	Source []source.Source

	// for alternative data
	Context context.Context
}

type Option func(o *Options)

// Default Config Manager
var DefaultConfig, _ = NewConfig()

// NewConfig returns new config
func NewConfig(opts ...Option) (Config, error) {
	return newConfig(opts...)
}

// Bytes return config as raw json
func Bytes() []byte {
	return DefaultConfig.Bytes()
}

// Map return config as a map
func Map() map[string]interface{} {
	return DefaultConfig.Map()
}

// Scan values to a go type
func Scan(v interface{}) error {
	return DefaultConfig.Scan(v)
}

// Sync force a source changeset sync
func Sync() error {
	return DefaultConfig.Sync()
}

// Get a value from the config
func Get(path ...string) reader.Value {
	return DefaultConfig.Get(path...)
}

// Load config sources
func Load(source ...source.Source) error {
	return DefaultConfig.Load(source...)
}

// Watch a value for changes
func Watch(path ...string) (Watcher, error) {
	return DefaultConfig.Watch(path...)
}

// LoadFile is shorthand for creating a file source and loading it
func LoadFile(path string) error {
	return Load(file.NewSource(
		file.WithPath(path),
	))
}
