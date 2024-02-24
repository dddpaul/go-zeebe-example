package zeebe

import (
	"context"
	"encoding/json"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"github.com/dddpaul/go-zeebe-example/pkg/logger"
	"github.com/dddpaul/go-zeebe-example/pkg/pubsub"
	"io"
	"net/http"
	"time"
)

const (
	SERVICE_TASK       = "service-task"
	FINAL_TASK         = "final-task"
	RISK_LEVEL_TASK    = "risk-level"
	LOOP_SETTINGS_TASK = "loop-settings"
	APPROVE_TASK       = "approve-app"
	REJECT_TASK        = "reject-app"
)

var pubSub pubsub.PubSub

type LoopSettings struct {
	Retries int    `json:"retries"`
	Timeout string `json:"timeout"`
}

func StartJobWorkers(client zbc.Client, ps pubsub.PubSub) {
	pubSub = ps
	go startJobWorker(client, SERVICE_TASK, handleJob)
	go startJobWorker(client, FINAL_TASK, handleFinalJob)
	go startJobWorker(client, RISK_LEVEL_TASK, handleRiskLevelJob)
	go startJobWorker(client, LOOP_SETTINGS_TASK, handleLoopSettingsJob)
	go startJobWorker(client, APPROVE_TASK, handleApproveAppJob)
	go startJobWorker(client, REJECT_TASK, handleRejectAppJob)
}

func startJobWorker(client zbc.Client, jobType string, handler func(client worker.JobClient, job entities.Job)) {
	jobWorker := client.NewJobWorker().
		JobType(jobType).
		Handler(handler).
		Open()
	defer jobWorker.Close()
	jobWorker.AwaitClose()
}

func handleJob(client worker.JobClient, job entities.Job) {
	ctx, cancel := context.WithTimeout(newContext(job), time.Second*5)
	defer cancel()
	logger.Log(ctx, nil).WithField(logger.INPUTS, job.Variables).Debugf("job activated")

	result := map[string]interface{}{
		"result": "service task success",
	}

	completeJob(ctx, client, job, result)
}

func handleFinalJob(client worker.JobClient, job entities.Job) {
	ctx, cancel := context.WithTimeout(newContext(job), time.Second*5)
	defer cancel()
	logger.Log(ctx, nil).WithField(logger.INPUTS, job.Variables).Debugf("final job activated")

	result := map[string]interface{}{
		"result": "final task success",
	}

	completeJob(ctx, client, job, result)

	err := pubSub.Publish(ctx, ctx.Value(logger.APP_ID).(string), "Success")
	if err != nil {
		logger.Log(ctx, err).WithField(logger.INPUTS, job.Variables).Error("error while publish")
		return
	}
}

func handleRiskLevelJob(client worker.JobClient, job entities.Job) {
	ctx, cancel := context.WithTimeout(newContext(job), time.Second*5)
	defer cancel()
	logger.Log(ctx, nil).WithField(logger.INPUTS, job.Variables).Debugf("risk level job activated")

	completeJob(ctx, client, job, map[string]interface{}{})
}

func handleLoopSettingsJob(client worker.JobClient, job entities.Job) {
	ctx, cancel := context.WithTimeout(newContext(job), time.Second*5)
	defer cancel()
	logger.Log(ctx, nil).WithField(logger.INPUTS, job.Variables).Debugf("loop settings job activated")

	loopSettings := LoopSettings{
		Retries: 1,
		Timeout: "20s",
	}

	resp, err := http.Get("http://192.168.0.100:10000/params.json")
	if err != nil {
		logger.Log(ctx, err).Error("Failed to fetch loop settings")
	} else {
		body, _ := io.ReadAll(resp.Body)
		err = json.Unmarshal(body, &loopSettings)
		if err != nil {
			logger.Log(ctx, err).Error("Failed to unmarshal loop settings")
		}
	}

	result := map[string]interface{}{
		"retries": loopSettings.Retries,
		"timeout": loopSettings.Timeout,
	}

	completeJob(ctx, client, job, result)
}

func handleApproveAppJob(client worker.JobClient, job entities.Job) {
	ctx, cancel := context.WithTimeout(newContext(job), time.Second*5)
	defer cancel()
	logger.Log(ctx, nil).WithField(logger.INPUTS, job.Variables).Debugf("approve job activated")
	completeJob(ctx, client, job, map[string]interface{}{})
}

func handleRejectAppJob(client worker.JobClient, job entities.Job) {
	ctx, cancel := context.WithTimeout(newContext(job), time.Second*5)
	defer cancel()
	logger.Log(ctx, nil).WithField(logger.INPUTS, job.Variables).Debugf("reject job activated")
	completeJob(ctx, client, job, map[string]interface{}{})
}

func completeJob(ctx context.Context, client worker.JobClient, job entities.Job, result map[string]interface{}) {
	req, _ := client.NewCompleteJobCommand().
		JobKey(job.Key).
		VariablesFromMap(result)
	if _, err := req.Send(ctx); err != nil {
		logger.Log(ctx, err).Errorf("error")
	}
	logger.Log(ctx, nil).WithField(logger.OUTPUTS, result).Debugf("job completed")
}

func failJob(ctx context.Context, client worker.JobClient, job entities.Job, e ZeebeBpmnError) {
	req := client.NewFailJobCommand().
		JobKey(job.GetKey()).
		Retries(job.Retries - 1).
		ErrorMessage(e.Message)
	if _, err := req.Send(ctx); err != nil {
		logger.Log(ctx, err).Errorf("error")
	}
	logger.Log(ctx, nil).Debugf("job failed")
}

func throwErrorJob(ctx context.Context, client worker.JobClient, job entities.Job, e ZeebeBpmnError) {
	req := client.NewThrowErrorCommand().
		JobKey(job.GetKey()).
		ErrorCode(e.Code)
	if _, err := req.Send(ctx); err != nil {
		logger.Log(ctx, err).Errorf("error")
	}
	logger.Log(ctx, nil).Debugf("job failed")
}

func newContext(job entities.Job) context.Context {
	vars, _ := job.GetVariablesAsMap()
	ctx := context.WithValue(context.Background(), logger.APP_ID, vars[APP_ID].(string))
	return context.WithValue(ctx, logger.JOB_TYPE, job.Type)
}
