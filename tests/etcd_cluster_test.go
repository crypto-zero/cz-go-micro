package tests

import (
	"context"
	"fmt"
	"strings"
	"time"

	dt "github.com/ory/dockertest/v3"
	dc "github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/suite"
	cetcd "go.etcd.io/etcd/client/v3"
)

type EtcdClusterTestSuite struct {
	suite.Suite

	pool *dt.Pool

	etcdClusterSize int

	etcdNetwork   *dt.Network
	etcdResources []*dt.Resource

	etcd1, etcd2 *cetcd.Client

	testKey, testContent string
}

func (ect *EtcdClusterTestSuite) SetupSuite() {
	ect.etcdClusterSize = 5

	pool, err := dt.NewPool("")
	ect.NoError(err, "docker test build pool failed")
	ect.pool = pool

	ect.SetupEtcdCluster()
	ect.SetupEtcdContent()
}

func (ect *EtcdClusterTestSuite) TearDownSuite() {
	ect.CleanupEtcdCluster()
}

func (ect *EtcdClusterTestSuite) SetupEtcdCluster() {
	etcdContainerName := "micro-testsuite-etcd"
	containerNameFn := func(index int) string {
		return fmt.Sprintf("%s_%d", etcdContainerName, index)
	}

	for i := 0; i < ect.etcdClusterSize; i++ {
		err := ect.pool.RemoveContainerByName(containerNameFn(i))
		ect.NoErrorf(err, "delete etcd container [%d] failed", i)
	}

	existNetworks, err := ect.pool.NetworksByName(etcdContainerName)
	for _, n := range existNetworks {
		err = ect.pool.RemoveNetwork(&n)
		ect.NoError(err, "delete docker network failed")
	}
	ect.NoError(err, "find docker network failed")

	subnet := "172.18.0."
	network, err := ect.pool.CreateNetwork(etcdContainerName, func(config *dc.CreateNetworkOptions) {
		config.Driver = "bridge"
		config.EnableIPv6 = false
		config.IPAM = &dc.IPAMOptions{
			Driver: "default",
			Config: []dc.IPAMConfig{
				{
					Subnet:     subnet + "0/16",
					IPRange:    "",
					Gateway:    subnet + "1",
					AuxAddress: nil,
				},
			},
			Options: nil,
		}
	})
	ect.NoError(err, "create etcd cluster network failed")
	ect.etcdNetwork = network

	advertiseClientUrls := ""
	for i := 0; i < ect.etcdClusterSize; i++ {
		cn := containerNameFn(i)
		ip := fmt.Sprintf("%s%d", subnet, i+2)
		advertiseClientUrls += fmt.Sprintf("%s=http://%s:2380,", cn, ip)
	}
	advertiseClientUrls = strings.TrimSuffix(advertiseClientUrls, ",")

	port := 2370
	for i := 0; i < ect.etcdClusterSize; i++ {
		port := port + i
		portBindings := map[dc.Port][]dc.PortBinding{
			dc.Port(fmt.Sprintf("%d/tcp", port)): {{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", port)}},
		}
		resource, err := ect.pool.RunWithOptions(&dt.RunOptions{
			Name:       containerNameFn(i),
			Hostname:   containerNameFn(i),
			Repository: "ghcr.io/crypto-zero/etcd",
			Networks:   []*dt.Network{ect.etcdNetwork},
			Env: []string{
				fmt.Sprintf("ETCD_INITIAL_CLUSTER=%s", advertiseClientUrls),
				fmt.Sprintf("ETCD_INITIAL_ADVERTISE_PEER_URLS=http://%s:2380", fmt.Sprintf("%s%d", subnet, i+2)),
				fmt.Sprintf("ETCD_ADVERTISE_CLIENT_URLS=http://127.0.0.1:%d", port),
				fmt.Sprintf("ETCD_LISTEN_CLIENT_URLS=http://0.0.0.0:%d", port),
			},
			ExposedPorts: []string{fmt.Sprintf("%d/tcp", port)},
			Tag:          "v3.5.5",
			PortBindings: portBindings,
		}, func(config *dc.HostConfig) {
		})
		ect.NoError(err, "start test etcd [%d] failed", i)
		ect.etcdResources = append(ect.etcdResources, resource)
	}
}

func (ect *EtcdClusterTestSuite) CleanupEtcdCluster() {
	for idx, r := range ect.etcdResources {
		err := r.Close()
		ect.NoErrorf(err, "cleanup etcd cluster [%d] failed", idx)
	}
	err := ect.etcdNetwork.Close()
	ect.NoError(err, "cleanup etcd network failed")
}

func (ect *EtcdClusterTestSuite) SetupEtcdContent() {
	ect.testKey, ect.testContent = "/project/hello", `{"name": "hello"}`

	var err error
	ect.etcd1, err = cetcd.New(cetcd.Config{
		Endpoints:   []string{"localhost:2370"},
		DialTimeout: 3 * time.Second,
	})
	ect.NoError(err, "connect to etcd-1 failed")

	ect.etcd2, err = cetcd.New(cetcd.Config{
		Endpoints:   []string{"localhost:2371"},
		DialTimeout: 3 * time.Second,
	})
	ect.NoError(err, "connect to etcd-2 failed")

	ctx := context.Background()
	err = ect.pool.Retry(func() error {
		if _, err = ect.etcd1.Put(ctx, ect.testKey, ect.testContent); err != nil {
			return err
		}
		return nil
	})
	ect.NoError(err, "write etcd key/value failed")
}
