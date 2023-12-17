package zeebe

import (
	"context"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	log "github.com/sirupsen/logrus"
)

const APP_ID = "app_id"
const MESSAGE = "message"

func NewClient(addr string) zbc.Client {
	client, err := zbc.NewClient(&zbc.ClientConfig{
		GatewayAddress:         addr,
		UsePlaintextConnection: true,
	})
	if err != nil {
		panic(err)
	}
	return client
}

func DeployProcessDefinition(client zbc.Client, processID string) {
	ctx := context.Background()
	response, err := client.NewDeployResourceCommand().
		AddResourceFile(processID + ".bpmn").
		Send(ctx)
	if err != nil {
		panic(err)
	}
	log.Printf("Process definitions deployed: %v", response.GetDeployments())
}
