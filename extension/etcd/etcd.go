package etcd

import (
	"context"
	"time"

	"c-z.dev/go-micro/logger"

	"go.etcd.io/etcd/client/v3"
)

type Client struct {
	*clientv3.Client
}

func (c Client) Watch(ctx context.Context, key string, handler func([]*clientv3.Event, error),
	opts ...clientv3.OpOption,
) {
	go c.watch(ctx, key, handler, opts...)
}

func (c Client) watch(ctx context.Context, key string, handler func([]*clientv3.Event, error),
	opts ...clientv3.OpOption,
) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		wc := c.Watcher.Watch(ctx, key, opts...)
	LOOP:
		for {
			select {
			case es, ok := <-wc:
				if !ok {
					break LOOP
				}
				events, err := es.Events, es.Err()
				if err != nil {
					if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
						logger.Errorf("etcd watch key: %s getting error: %v", key, err)
					}
				}
				handler(events, err)
			case <-ctx.Done():
				return
			}
		}
	}
}

func New(config clientv3.Config) (Client, error) {
	config.AutoSyncInterval = time.Second
	if c, err := clientv3.New(config); err != nil {
		return Client{}, err
	} else {
		return Client{c}, nil
	}
}

func NewMaybeClient(config clientv3.Config) func() (Client, error) {
	c, err := New(config)
	return func() (Client, error) {
		return c, err
	}
}
