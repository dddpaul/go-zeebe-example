package zeebe

import (
	"context"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	log "github.com/sirupsen/logrus"
)

const APP_ID = "app_id"
const MESSAGE = "message"

func NewClient(zeebeBrokerAddr string) zbc.Client {
	zbClient, err := zbc.NewClient(&zbc.ClientConfig{
		GatewayAddress:         zeebeBrokerAddr,
		UsePlaintextConnection: true,
	})
	if err != nil {
		panic(err)
	}
	return zbClient
}

func DeployProcessDefinition(client zbc.Client) {
	ctx := context.Background()
	response, err := client.NewDeployResourceCommand().
		AddResourceFile("diagram_1.bpmn").
		Send(ctx)
	if err != nil {
		panic(err)
	}
	log.Printf("Process definitions deployed: %v", response.GetDeployments())
}
