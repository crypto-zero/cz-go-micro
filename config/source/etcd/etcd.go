package etcd

import (
	"context"
	"fmt"
	"time"

	"c-z.dev/go-micro/config/source"
	eetcd "c-z.dev/go-micro/extension/etcd"

	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/client/v3"
)

// Currently a single etcd reader
type etcd struct {
	ctx         context.Context
	maybeClient eetcd.MaybeClient

	prefix      string
	stripPrefix string
	opts        source.Options
}

var DefaultPrefix = "/micro/config/"

func (c *etcd) Read() (*source.ChangeSet, error) {
	cc, err := c.maybeClient()
	if err != nil {
		return nil, err
	}

	rsp, err := cc.Get(context.Background(), c.prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	if rsp == nil || len(rsp.Kvs) == 0 {
		return nil, fmt.Errorf("source not found: %s", c.prefix)
	}

	kvs := make([]*mvccpb.KeyValue, 0, len(rsp.Kvs))
	for _, v := range rsp.Kvs {
		kvs = append(kvs, v)
	}

	data := makeMap(c.opts.Encoder, kvs, c.stripPrefix)

	b, err := c.opts.Encoder.Encode(data)
	if err != nil {
		return nil, fmt.Errorf("error reading source: %v", err)
	}

	cs := &source.ChangeSet{
		Timestamp: time.Now(),
		Source:    c.String(),
		Data:      b,
		Format:    c.opts.Encoder.String(),
	}
	cs.Checksum = cs.Sum()

	return cs, nil
}

func (c *etcd) String() string {
	return "etcd"
}

func (c *etcd) Watch() (source.Watcher, error) {
	cc, err := c.maybeClient()
	if err != nil {
		return nil, err
	}
	cs, err := c.Read()
	if err != nil {
		return nil, err
	}
	return newWatcher(c.ctx, c, c.prefix, c.stripPrefix, cc, cs, c.opts)
}

func (c *etcd) Write(cs *source.ChangeSet) error {
	return nil
}

func NewSource(opts ...source.Option) source.Source {
	options := source.NewOptions(opts...)

	var endpoints []string

	// check if there are any addrs
	addrs, ok := options.Context.Value(addressKey{}).([]string)
	if ok {
		endpoints = eetcd.FillAddressesPort(addrs)
	}

	if len(endpoints) == 0 {
		endpoints = []string{"localhost:2379"}
	}

	// check dial timeout option
	dialTimeout, ok := options.Context.Value(dialTimeoutKey{}).(time.Duration)
	if !ok {
		dialTimeout = 3 * time.Second // default dial timeout
	}

	config := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: dialTimeout,
	}

	u, ok := options.Context.Value(authKey{}).(*authCreds)
	if ok {
		config.Username = u.Username
		config.Password = u.Password
	}

	prefix := DefaultPrefix
	sp := ""
	f, ok := options.Context.Value(prefixKey{}).(string)
	if ok {
		prefix = f
	}

	if b, ok := options.Context.Value(stripPrefixKey{}).(bool); ok && b {
		sp = prefix
	}

	return &etcd{
		prefix:      prefix,
		stripPrefix: sp,
		opts:        options,
		ctx:         context.Background(),
		maybeClient: eetcd.NewMaybeClient(config),
	}
}
