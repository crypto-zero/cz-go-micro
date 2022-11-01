package tests

import (
	"os"
	"sort"
	"strings"
	"testing"

	"c-z.dev/go-micro/registry"
	etcdregistry "c-z.dev/go-micro/registry/etcd"
	"github.com/stretchr/testify/suite"
)

type EtcdRegistryTestSuite struct {
	EtcdClusterTestSuite
	registry registry.Registry
}

func (ert *EtcdRegistryTestSuite) SetupSuite() {
	ert.EtcdClusterTestSuite.SetupSuite(3)
	ert.SetupRegistry()
}

func (ert *EtcdRegistryTestSuite) SetupRegistry() {
	ert.registry = etcdregistry.NewRegistry(
		registry.Addrs("localhost:2370"),
	)
}

func (ert *EtcdRegistryTestSuite) TestRegistry() {
	newService := func(name, version, nodeID, address string) *registry.Service {
		return &registry.Service{
			Name:      name,
			Version:   version,
			Metadata:  nil,
			Endpoints: nil,
			Nodes: []*registry.Node{
				{
					Id:       nodeID,
					Address:  address,
					Metadata: nil,
				},
			},
		}
	}
	version := "1.0"
	serviceAName := "test-service/a"
	serviceANodes := []struct{ ID, Address string }{
		{"100", "127.0.0.1:100"},
		{"101", "127.0.0.1:101"},
		{"102", "127.0.0.1:102"},
	}
	for idx, n := range serviceANodes {
		service := newService(serviceAName, version, n.ID, n.Address)
		err := ert.registry.Register(service)
		ert.NoErrorf(err, "register service a [%d] failed", idx)
	}

	services, err := ert.registry.GetService(serviceAName)
	ert.NoError(err, "get services a failed")

	ert.Equal(1, len(services), "get services a size not equal")

	serviceA := services[0]
	sort.Slice(serviceA.Nodes, func(i, j int) bool {
		return strings.Compare(serviceA.Nodes[i].Id, serviceA.Nodes[j].Id) < 0
	})

	ert.Equal(len(serviceA.Nodes), len(serviceANodes), "get service a node size not equal")

	for idx, n := range serviceA.Nodes {
		expected := serviceANodes[idx]
		ert.Equal(expected.ID, n.Id, "service a node [%d] id not equal", idx)
		ert.Equal(expected.Address, n.Address, "service a node [%d] address not equal", idx)
	}

	services, err = ert.registry.ListServices()
	ert.NoError(err, "list services a failed")
	ert.Equal(1, len(services), "list services a size not equal")

	w, err := ert.registry.Watch()
	ert.NoError(err, "watch services a failed")

	serviceA = newService(serviceAName, version, "110", "127.0.0.1:110")
	err = ert.registry.Register(serviceA)
	ert.NoError(err, "add another service a failed")

	event, err := w.Next()
	ert.NoError(err, "watch services next add event failed")
	ert.Equal(true, event.IsCreate(), "watch service new event is not create event")
	ert.Equal(1, len(event.Service.Nodes), "watch service new add event node size is not 1")
	node, expectedNode := event.Service.Nodes[0], serviceA.Nodes[0]
	ert.Equal(node.Id, expectedNode.Id, "watch service add event node id not equal")
	ert.Equal(node.Address, expectedNode.Address, "watch service add event node address not equal")

	err = ert.registry.Deregister(serviceA)
	ert.NoError(err, "remove another service a failed")

	event, err = w.Next()
	ert.NoError(err, "watch services next remove event failed")
	ert.Equal(true, event.IsDelete(), "watch service new event is not delete event")
	ert.Equal(1, len(event.Service.Nodes), "watch service new remove event node size is not 1")
	node, expectedNode = event.Service.Nodes[0], serviceA.Nodes[0]
	ert.Equal(node.Id, expectedNode.Id, "watch service remove event node id not equal")
	ert.Equal(node.Address, expectedNode.Address, "watch service remove event node address not equal")

	w.Stop()

	for idx, n := range serviceANodes {
		service := newService(serviceAName, version, n.ID, n.Address)
		err := ert.registry.Deregister(service)
		ert.NoErrorf(err, "deregister service a [%d] failed", idx)
	}

	services, err = ert.registry.GetService(serviceAName)
	ert.ErrorIs(err, registry.ErrNotFound, "get services a should be not found")

	ert.Equal(0, len(services), "get services a size should be zero")
}

func TestEtcdRegistry(t *testing.T) {
	if os.Getenv("ENABLE_DOCKER_TEST") == "" {
		t.Skip("skip etcd registry tests.")
		return
	}
	suite.Run(t, new(EtcdRegistryTestSuite))
}
