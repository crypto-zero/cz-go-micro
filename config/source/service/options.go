package service

import (
	"context"

	"c-z.dev/go-micro/config/source"
)

type (
	serviceNameKey struct{}
	namespaceKey   struct{}
	pathKey        struct{}
)

func ServiceName(name string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, serviceNameKey{}, name)
	}
}

func Namespace(namespace string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, namespaceKey{}, namespace)
	}
}

func Path(path string) source.Option {
	return func(o *source.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, pathKey{}, path)
	}
}
