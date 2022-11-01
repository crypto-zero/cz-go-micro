// Package etcd provides an etcd service registry
package etcd

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	eetcd "c-z.dev/go-micro/extension/etcd"
	"c-z.dev/go-micro/logger"
	"c-z.dev/go-micro/registry"

	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	"go.etcd.io/etcd/client/v3"
)

var prefix = "/micro/registry/"

type baseEtcd struct {
	ctx        context.Context
	apiTimeout time.Duration
	client     eetcd.MaybeClient
}

func (e baseEtcd) call(ctx context.Context, f func(ctx context.Context, c eetcd.Client) error) error {
	if ctx == nil {
		ctx = e.ctx
	}
	c, err := e.client()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(e.ctx, e.apiTimeout)
	defer cancel()
	return f(ctx, c)
}

func (e baseEtcd) encodeService(s *registry.Service) (string, error) {
	if b, err := json.Marshal(s); err == nil {
		return string(b), nil
	} else {
		return "", err
	}
}

func (e baseEtcd) decodeService(ds []byte) (*registry.Service, error) {
	var s registry.Service
	if err := json.Unmarshal(ds, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (e baseEtcd) nodePath(serviceName, nodeID string) string {
	service := strings.Replace(serviceName, "/", "-", -1)
	node := strings.Replace(nodeID, "/", "-", -1)
	return path.Join(prefix, service, node)
}

func (e baseEtcd) servicePath(s string) string {
	return path.Join(prefix, strings.Replace(s, "/", "-", -1))
}

type etcdRegistry struct {
	*baseEtcd
	options registry.Options

	sync.RWMutex
	register map[string]string
	leases   map[string]clientv3.LeaseID
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	e := &etcdRegistry{
		baseEtcd: &baseEtcd{},
		options:  registry.Options{},
		register: make(map[string]string),
		leases:   make(map[string]clientv3.LeaseID),
	}
	e.configure(opts...)
	return e
}

func (e *etcdRegistry) configure(opts ...registry.Option) {
	config := clientv3.Config{
		Endpoints: []string{"127.0.0.1:2379"},
	}

	for _, o := range opts {
		o(&e.options)
	}

	if e.options.Timeout == 0 {
		e.options.Timeout = 5 * time.Second
	}

	if e.options.Secure || e.options.TLSConfig != nil {
		tlsConfig := e.options.TLSConfig
		if tlsConfig == nil {
			tlsConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
		config.TLS = tlsConfig
	}

	if e.options.Context != nil {
		u, ok := e.options.Context.Value(authKey{}).(*authCreds)
		if ok {
			config.Username = u.Username
			config.Password = u.Password
		}
	}

	// if we got addresses then we'll update
	if addresses := eetcd.FillAddressesPort(e.options.Addrs); len(addresses) > 0 {
		config.Endpoints = addresses
	}

	// close last client
	if c := e.client; c != nil {
		if c := c.Client(); c.Client != nil {
			_ = c.Close()
		}
	}
	e.client = eetcd.NewMaybeClient(config)

	e.ctx = e.options.Context
	if e.ctx == nil {
		e.ctx = context.Background()
	}
	e.apiTimeout = e.options.Timeout
}

func (e *etcdRegistry) Init(opts ...registry.Option) error {
	e.configure(opts...)
	return e.client.Error()
}

func (e *etcdRegistry) Options() registry.Options {
	return e.options
}

func (e *etcdRegistry) registerNode(s *registry.Service, node *registry.Node,
	opts ...registry.RegisterOption,
) (err error) {
	var options registry.RegisterOptions
	for _, o := range opts {
		o(&options)
	}

	if err = e.client.Error(); err != nil {
		return err
	}
	if len(s.Nodes) == 0 {
		return errors.New("require at least one node")
	}

	// check existing lease cache
	e.RLock()
	leaseID, ok := e.leases[s.Name+node.Id]
	e.RUnlock()

	// missing lease, check if the key exists
	if !ok {
		// look for the existing key
		var rsp *clientv3.GetResponse
		err = e.call(options.Context, func(ctx context.Context, c eetcd.Client) (err error) {
			rsp, err = c.Get(ctx, e.nodePath(s.Name, node.Id), clientv3.WithSerializable())
			return
		})
		if err != nil {
			return err
		}

		// get the existing lease
		for _, kv := range rsp.Kvs {
			if kv.Lease == 0 {
				continue
			}
			leaseID = clientv3.LeaseID(kv.Lease)

			// decode the existing node
			srv, err := e.decodeService(kv.Value)
			if err != nil || srv == nil || len(srv.Nodes) == 0 {
				continue
			}

			// create hash of service; uint64
			d, _ := json.Marshal(srv.Nodes[0])
			h := fmt.Sprintf("%x", d)

			// save the info
			e.Lock()
			e.leases[s.Name+node.Id] = leaseID
			e.register[s.Name+node.Id] = h
			e.Unlock()
		}
	}

	// renew the lease if it exists
	var leaseNotFound bool
	if leaseID > 0 {
		if logger.V(logger.TraceLevel, logger.DefaultLogger) {
			logger.Tracef("Renewing existing lease for %s %d", s.Name, leaseID)
		}
		err = e.call(options.Context, func(ctx context.Context, c eetcd.Client) (err error) {
			_, err = c.KeepAliveOnce(ctx, leaseID)
			return
		})
		if err != nil {
			if err != rpctypes.ErrLeaseNotFound {
				return err
			}

			if logger.V(logger.TraceLevel, logger.DefaultLogger) {
				logger.Tracef("Lease not found for %s %d", s.Name, leaseID)
			}
			// lease not found do register
			leaseNotFound = true
		}
	}

	// create hash of service; uint64
	d, _ := json.Marshal(node)
	h := fmt.Sprintf("%x", d)

	// get existing hash for the service node
	e.Lock()
	v, ok := e.register[s.Name+node.Id]
	e.Unlock()

	// the service is unchanged, skip registering
	if ok && v == h && !leaseNotFound {
		if logger.V(logger.TraceLevel, logger.DefaultLogger) {
			logger.Tracef("Service %s node %s unchanged skipping registration", s.Name, node.Id)
		}
		return nil
	}

	service := &registry.Service{
		Name:      s.Name,
		Version:   s.Version,
		Metadata:  s.Metadata,
		Endpoints: s.Endpoints,
		Nodes:     []*registry.Node{node},
	}
	serviceData, err := e.encodeService(service)
	if err != nil {
		return err
	}

	// get a lease used to expire keys since we have a ttl
	var lgr *clientv3.LeaseGrantResponse
	if options.TTL.Seconds() > 0 {
		err = e.call(options.Context, func(ctx context.Context, c eetcd.Client) (err error) {
			lgr, err = c.Grant(ctx, int64(options.TTL.Seconds()))
			return
		})
		if err != nil {
			return err
		}
	}

	if logger.V(logger.TraceLevel, logger.DefaultLogger) {
		logger.Tracef("Registering %s id %s with lease %v and leaseID %v and ttl %v", service.Name, node.Id, lgr, lgr.ID, options.TTL)
	}

	// create an entry for the node
	var putOpts []clientv3.OpOption
	if lgr != nil {
		putOpts = append(putOpts, clientv3.WithLease(lgr.ID))
	}
	err = e.call(options.Context, func(ctx context.Context, c eetcd.Client) (err error) {
		_, err = c.Put(ctx, e.nodePath(service.Name, node.Id), serviceData, putOpts...)
		return err
	})
	if err != nil {
		return err
	}

	e.Lock()
	// save our hash of the service
	e.register[s.Name+node.Id] = h
	// save our leaseID of the service
	if lgr != nil {
		e.leases[s.Name+node.Id] = lgr.ID
	}
	e.Unlock()
	return nil
}

func (e *etcdRegistry) Deregister(s *registry.Service, opts ...registry.DeregisterOption) (err error) {
	var options registry.DeregisterOptions
	for _, o := range opts {
		o(&options)
	}
	if err = e.client.Error(); err != nil {
		return err
	}
	if len(s.Nodes) == 0 {
		return errors.New("require at least one node")
	}

	for _, node := range s.Nodes {
		e.Lock()
		// delete our hash of the service
		delete(e.register, s.Name+node.Id)
		// delete our lease of the service
		delete(e.leases, s.Name+node.Id)
		e.Unlock()

		if logger.V(logger.TraceLevel, logger.DefaultLogger) {
			logger.Tracef("Unregistering %s id %s", s.Name, node.Id)
		}

		err = e.call(options.Context, func(ctx context.Context, c eetcd.Client) (err error) {
			_, err = c.Delete(ctx, e.nodePath(s.Name, node.Id))
			return
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *etcdRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) (lastErr error) {
	if len(s.Nodes) == 0 {
		return errors.New("require at least one node")
	}
	// register each node individually
	for _, node := range s.Nodes {
		lastErr = e.registerNode(s, node, opts...)
	}
	return lastErr
}

func (e *etcdRegistry) GetService(name string, opts ...registry.GetOption) (out []*registry.Service, err error) {
	var options registry.GetOptions
	for _, o := range opts {
		o(&options)
	}

	var rsp *clientv3.GetResponse
	err = e.call(options.Context, func(ctx context.Context, c eetcd.Client) (err error) {
		rsp, err = c.Get(ctx, e.servicePath(name)+"/", clientv3.WithPrefix(), clientv3.WithSerializable())
		return
	})
	if err != nil {
		return nil, err
	}
	if len(rsp.Kvs) == 0 {
		return nil, registry.ErrNotFound
	}

	// merge service by name & version
	serviceMap := map[string]*registry.Service{}
	for _, n := range rsp.Kvs {
		var sn *registry.Service
		if sn, err = e.decodeService(n.Value); err != nil || sn == nil {
			continue
		}

		s, ok := serviceMap[sn.Version]
		if !ok {
			s = &registry.Service{
				Name:      sn.Name,
				Version:   sn.Version,
				Metadata:  sn.Metadata,
				Endpoints: sn.Endpoints,
			}
			serviceMap[s.Version] = s
		}
		s.Nodes = append(s.Nodes, sn.Nodes...)
	}

	services := make([]*registry.Service, 0, len(serviceMap))
	for _, service := range serviceMap {
		services = append(services, service)
	}
	return services, nil
}

func (e *etcdRegistry) ListServices(opts ...registry.ListOption) (out []*registry.Service, err error) {
	var options registry.ListOptions
	for _, o := range opts {
		o(&options)
	}

	var rsp *clientv3.GetResponse
	err = e.call(options.Context, func(ctx context.Context, c eetcd.Client) (err error) {
		rsp, err = c.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithSerializable())
		return err
	})
	if err != nil {
		return nil, err
	}
	if len(rsp.Kvs) == 0 {
		return nil, nil
	}

	versions := make(map[string]*registry.Service)
	for _, n := range rsp.Kvs {
		sn, err := e.decodeService(n.Value)
		if sn == nil || err != nil {
			continue
		}
		v, ok := versions[sn.Name+sn.Version]
		if !ok {
			versions[sn.Name+sn.Version] = sn
			continue
		}
		// append to service:version nodes
		v.Nodes = append(v.Nodes, sn.Nodes...)
	}

	services := make([]*registry.Service, 0, len(versions))
	for _, service := range versions {
		services = append(services, service)
	}

	// sort the services
	sort.Slice(services, func(i, j int) bool { return services[i].Name < services[j].Name })
	return services, nil
}

func (e *etcdRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	return newEtcdWatcher(e.baseEtcd, opts...)
}

func (e *etcdRegistry) String() string {
	return "etcd"
}
