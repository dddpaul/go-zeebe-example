package handlers

import (
	"context"
	"encoding/json"
	"github.com/camunda-cloud/zeebe/clients/go/pkg/entities"
	"github.com/camunda-cloud/zeebe/clients/go/pkg/worker"
	"github.com/camunda-cloud/zeebe/clients/go/pkg/zbc"
	"io"
	"log"
	"net/http"
	"time"
)

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

func Sync(zbClient zbc.Client, w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Start the process instance
	response, err := zbClient.NewCreateInstanceCommand().
		BPMNProcessId("diagram_1").
		LatestVersion().
		Send(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Process instance ID = %d", response.ProcessInstanceKey)

	// Listen for the job worker result or the timeout
	resultChan := make(chan string, 1)
	go func() {
		jobWorker := zbClient.NewJobWorker().
			JobType("end-task").
			Handler(func(jobClient worker.JobClient, job entities.Job) {
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
		err := json.NewEncoder(w).Encode(
			StartProcessResponse{
				ProcessInstanceKey: response.GetProcessInstanceKey(),
				Result:             result,
			})
		if err != nil {
			return
		}
	case <-ctx.Done():
		// If the context is done, it means we hit the timeout
		http.Error(w, "timeout waiting for the process to complete", http.StatusRequestTimeout)
	}
}

func Callback(zbClient zbc.Client, w http.ResponseWriter, r *http.Request) {
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
}
