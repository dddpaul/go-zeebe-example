package handlers

import (
	"context"
	"encoding/json"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"github.com/dddpaul/go-zeebe-example/pkg/cache"
	"github.com/dddpaul/go-zeebe-example/pkg/logger"
	"github.com/dddpaul/go-zeebe-example/pkg/zeebe"
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

func Sync(zbClient zbc.Client, zbProcessID string, w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	id := ctx.Value(logger.APP_ID).(string)

	processInstanceKey, err := startProcess(ctx, zbClient, zbProcessID, id)
	if err != nil {
		respondWithError(ctx, w, err, http.StatusInternalServerError)
		return
	}

	ch, cleanup := cache.Add(id)
	defer cleanup(id)

	select {
	case <-ch:
		respondWithJSON(w, StartProcessResponse{
			ProcessInstanceKey: processInstanceKey,
			Result:             "Success",
		})
	case <-ctx.Done():
		respondWithError(ctx, w, ctx.Err(), http.StatusRequestTimeout)
	}
}

func SyncWithResult(zbClient zbc.Client, zbProcessID string, w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	id := ctx.Value(logger.APP_ID).(string)

	cmd, _ := zbClient.NewCreateInstanceCommand().
		BPMNProcessId(zbProcessID).
		LatestVersion().
		VariablesFromMap(map[string]interface{}{
			zeebe.APP_ID: id,
		})
	resp, err := cmd.WithResult().Send(ctx)

	if err != nil {
		respondWithError(ctx, w, err, http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, StartProcessResponse{
		ProcessInstanceKey: resp.GetProcessInstanceKey(),
		Result:             "Success",
	})
}

func Callback(zbClient zbc.Client, w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	id := ctx.Value(logger.APP_ID).(string)
	var callbackReq CallbackRequest

	if err := json.NewDecoder(r.Body).Decode(&callbackReq); err != nil {
		respondWithError(ctx, w, err, http.StatusBadRequest)
		return
	}
	defer closeBody(ctx, r.Body)

	if err := publishCallbackMessage(ctx, zbClient, id, callbackReq.Message); err != nil {
		respondWithError(ctx, w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func respondWithError(ctx context.Context, w http.ResponseWriter, err error, statusCode int) {
	logger.Log(ctx, err).Error("error")
	http.Error(w, err.Error(), statusCode)
}

func respondWithJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

func closeBody(ctx context.Context, body io.ReadCloser) {
	if err := body.Close(); err != nil {
		logger.Log(ctx, err).Error("error closing body")
	}
}

func startProcess(ctx context.Context, zbClient zbc.Client, zbProcessID, id string) (int64, error) {
	cmd, _ := zbClient.NewCreateInstanceCommand().
		BPMNProcessId(zbProcessID).
		LatestVersion().
		VariablesFromMap(map[string]interface{}{
			zeebe.APP_ID: id,
		})
	resp, err := cmd.Send(ctx)
	if err != nil {
		return 0, err
	}
	return resp.GetProcessInstanceKey(), nil
}

func publishCallbackMessage(ctx context.Context, zbClient zbc.Client, id, message string) error {
	cmd, _ := zbClient.NewPublishMessageCommand().
		MessageName("callback").
		CorrelationKey(id).
		VariablesFromMap(map[string]interface{}{
			zeebe.MESSAGE: message,
		})
	_, err := cmd.Send(ctx)
	return err
}
