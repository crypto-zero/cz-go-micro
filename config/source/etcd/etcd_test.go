package etcd

import (
	"context"
	"os"
	"testing"

	cetcd "go.etcd.io/etcd/client/v3"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestETCDGet(t *testing.T) {
	etcdHost := os.Getenv("ETCD_HOST")
	if etcdHost == "" {
		t.Skip("no etcd host set. skip etcd test")
		return
	}

	cfg := cetcd.Config{Endpoints: []string{etcdHost}}
	c, err := cetcd.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Get(context.Background(), "/", cetcd.WithPrefix())
	if err != nil {
		t.Fatal(err)
	}
	t.Log("etcd get test ok.")
}
