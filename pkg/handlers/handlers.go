package handlers

import (
	"context"
	"encoding/json"
	"github.com/camunda-cloud/zeebe/clients/go/pkg/entities"
	"github.com/camunda-cloud/zeebe/clients/go/pkg/worker"
	"github.com/camunda-cloud/zeebe/clients/go/pkg/zbc"
	"github.com/google/uuid"
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
	Uuid    string `json:"uuid"`
	Message string `json:"message"`
}

func Sync(zbClient zbc.Client, w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	// Generate UUID for correlation key
	id, err := uuid.NewUUID()
	if err != nil {
		panic(err)
	}

	// Start the process instance
	variables := map[string]interface{}{
		"uuid": id.String(),
	}
	command, err := zbClient.NewCreateInstanceCommand().
		BPMNProcessId("diagram_1").
		LatestVersion().
		VariablesFromMap(variables)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response, err := command.Send(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Process instance ID = %d, uuid = %s", response.ProcessInstanceKey, id.String())

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

	variables := map[string]interface{}{
		"message": callbackReq.Message,
	}
	command, err := zbClient.NewPublishMessageCommand().
		MessageName("callback").
		CorrelationKey(callbackReq.Uuid).
		VariablesFromMap(variables)

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
