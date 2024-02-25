package main

import (
	"context"
	"fmt"
	wait "github.com/antelman107/net-wait-go/wait"
	"github.com/dddpaul/go-zeebe-example/pkg/service"
	"github.com/google/uuid"
	"github.com/phayes/freeport"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	waitFor "github.com/testcontainers/testcontainers-go/wait"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func Test_Main(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	client := http.DefaultClient

	zbBrokerAddr, teardown := startTestContainer(t, false)
	defer teardown()

	port, err := freeport.GetFreePort()
	require.NoError(t, err)
	appHostAndPort := "127.0.0.1:" + strconv.Itoa(port)

	zbProcessID := "diagram_1"
	s := service.New(
		service.WithHttpPort(":"+strconv.Itoa(port)),
		service.WithZeebe(zbBrokerAddr, zbProcessID, 0, 0, false))
	go s.Start()

	if !wait.New(wait.WithDeadline(5 * time.Second)).Do([]string{appHostAndPort}) {
		panic("Application failed to start")
	}

	t.Run("should create process on /sync call and finish it on /callback call", func(t *testing.T) {
		// given
		id := uuid.NewString()
		message := "TEST"

		// when
		req := newPostRequest(t, appHostAndPort, service.SYNC_PATH, nil, id)
		var resp *http.Response
		done := make(chan bool)
		go func() {
			resp, err = client.Do(req)
			require.NoError(t, err)
			done <- true
		}()

		// and
		cbBody := strings.NewReader("{ \"message\" : \"" + message + "\" }")
		cbReq := newPostRequest(t, appHostAndPort, service.CALLBACK_PATH, cbBody, id)
		cbResp, err := client.Do(cbReq)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, cbResp.StatusCode)

		// then
		<-done
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		fmt.Println(string(body))
	})

	t.Run("should create process on /sync-with-result call and finish it on /callback call", func(t *testing.T) {
		// given
		id := uuid.NewString()
		message := "TEST"

		// when
		req := newPostRequest(t, appHostAndPort, service.SYNC_WITH_RESULT_PATH, nil, id)
		var resp *http.Response
		done := make(chan bool)
		go func() {
			resp, err = client.Do(req)
			require.NoError(t, err)
			done <- true
		}()

		// and
		cbBody := strings.NewReader("{ \"message\" : \"" + message + "\" }")
		cbReq := newPostRequest(t, appHostAndPort, service.CALLBACK_PATH, cbBody, id)
		cbResp, err := client.Do(cbReq)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, cbResp.StatusCode)

		// then
		<-done
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		fmt.Println(string(body))
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
		WaitingFor:   waitFor.NewLogStrategy("Partition-1 recovered, marking it as healthy").WithStartupTimeout(time.Second * 60),
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

	// To pass "Expected to execute command on partition 1, but either it does not exist, or the gateway is not yet aware of it"
	// Seems like there slight window of unavailability after "Partition-1 recovered, marking it as healthy" log message
	time.Sleep(1 * time.Second)

	return fmt.Sprintf("%s:%s", host, port.Port()), func() {
		if err := container.Terminate(ctx); err != nil {
			panic(err)
		}
	}
}

func newPostRequest(t *testing.T, appHostAndPort string, path string, body io.Reader, header string) *http.Request {
	url := "http://" + singleJoiningSlash(appHostAndPort, path)
	req, err := http.NewRequest("POST", url, body)
	require.NoError(t, err)
	req.Header.Set("X-APP-ID", header)
	return req
}

// Taken from net/http/httputil/reverseproxy.go
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
