package cmd

import (
	"context"

	"c-z.dev/go-micro/auth"
	"c-z.dev/go-micro/broker"
	"c-z.dev/go-micro/client"
	"c-z.dev/go-micro/client/selector"
	"c-z.dev/go-micro/config"
	"c-z.dev/go-micro/debug/profile"
	"c-z.dev/go-micro/debug/trace"
	"c-z.dev/go-micro/registry"
	"c-z.dev/go-micro/runtime"
	"c-z.dev/go-micro/server"
	"c-z.dev/go-micro/store"
	"c-z.dev/go-micro/transport"
)

type Options struct {
	// For the Command Line itself
	Name        string
	Description string
	Version     string

	// We need pointers to things, so we can swap them out if needed.
	Broker    *broker.Broker
	Registry  *registry.Registry
	Selector  *selector.Selector
	Transport *transport.Transport
	Config    *config.Config
	Client    *client.Client
	Server    *server.Server
	Runtime   *runtime.Runtime
	Store     *store.Store
	Tracer    *trace.Tracer
	Auth      *auth.Auth
	Profile   *profile.Profile

	Brokers    map[string]func(...broker.Option) broker.Broker
	Configs    map[string]func(...config.Option) (config.Config, error)
	Clients    map[string]func(...client.Option) client.Client
	Registries map[string]func(...registry.Option) registry.Registry
	Selectors  map[string]func(...selector.Option) selector.Selector
	Servers    map[string]func(...server.Option) server.Server
	Transports map[string]func(...transport.Option) transport.Transport
	Runtimes   map[string]func(...runtime.Option) runtime.Runtime
	Stores     map[string]func(...store.Option) store.Store
	Tracers    map[string]func(...trace.Option) trace.Tracer
	Auths      map[string]func(...auth.Option) auth.Auth
	Profiles   map[string]func(...profile.Option) profile.Profile

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

// Name command line Name
func Name(n string) Option {
	return func(o *Options) {
		o.Name = n
	}
}

// Description command line Description
func Description(d string) Option {
	return func(o *Options) {
		o.Description = d
	}
}

// Version command line Version
func Version(v string) Option {
	return func(o *Options) {
		o.Version = v
	}
}

func Broker(b *broker.Broker) Option {
	return func(o *Options) {
		o.Broker = b
	}
}

func Config(c *config.Config) Option {
	return func(o *Options) {
		o.Config = c
	}
}

func Selector(s *selector.Selector) Option {
	return func(o *Options) {
		o.Selector = s
	}
}

func Registry(r *registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

func Runtime(r *runtime.Runtime) Option {
	return func(o *Options) {
		o.Runtime = r
	}
}

func Transport(t *transport.Transport) Option {
	return func(o *Options) {
		o.Transport = t
	}
}

func Client(c *client.Client) Option {
	return func(o *Options) {
		o.Client = c
	}
}

func Server(s *server.Server) Option {
	return func(o *Options) {
		o.Server = s
	}
}

func Store(s *store.Store) Option {
	return func(o *Options) {
		o.Store = s
	}
}

func Tracer(t *trace.Tracer) Option {
	return func(o *Options) {
		o.Tracer = t
	}
}

func Auth(a *auth.Auth) Option {
	return func(o *Options) {
		o.Auth = a
	}
}

func Profile(p *profile.Profile) Option {
	return func(o *Options) {
		o.Profile = p
	}
}

// NewBroker new broker func
func NewBroker(name string, b func(...broker.Option) broker.Broker) Option {
	return func(o *Options) {
		o.Brokers[name] = b
	}
}

// NewClient new client func
func NewClient(name string, b func(...client.Option) client.Client) Option {
	return func(o *Options) {
		o.Clients[name] = b
	}
}

// NewRegistry new registry func
func NewRegistry(name string, r func(...registry.Option) registry.Registry) Option {
	return func(o *Options) {
		o.Registries[name] = r
	}
}

// NewSelector new selector func
func NewSelector(name string, s func(...selector.Option) selector.Selector) Option {
	return func(o *Options) {
		o.Selectors[name] = s
	}
}

// NewServer new server func
func NewServer(name string, s func(...server.Option) server.Server) Option {
	return func(o *Options) {
		o.Servers[name] = s
	}
}

// NewTransport new transport func
func NewTransport(name string, t func(...transport.Option) transport.Transport) Option {
	return func(o *Options) {
		o.Transports[name] = t
	}
}

// NewRuntime new runtime func
func NewRuntime(name string, r func(...runtime.Option) runtime.Runtime) Option {
	return func(o *Options) {
		o.Runtimes[name] = r
	}
}

// NewTracer new tracer func
func NewTracer(name string, t func(...trace.Option) trace.Tracer) Option {
	return func(o *Options) {
		o.Tracers[name] = t
	}
}

// NewAuth new auth func
func NewAuth(name string, t func(...auth.Option) auth.Auth) Option {
	return func(o *Options) {
		o.Auths[name] = t
	}
}
