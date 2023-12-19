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

func StartProcess(ctx context.Context, zbClient zbc.Client, zbProcessID, id string) (int64, error) {
	cmd, _ := zbClient.NewCreateInstanceCommand().
		BPMNProcessId(zbProcessID).
		LatestVersion().
		VariablesFromMap(map[string]interface{}{
			APP_ID: id,
		})
	resp, err := cmd.Send(ctx)
	if err != nil {
		return 0, err
	}
	return resp.GetProcessInstanceKey(), nil
}

func StartProcessWithResult(ctx context.Context, zbClient zbc.Client, zbProcessID, id string) (int64, string, error) {
	cmd, _ := zbClient.NewCreateInstanceCommand().
		BPMNProcessId(zbProcessID).
		LatestVersion().
		VariablesFromMap(map[string]interface{}{
			APP_ID: id,
		})
	resp, err := cmd.WithResult().Send(ctx)
	if err != nil {
		return 0, "", err
	}
	return resp.GetProcessInstanceKey(), resp.GetVariables(), nil
}
