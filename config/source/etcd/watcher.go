package etcd

import (
	"context"
	"errors"
	"time"

	"c-z.dev/go-micro/config/source"
	eetcd "c-z.dev/go-micro/extension/etcd"
	"c-z.dev/go-micro/logger"
	cetcd "go.etcd.io/etcd/client/v3"
)

type changeSetReader interface {
	Read() (*source.ChangeSet, error)
}

type watcher struct {
	ctx    context.Context
	cancel context.CancelFunc

	opts        source.Options
	name        string
	stripPrefix string

	cs *source.ChangeSet
	ch chan *source.ChangeSet
}

func newWatcher(ctx context.Context, reader changeSetReader, key, strip string, c eetcd.Client,
	cs *source.ChangeSet, opts source.Options,
) (source.Watcher, error) {
	ctx, cancel := context.WithCancel(ctx)
	w := &watcher{
		ctx:         ctx,
		cancel:      cancel,
		opts:        opts,
		name:        "etcd",
		stripPrefix: strip,
		cs:          cs,
		ch:          make(chan *source.ChangeSet),
	}
	c.Watch(ctx, key, func(events []*cetcd.Event, err error) {
		if events != nil {
			w.handle(events)
		}
		if err == nil {
			return
		}
		cs, xerr := reader.Read()
		if xerr == nil {
			w.cs = cs
			w.ch <- cs
		}
		if xerr != nil && logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Errorf("etcd watcher key: %s reload change set error: %v", key, xerr)
		}
	}, cetcd.WithPrefix())
	return w, nil
}

func (w *watcher) handle(evs []*cetcd.Event) {
	data := w.cs.Data

	var vals map[string]interface{}

	// unpackage existing changeset
	if err := w.opts.Encoder.Decode(data, &vals); err != nil {
		return
	}

	// update base changeset
	d := makeEvMap(w.opts.Encoder, vals, evs, w.stripPrefix)

	// pack the changeset
	b, err := w.opts.Encoder.Encode(d)
	if err != nil {
		return
	}

	// create new changeset
	cs := &source.ChangeSet{
		Timestamp: time.Now(),
		Source:    w.name,
		Data:      b,
		Format:    w.opts.Encoder.String(),
	}
	cs.Checksum = cs.Sum()

	// set base change set
	w.cs = cs

	// send update
	w.ch <- cs
}

func (w *watcher) Next() (*source.ChangeSet, error) {
	select {
	case cs := <-w.ch:
		return cs, nil
	case <-w.ctx.Done():
		return nil, errors.New("watcher stopped")
	}
}

func (w *watcher) Stop() error {
	w.cancel()
	return nil
}
