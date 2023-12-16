package main

import (
	"context"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"github.com/dddpaul/go-zeebe-example/pkg/handlers"
	"github.com/dddpaul/go-zeebe-example/pkg/workers"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
)

const ZeebeBrokerAddr = "192.168.0.100:26500"

func NewZeebeClient() zbc.Client {
	zbClient, err := zbc.NewClient(&zbc.ClientConfig{
		GatewayAddress:         ZeebeBrokerAddr,
		UsePlaintextConnection: true,
	})
	if err != nil {
		log.Fatalf("Failed to create Zeebe client: %v", err)
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
	log.Printf("Process definition deployed %s", response.String())
}

func main() {
	r := chi.NewRouter()

	zbClient := NewZeebeClient()
	defer func(z zbc.Client) {
		err := z.Close()
		if err != nil {
			panic(err)
		}
	}(zbClient)

	// Deploy process and start job workers
	DeployProcessDefinition(zbClient)
	go workers.StartJobWorkers(zbClient)

	r.Post("/sync", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("/sync request=%v", r)
		handlers.Sync(zbClient, w, r)
	})

	r.Post("/callback", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("/callback request=%v", r)
		handlers.Callback(zbClient, w, r)
	})

	port := ":8080"
	log.Printf("Start HTTP service on port %s with Zeebe broker %s", port, ZeebeBrokerAddr)
	err := http.ListenAndServe(port, r)
	if err != nil {
		panic(err)
	}
}
