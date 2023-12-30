package main

import (
	"bytes"
	"context"
	"fmt"
	wait_for "github.com/antelman107/net-wait-go/wait"
	"github.com/dddpaul/go-zeebe-example/pkg/service"
	"github.com/google/uuid"
	"github.com/phayes/freeport"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func Test_Main(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	zbBrokerAddr, teardown := startTestContainer(t, true)
	defer teardown()

	port, err := freeport.GetFreePort()
	require.NoError(t, err)
	appHostAndPort := "127.0.0.1:" + strconv.Itoa(port)

	zbProcessID := "diagram_1"
	s := service.New(
		service.WithHttpPort(":"+strconv.Itoa(port)),
		service.WithZeebe(zbBrokerAddr, zbProcessID))
	go s.Start()

	if !wait_for.New(wait_for.WithDeadline(5 * time.Second)).Do([]string{appHostAndPort}) {
		panic("Application failed to start")
	}

	t.Run("should create process on /sync call and finish it on /callback call", func(t *testing.T) {
		// given
		client := http.DefaultClient
		id := uuid.NewString()
		message := "TEST"

		// when
		url := "http://127.0.0.1:" + strconv.Itoa(port) + "/sync"
		req, _ := http.NewRequest("POST", url, nil)
		req.Header.Set("X-APP-ID", id)
		var resp *http.Response
		done := make(chan bool)
		go func() {
			resp, err = client.Do(req)
			require.NoError(t, err)
			done <- true
		}()

		// and
		url = "http://127.0.0.1:" + strconv.Itoa(port) + "/callback"
		cbReq, _ := http.NewRequest("POST", url, bytes.NewReader([]byte("{ \"message\" : \""+message+"\" }")))
		cbReq.Header.Set("X-APP-ID", id)
		resp, err = client.Do(cbReq)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// then
		<-done
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		fmt.Println(resp)
	})
}

//t.Run("should deploy process", func(t *testing.T) {
//	// given
//	zbClient := zeebe.NewClient(zbBrokerAddr)
//
//	// when
//	err := zeebe.DeployProcessDefinition(zbClient, zbProcessID)
//
//	// then
//	require.NoError(t, err)
//})

type ContainerLogConsumer struct {
}

func (c *ContainerLogConsumer) Accept(l testcontainers.Log) {
	fmt.Print(string(l.Content))
}

func startTestContainer(t *testing.T, loggingEnabled bool) (hostAndPort string, teardown func()) {
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

	if loggingEnabled {
		container.FollowOutput(&ContainerLogConsumer{})
		err = container.StartLogProducer(ctx)
		require.NoError(t, err)
	}

	return fmt.Sprintf("%s:%s", host, port.Port()), func() {
		if err := container.Terminate(ctx); err != nil {
			panic(err)
		}
	}
}
