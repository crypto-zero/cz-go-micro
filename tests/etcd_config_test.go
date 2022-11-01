package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"c-z.dev/go-micro/config"
	"c-z.dev/go-micro/config/source/etcd"

	"github.com/stretchr/testify/suite"
)

type EtcdConfigTestSuite struct {
	EtcdClusterTestSuite
	config config.Config
}

func (ect *EtcdConfigTestSuite) SetupSuite() {
	ect.EtcdClusterTestSuite.SetupSuite()
	ect.SetupConfig()
}

func (ect *EtcdConfigTestSuite) SetupConfig() {
	cf, err := config.NewConfig(
		config.WithSource(etcd.NewSource(
			etcd.WithAddress("localhost:2370"),
			etcd.WithPrefix("/project"),
		)),
	)
	ect.NoError(err, "initial etcd config source failed")
	ect.config = cf
}

func (ect *EtcdConfigTestSuite) TestConfigSource() {
	expected := "hello"
	ctx := context.Background()

	value := ect.config.Get("project", "hello", "name").String("")
	ect.Equal(expected, value, "config read key value failed")

	err := ect.pool.Client.StopContainer(ect.etcdResources[0].Container.ID, 0)
	ect.NoError(err, "temporary stop etcd failed")

	time.Sleep(1 * time.Second)

	revision := int64(0)
	for j := 0; j < 100; j++ {
		newValue := fmt.Sprintf(`{"name": "hello1%d"}`, j)
		rsp, err := ect.etcd2.Put(ctx, ect.testKey, newValue)
		ect.NoError(err, "write value to etcd2 failed")
		revision = rsp.Header.Revision
	}
	_, err = ect.etcd2.Compact(ctx, revision)
	ect.NoError(err, "compact from etcd2 failed")

	time.Sleep(2 * time.Second)

	newValue, expected := `{"name": "hello1"}`, "hello1"
	_, err = ect.etcd2.Put(ctx, ect.testKey, newValue)
	ect.NoError(err, "write new value to etcd failed")

	err = ect.pool.Client.StartContainer(ect.etcdResources[0].Container.ID, nil)
	ect.NoError(err, "temporary start etcd failed")

	time.Sleep(2 * time.Second)

	value = ect.config.Get("project", "hello", "name").String("")
	ect.Equal(expected, value, "config read key value failed")
}

func TestEtcdConfigSource(t *testing.T) {
	if os.Getenv("ENABLE_DOCKER_TEST") == "" {
		t.Skip("skip etcd config source tests.")
		return
	}
	suite.Run(t, new(EtcdConfigTestSuite))
}
