package main

import (
	"context"
	"encoding/json"
	"github.com/camunda-cloud/zeebe/clients/go/pkg/entities"
	"github.com/camunda-cloud/zeebe/clients/go/pkg/worker"
	"github.com/camunda-cloud/zeebe/clients/go/pkg/zbc"
	"github.com/go-chi/chi/v5"
	"io"
	"log"
	"net/http"
	"time"
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
	_, err := client.NewDeployProcessCommand().
		AddResourceFile("BPMN").
		Send(ctx)
	if err != nil {
		panic(err)
	}
}

func StartJobWorkers(client zbc.Client) {
	jobWorker := client.NewJobWorker().JobType("service-task").Handler(handleJob).Open()
	defer jobWorker.Close()
	jobWorker.AwaitClose()
}

func handleJob(client worker.JobClient, job entities.Job) {
	// Here you would implement your business logic to handle the job.
	log.Printf("Handling job: %s", job.Type)

	// Complete the job with the result "Yes".
	request, err := client.NewCompleteJobCommand().JobKey(job.Key).VariablesFromMap(map[string]interface{}{
		"result": "Yes",
	})
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_, err = request.Send(ctx)
	if err != nil {
		panic(err)
	}

	log.Printf("Successfully completed job: %s", job.Type)
}

type StartProcessResponse struct {
	ProcessInstanceKey int64  `json:"processInstanceKey"`
	Result             string `json:"result"`
}

type CallbackRequest struct {
	ProcessInstanceKey int64                  `json:"processInstanceKey"`
	MessageName        string                 `json:"messageName"`
	CorrelationKey     string                 `json:"correlationKey"`
	Variables          map[string]interface{} `json:"variables"`
}

func main() {
	r := chi.NewRouter()

	zbClient := NewZeebeClient()
	defer zbClient.Close()

	// Deploy process
	DeployProcessDefinition(zbClient)

	// Start job workers
	go StartJobWorkers(zbClient)

	r.Post("/sync", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Start the process instance
		response, err := zbClient.NewCreateInstanceCommand().
			BPMNProcessId("CarInsuranceProcess").
			LatestVersion().
			Send(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Listen for the job worker result or the timeout
		resultChan := make(chan string, 1)
		go func() {
			jobWorker := zbClient.NewJobWorker().JobType("end-task").Handler(func(jobClient worker.JobClient, job entities.Job) {
				jobKey := job.GetKey()
				// Assuming that the job worker will complete the task with a variable "result"
				variables := map[string]interface{}{
					"result": "Yes",
				}
				request, err := jobClient.NewCompleteJobCommand().JobKey(jobKey).VariablesFromMap(variables)
				if err != nil {
					panic(err)
				}
				_, err = request.Send(ctx)
				if err != nil {
					panic(err)
				}
				resultChan <- "Done"
			}).Open()
			defer jobWorker.Close()
			jobWorker.AwaitClose()
		}()

		select {
		case result := <-resultChan:
			// If we got the result within the timeout, send it back
			err := json.NewEncoder(w).Encode(StartProcessResponse{ProcessInstanceKey: response.GetProcessInstanceKey(), Result: result})
			if err != nil {
				return
			}
		case <-ctx.Done():
			// If the context is done, it means we hit the timeout
			http.Error(w, "timeout waiting for the process to complete", http.StatusRequestTimeout)
		}
	})

	r.Post("/callback", func(w http.ResponseWriter, r *http.Request) {
		var callbackReq CallbackRequest

		// Read the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Unmarshal the JSON request body into the CallbackRequest struct
		err = json.Unmarshal(body, &callbackReq)
		if err != nil {
			http.Error(w, "failed to unmarshal request body", http.StatusBadRequest)
			return
		}

		// Publish a message to the process instance
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		command, err := zbClient.NewPublishMessageCommand().
			MessageName(callbackReq.MessageName).
			CorrelationKey(callbackReq.CorrelationKey).
			VariablesFromMap(callbackReq.Variables)

		if err != nil {
			_, err := command.Send(ctx)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		// Respond with a success message
		w.WriteHeader(http.StatusNoContent)
	})

	http.ListenAndServe(":8080", r)
}
