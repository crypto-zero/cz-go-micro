package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"c-z.dev/go-micro/client"
	cgrpc "c-z.dev/go-micro/client/grpc"
	"c-z.dev/go-micro/client/selector"
	"c-z.dev/go-micro/metadata"
	"c-z.dev/go-micro/registry"
	memregistry "c-z.dev/go-micro/registry/memory"
	"c-z.dev/go-micro/server"
	sgrpc "c-z.dev/go-micro/server/grpc"
	"c-z.dev/go-micro/tests/proto"

	"github.com/stretchr/testify/suite"
)

type GRPCExampleHandler struct{}

func (G *GRPCExampleHandler) Hello(ctx context.Context, request *proto.HelloRequest, reply *proto.HelloReply) error {
	reply.Welcome, reply.Time = fmt.Sprintf("welcome: %s", request.Name), time.Now().UnixMilli()
	return nil
}

func (G *GRPCExampleHandler) HelloStreamRequestX(ctx context.Context, stream proto.ExampleSrv_HelloStreamRequestXStream) error {
	for {
		in, err := stream.Recv()
		if err != nil {
			return err
		}
		err = stream.SendMsg(&proto.HelloStreamReply{
			Time:    in.Time,
			Content: fmt.Sprintf("hello stream request x: %s", in.Content),
		})
		if err != nil {
			return err
		}
	}
}

func (G *GRPCExampleHandler) HelloStreamReplyX(ctx context.Context, request *proto.HelloStreamRequest, stream proto.ExampleSrv_HelloStreamReplyXStream) error {
	for i := 0; i < 3; i++ {
		err := stream.SendMsg(&proto.HelloStreamReply{
			Time:    request.Time,
			Content: fmt.Sprintf("hello stream reply x: %s", request.Content),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (G *GRPCExampleHandler) HelloStreamRequestReply(ctx context.Context, stream proto.ExampleSrv_HelloStreamRequestReplyStream) error {
	for {
		in, err := stream.Recv()
		if err != nil {
			return err
		}
		err = stream.SendMsg(&proto.HelloStreamReply{
			Time:    in.Time,
			Content: fmt.Sprintf("hello stream request reply: %s", in.Content),
		})
		if err != nil {
			return err
		}
	}
}

type GRPCTestSuite struct {
	suite.Suite

	testContext context.Context

	registry registry.Registry
	server   server.Server
	client   client.Client

	serverName string
}

func (gs *GRPCTestSuite) SetupSuite() {
	gs.SetupContext()
	gs.SetupRegistry()
	gs.SetupGRPCServerClient()
	gs.StartGRPCServer()
}

func (gs *GRPCTestSuite) TearDownSuite() {
	gs.StopGRPCServer()
}

func (gs *GRPCTestSuite) SetupContext() {
	gs.testContext = context.Background()
}

func (gs *GRPCTestSuite) SetupRegistry() {
	gs.registry = memregistry.NewRegistry()
	err := gs.registry.Init()
	gs.NoError(err, "initial memory registry failed")
}

func (gs *GRPCTestSuite) SetupGRPCServerClient() {
	gs.serverName = "grpc-server-test"

	gs.server = sgrpc.NewServer()
	gs.client = cgrpc.NewClient()

	err := gs.server.Init(
		server.Name(gs.serverName),
		server.Registry(gs.registry),
		server.Address("127.0.0.1:0"),
	)
	gs.NoError(err, "initial grpc server failed")

	err = proto.RegisterExampleSrvHandler(gs.server, new(GRPCExampleHandler))
	gs.NoError(err, "set grpc server handler failed")

	err = gs.client.Init(
		client.Registry(gs.registry),
		client.Selector(selector.NewSelector(selector.Registry(gs.registry))),
	)
	gs.NoError(err, "initial grpc client failed")
}

func (gs *GRPCTestSuite) StartGRPCServer() {
	err := gs.server.Start()
	gs.NoError(err, "start grpc server failed")
}

func (gs *GRPCTestSuite) StopGRPCServer() {
	err := gs.server.Stop()
	gs.NoError(err, "stop grpc server failed")
}

func (gs *GRPCTestSuite) TestBasicRequest() {
	ctx := metadata.NewContext(gs.testContext, map[string]string{"Connection": "keep-alive"})
	in := &proto.HelloRequest{Name: "TestBasicRequest"}
	req := gs.client.NewRequest(gs.serverName, "ExampleSrv.Hello", in)
	out := new(proto.HelloReply)
	err := gs.client.Call(ctx, req, out)
	gs.NoError(err, "call hello failed")
	gs.Equal("welcome: TestBasicRequest", out.Welcome, "call hello reply not equal")
}

func (gs *GRPCTestSuite) TestBasicRequestContentType() {
	ctx := metadata.NewContext(gs.testContext, map[string]string{"Connection": "keep-alive"})
	in := &proto.HelloRequest{Name: "TestBasicRequest"}
	req := gs.client.NewRequest(gs.serverName, "ExampleSrv.Hello", in, client.WithContentType("application/json"))
	out := new(proto.HelloReply)
	err := gs.client.Call(ctx, req, out)
	gs.NoError(err, "call hello failed")
	gs.Equal("welcome: TestBasicRequest", out.Welcome, "call hello reply not equal")
}

func TestGRPCSuite(t *testing.T) {
	suite.Run(t, new(GRPCTestSuite))
}
