package etcd

import (
	"context"
	"errors"

	eetcd "c-z.dev/go-micro/extension/etcd"

	"c-z.dev/go-micro/registry"

	"go.etcd.io/etcd/client/v3"
)

type etcdWatcher struct {
	baseEtcd
	cancel context.CancelFunc

	w clientv3.WatchChan
}

func newEtcdWatcher(be *baseEtcd, opts ...registry.WatchOption) (out registry.Watcher, err error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	ctx := be.ctx
	if wo.Context != nil {
		ctx = wo.Context
	}
	ctx, cancel := context.WithCancel(ctx)

	watchPath := prefix
	if len(wo.Service) > 0 {
		watchPath = be.servicePath(wo.Service) + "/"
	}

	var wc clientv3.WatchChan
	err = be.call(ctx, func(_ context.Context, c eetcd.Client) (err error) {
		wc = c.Watcher.Watch(ctx, watchPath, clientv3.WithPrefix(), clientv3.WithPrevKV())
		return nil
	})
	ew := &etcdWatcher{
		baseEtcd: *be,
		cancel:   cancel,
		w:        wc,
	}
	ew.baseEtcd.ctx = ctx
	return ew, nil
}

func (ew *etcdWatcher) Next() (*registry.Result, error) {
	for wresp := range ew.w {
		if wresp.Err() != nil {
			return nil, wresp.Err()
		}
		if wresp.Canceled {
			return nil, errors.New("could not get next because canceled")
		}
		for _, ev := range wresp.Events {
			var err error
			var action string
			var service *registry.Service

			switch ev.Type {
			case clientv3.EventTypePut:
				if ev.IsCreate() {
					action = registry.ResultActionCreate
				} else if ev.IsModify() {
					action = registry.ResultActionUpdate
				}
				service, err = ew.decodeService(ev.Kv.Value)
			case clientv3.EventTypeDelete:
				// get service from prevKv
				action = registry.ResultActionDelete
				service, err = ew.decodeService(ev.PrevKv.Value)
			}
			if err != nil {
				continue
			}
			return &registry.Result{Action: action, Service: service}, nil
		}
	}
	return nil, errors.New("could not get next")
}

func (ew *etcdWatcher) Stop() {
	ew.cancel()
}
