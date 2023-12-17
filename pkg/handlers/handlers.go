package handlers

import (
	"context"
	"encoding/json"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"github.com/dddpaul/go-zeebe-example/pkg/cache"
	"github.com/dddpaul/go-zeebe-example/pkg/logger"
	"github.com/dddpaul/go-zeebe-example/pkg/zeebe"
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
	Message string `json:"message"`
}

func Sync(zbClient zbc.Client, w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	id := ctx.Value(logger.APP_ID).(string)

	// Start the process instance
	command, _ := zbClient.NewCreateInstanceCommand().
		BPMNProcessId("diagram_1").
		LatestVersion().
		VariablesFromMap(map[string]interface{}{
			zeebe.APP_ID: id,
		})
	resp, err := command.Send(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Log(ctx, nil).WithFields(log.Fields{
		logger.BMPN_ID:     resp.BpmnProcessId,
		logger.PROCESS_KEY: resp.ProcessInstanceKey,
	}).Infof("new process instance")

	// Listen for the job worker result or the timeout
	ch := make(chan bool, 1)
	cache.Add(id, ch)

	select {
	case <-ch:
		// If we got the result within the timeout, send it back
		err := json.NewEncoder(w).Encode(
			StartProcessResponse{
				ProcessInstanceKey: resp.GetProcessInstanceKey(),
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
	id := ctx.Value(logger.APP_ID).(string)
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
	logger.Log(ctx, nil).WithFields(log.Fields{
		"message": callbackReq.Message,
	}).Debugf("callback request")

	// Publish a message to the process instance
	command, _ := zbClient.NewPublishMessageCommand().
		MessageName("callback").
		CorrelationKey(id).
		VariablesFromMap(map[string]interface{}{
			"message": callbackReq.Message,
		})
	response, err := command.Send(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Log(ctx, nil).WithFields(log.Fields{
		"message":     callbackReq.Message,
		"message-key": response.GetKey(),
	}).Infof("callback message sent")

	// Respond with a success message
	w.WriteHeader(http.StatusNoContent)
}
