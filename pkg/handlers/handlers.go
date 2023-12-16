package handlers

import (
	"context"
	"encoding/json"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"github.com/dddpaul/go-zeebe-example/pkg/cache"
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
	ch := make(chan bool, 1)
	cache.Add(id.String(), ch)

	select {
	case <-ch:
		// If we got the result within the timeout, send it back
		err := json.NewEncoder(w).Encode(
			StartProcessResponse{
				ProcessInstanceKey: response.GetProcessInstanceKey(),
				Result:             "Success",
			})
		if err != nil {
			return
		}
		cache.Del(id.String())
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
	log.Printf("Parsed callback request: %v", callbackReq)

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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := command.Send(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Message sent %s", response.String())

	// Respond with a success message
	w.WriteHeader(http.StatusNoContent)
}
