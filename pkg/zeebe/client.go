package zeebe

import (
	"context"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	log "github.com/sirupsen/logrus"
)

// BPMN input and output variables
const (
	APP_ID  = "app_id"
	MESSAGE = "message"
)

// BPMN message references for message catch events
const (
	MESSAGE_CALLBACK = "callback"
)

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

func DeployProcessDefinition(client zbc.Client, processID string) error {
	ctx := context.Background()
	response, err := client.NewDeployResourceCommand().
		AddResourceFile(processID + ".bpmn").
		Send(ctx)
	if err != nil {
		return err
	}
	log.Printf("Process definitions deployed: %v", response.GetDeployments())
	return nil
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

func PublishCallbackMessage(ctx context.Context, zbClient zbc.Client, id, message string) error {
	cmd, _ := zbClient.NewPublishMessageCommand().
		MessageName(MESSAGE_CALLBACK).
		CorrelationKey(id).
		VariablesFromMap(map[string]interface{}{
			MESSAGE: message,
		})
	_, err := cmd.Send(ctx)
	return err
}
