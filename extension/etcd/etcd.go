package etcd

import (
	"context"
	"fmt"
	"net"
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

type MaybeClient func() (Client, error)

func (mc MaybeClient) Client() Client {
	c, _ := mc()
	return c
}

func (mc MaybeClient) Error() error {
	_, err := mc()
	return err
}

func NewMaybeClient(config clientv3.Config) MaybeClient {
	c, err := New(config)
	return func() (Client, error) {
		return c, err
	}
}

func FillAddressesPort(addresses []string) (out []string) {
	for _, address := range addresses {
		addr, port, err := net.SplitHostPort(address)
		if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
			port = "2379"
			addr = address
			out = append(out, fmt.Sprintf("%s:%s", addr, port))
			return
		}
		out = append(out, address)
	}
	return
}
