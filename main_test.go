package main

import (
	"context"
	"fmt"
	"github.com/dddpaul/go-zeebe-example/pkg/zeebe"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	wait "github.com/testcontainers/testcontainers-go/wait"
	"testing"
	"time"
)

func Test_ZeebeStart(t *testing.T) {
	zbBrokerAddr, teardown := startTestContainer(t)
	defer teardown()

	t.Run("should deploy process", func(t *testing.T) {
		// given
		zbClient := zeebe.NewClient(zbBrokerAddr)
		zbProcessID := "diagram_1"

		// when
		err := zeebe.DeployProcessDefinition(zbClient, zbProcessID)

		// then
		require.NoError(t, err)
	})
}

func startTestContainer(t *testing.T) (hostAndPort string, teardown func()) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "camunda/zeebe:8.3.3",
		ExposedPorts: []string{"26500/tcp"},
		WaitingFor:   wait.NewLogStrategy("Partition-1 recovered, marking it as healthy").WithStartupTimeout(time.Second * 60),
		Env: map[string]string{
			"ZEEBE_BROKER_GATEWAY_ENABLE":      "true",
			"ZEEBE_BROKER_CLUSTER_CLUSTERSIZE": "1",
		},
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "26500")
	require.NoError(t, err)

	return fmt.Sprintf("%s:%s", host, port.Port()), func() {
		if err := container.Terminate(ctx); err != nil {
			panic(err)
		}
	}
}
