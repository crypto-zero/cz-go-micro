package tests

import (
	"testing"

	"c-z.dev/go-micro/client"
	cgrpc "c-z.dev/go-micro/client/grpc"
	staticselector "c-z.dev/go-micro/client/selector/static"
	"c-z.dev/go-micro/metadata"
	"c-z.dev/go-micro/proxy"
	"c-z.dev/go-micro/proxy/mucp"
	memregistry "c-z.dev/go-micro/registry/memory"
	"c-z.dev/go-micro/router"
	"c-z.dev/go-micro/server"
	sgrpc "c-z.dev/go-micro/server/grpc"
	"c-z.dev/go-micro/tests/proto"
	"github.com/stretchr/testify/suite"
)

type ProxyGRPCTestSuite struct {
	GRPCTestSuite
	router      router.Router
	proxy       proxy.Proxy
	proxyServer server.Server
	proxyClient client.Client
}

func (pgs *ProxyGRPCTestSuite) SetupSuite() {
	pgs.GRPCTestSuite.SetupSuite()
	pgs.SetupRouter()
	pgs.StartRouter()
	pgs.SetupProxy()
	pgs.StartProxy()
}

func (pgs *ProxyGRPCTestSuite) TearDownSuite() {
	pgs.StopProxy()
	pgs.StopRouter()
	pgs.GRPCTestSuite.TearDownSuite()
}

func (pgs *ProxyGRPCTestSuite) SetupRouter() {
	pgs.router = router.NewRouter(router.Registry(pgs.registry))
}

func (pgs *ProxyGRPCTestSuite) StartRouter() {
	err := pgs.router.Start()
	pgs.NoError(err, "start router failed")
}

func (pgs *ProxyGRPCTestSuite) StopRouter() {
	err := pgs.router.Stop()
	pgs.NoError(err, "stop router failed")
}

func (pgs *ProxyGRPCTestSuite) SetupProxy() {
	pgs.proxy = mucp.NewProxy(
		proxy.WithRouter(pgs.router),
		proxy.WithClient(cgrpc.NewClient()),
	)
	pgs.proxyServer = sgrpc.NewServer(
		server.Name("mucp-proxy-server"),
		server.Address("127.0.0.1:9877"),
		server.Registry(memregistry.NewRegistry()),
		server.WithRouter(pgs.proxy),
	)
	pgs.proxyClient = cgrpc.NewClient(
		client.Registry(memregistry.NewRegistry()),
		client.Selector(staticselector.NewSelector(staticselector.WithStaticAddress("127.0.0.1:9877"))),
	)
}

func (pgs *ProxyGRPCTestSuite) StartProxy() {
	err := pgs.proxyServer.Start()
	pgs.NoError(err, "start proxy server failed")
}

func (pgs *ProxyGRPCTestSuite) StopProxy() {
	err := pgs.proxyServer.Stop()
	pgs.NoError(err, "stop proxy server failed")
}

func (pgs *ProxyGRPCTestSuite) TestProxyBasicRequest() {
	ctx := metadata.NewContext(pgs.testContext, map[string]string{"Connection": "keep-alive"})
	in := &proto.HelloRequest{Name: "TestBasicRequest"}
	req := pgs.proxyClient.NewRequest(pgs.serverName, "ExampleSrv.Hello", in)
	out := new(proto.HelloReply)
	err := pgs.proxyClient.Call(ctx, req, out)
	pgs.NoError(err, "call proxy hello failed")
	pgs.Equal("welcome: TestBasicRequest", out.Welcome, "call hello reply not equal")
}

func TestProxyGRPCSuite(t *testing.T) {
	suite.Run(t, new(ProxyGRPCTestSuite))
}
