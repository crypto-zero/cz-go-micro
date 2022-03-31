package wrapper_test

import (
	"context"
	"testing"

	"c-z.dev/go-micro/broker"
	bmemory "c-z.dev/go-micro/broker/memory"
	"c-z.dev/go-micro/client"
	rmemory "c-z.dev/go-micro/registry/memory"
	"c-z.dev/go-micro/server"
	tmemory "c-z.dev/go-micro/transport/memory"
	"c-z.dev/go-micro/util/wrapper"
)

type TestFoo struct{}

type TestReq struct{}

type TestRsp struct {
	Data string
}

func (h *TestFoo) Bar(ctx context.Context, req *TestReq, rsp *TestRsp) error {
	rsp.Data = "pass"
	return nil
}

func TestStaticClientWrapper(t *testing.T) {
	var err error

	req := client.NewRequest("go.micro.service.foo", "TestFoo.Bar", &TestReq{}, client.WithContentType("application/json"))
	rsp := &TestRsp{}

	reg := rmemory.NewRegistry()
	brk := bmemory.NewBroker(broker.Registry(reg))
	tr := tmemory.NewTransport()

	srv := server.NewServer(
		server.Broker(brk),
		server.Registry(reg),
		server.Name("go.micro.service.foo"),
		server.Address("127.0.0.1:0"),
		server.Transport(tr),
	)
	if err = srv.Handle(srv.NewHandler(&TestFoo{})); err != nil {
		t.Fatal(err)
	}

	if err = srv.Start(); err != nil {
		t.Fatal(err)
	}

	cli := client.NewClient(
		client.Registry(reg),
		client.Broker(brk),
		client.Transport(tr),
	)

	w1 := wrapper.StaticClient("xxx_localhost:12345", cli)
	if err = w1.Call(context.TODO(), req, nil); err == nil {
		t.Fatal("address xxx_#localhost:12345 must not exists and call must be failed")
	}

	w2 := wrapper.StaticClient(srv.Options().Address, cli)
	if err = w2.Call(context.TODO(), req, rsp); err != nil {
		t.Fatal(err)
	} else if rsp.Data != "pass" {
		t.Fatalf("something wrong with response: %#+v", rsp)
	}
}
