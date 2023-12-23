package handlers

import (
	"context"
	"encoding/json"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"github.com/dddpaul/go-zeebe-example/pkg/logger"
	"github.com/dddpaul/go-zeebe-example/pkg/pubsub"
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

func Sync(zbClient zbc.Client, zbProcessID string, pubSub pubsub.PubSub, w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	id := ctx.Value(logger.APP_ID).(string)

	processInstanceKey, err := zeebe.StartProcess(ctx, zbClient, zbProcessID, id)
	if err != nil {
		respondWithError(ctx, w, err, http.StatusInternalServerError)
		return
	}

	//ch, cleanup := cache.Add(id)
	//defer cleanup(id)

	ch := pubSub.Subscribe(ctx, id)

	select {
	case result := <-ch:
		respondWithJSON(w, StartProcessResponse{
			ProcessInstanceKey: processInstanceKey,
			Result:             result.Text,
		})
	case <-ctx.Done():
		respondWithError(ctx, w, ctx.Err(), http.StatusRequestTimeout)
	}
}

func SyncWithResult(zbClient zbc.Client, zbProcessID string, w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	id := ctx.Value(logger.APP_ID).(string)

	processInstanceKey, variables, err := zeebe.StartProcessWithResult(ctx, zbClient, zbProcessID, id)
	if err != nil {
		respondWithError(ctx, w, err, http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, StartProcessResponse{
		ProcessInstanceKey: processInstanceKey,
		Result:             variables,
	})
}

func Callback(zbClient zbc.Client, w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	id := ctx.Value(logger.APP_ID).(string)
	var req CallbackRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(ctx, w, err, http.StatusBadRequest)
		return
	}
	defer closeBody(ctx, r.Body)

	if err := zeebe.PublishCallbackMessage(ctx, zbClient, id, req.Message); err != nil {
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
