package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/dddpaul/go-zeebe-example/pkg/service"
	"github.com/google/uuid"
	"github.com/phayes/freeport"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	wait "github.com/testcontainers/testcontainers-go/wait"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func Test_ZeebeStart(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	zbBrokerAddr, teardown := startTestContainer(t)
	defer teardown()

	// TODO: Replace with wait-for-port
	time.Sleep(3 * time.Second)

	zbProcessID := "diagram_1"
	port, err := freeport.GetFreePort()
	require.NoError(t, err)

	s := service.New(
		service.WithHttpPort(":"+strconv.Itoa(port)),
		service.WithZeebe(zbBrokerAddr, zbProcessID))
	go s.Start()

	// TODO: Replace with wait-for-port and deployed process
	time.Sleep(3 * time.Second)

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

	t.Run("should create process on /sync call", func(t *testing.T) {
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

		// TODO: Replace with wait-for-log
		time.Sleep(1 * time.Second)

		// and
		url = "http://127.0.0.1:" + strconv.Itoa(port) + "/callback"
		req, _ = http.NewRequest("POST", url, bytes.NewReader([]byte("{ \"message\" : \""+message+"\" }")))
		req.Header.Set("X-APP-ID", id)
		_, err = client.Do(req)
		require.NoError(t, err)

		// then
		<-done
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		fmt.Println(resp)
	})

	// TODO: Replace with wait-process-end
	time.Sleep(3 * time.Second)
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
