package handlers

import (
	"context"
	"encoding/json"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"github.com/dddpaul/go-zeebe-example/pkg/cache"
	"github.com/dddpaul/go-zeebe-example/pkg/logger"
	log "github.com/sirupsen/logrus"
	"io"
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
	id := ctx.Value(logger.TRACE_ID).(string)

	// Start the process instance
	command, err := zbClient.NewCreateInstanceCommand().
		BPMNProcessId("diagram_1").
		LatestVersion().
		VariablesFromMap(map[string]interface{}{
			"uuid": id,
		})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response, err := command.Send(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Log(ctx, nil).WithFields(log.Fields{
		"bpmn-id":     response.BpmnProcessId,
		"process-key": response.ProcessInstanceKey,
	}).Infof("new process instance")

	// Listen for the job worker result or the timeout
	ch := make(chan bool, 1)
	cache.Add(id, ch)

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
		cache.Del(id)
	case <-ctx.Done():
		// If the context is done, it means we hit the timeout
		http.Error(w, "timeout waiting for the process to complete", http.StatusRequestTimeout)
	}
}

func Callback(zbClient zbc.Client, w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	var callbackReq CallbackRequest

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer func(b io.ReadCloser) {
		err := b.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}(r.Body)

	// Unmarshal the JSON request body into the CallbackRequest struct
	err = json.Unmarshal(body, &callbackReq)
	if err != nil {
		http.Error(w, "failed to unmarshal request body", http.StatusBadRequest)
		return
	}
	logger.Log(ctx, nil).WithField("uuid", callbackReq.Uuid).WithField("message", callbackReq.Message).Debugf("callback request")

	// Publish a message to the process instance
	command, err := zbClient.NewPublishMessageCommand().
		MessageName("callback").
		CorrelationKey(callbackReq.Uuid).
		VariablesFromMap(map[string]interface{}{
			"message": callbackReq.Message,
		})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response, err := command.Send(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Log(ctx, nil).WithFields(log.Fields{
		"uuid":        callbackReq.Uuid,
		"message":     callbackReq.Message,
		"message-key": response.GetKey(),
	}).Infof("callback message sent")

	// Respond with a success message
	w.WriteHeader(http.StatusNoContent)
}
