package zeebe

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"github.com/dddpaul/go-zeebe-example/pkg/logger"
	"github.com/dddpaul/go-zeebe-example/pkg/pubsub"
	"github.com/dddpaul/go-zeebe-example/pkg/stats"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
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

var (
	maxJobsActive = 0
	concurrency   = 0
	streamEnabled = false
	pubSub        pubsub.PubSub
)

type LoopSettings struct {
	Retries int    `json:"retries"`
	Timeout string `json:"timeout"`
}

func StartJobWorkers(client zbc.Client, mja int, c int, se bool, ps pubsub.PubSub) {
	maxJobsActive = mja
	concurrency = c
	streamEnabled = se
	pubSub = ps
	go startJobWorker(client, SERVICE_TASK, handleJob)
	go startJobWorker(client, FINAL_TASK, handleFinalJob)
	go startJobWorker(client, RISK_LEVEL_TASK, handleRiskLevelJob)
	go startJobWorker(client, LOOP_SETTINGS_TASK, handleLoopSettingsJob)
	go startJobWorker(client, APPROVE_TASK, handleApproveAppJob)
	go startJobWorker(client, REJECT_TASK, handleRejectAppJob)
}

func startJobWorker(client zbc.Client, jobType string, handler func(client worker.JobClient, job entities.Job)) {
	builder := client.NewJobWorker().
		JobType(jobType).
		Handler(handler).
		StreamEnabled(streamEnabled)
	if maxJobsActive > 0 {
		builder = builder.MaxJobsActive(maxJobsActive)
	}
	if concurrency > 0 {
		builder = builder.Concurrency(concurrency)
	}
	jobWorker := builder.Open()
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

	s, err := extractStringVar(job, "chance")
	if err != nil {
		throwErrorJob(ctx, client, job, NewZeebeBpmnError(RiskLevelError, err))
	}

	chance, err := strconv.Atoi(s)
	if err != nil {
		throwErrorJob(ctx, client, job, NewZeebeBpmnError(RiskLevelError, err))
	}

	if !slices.Contains(RiskLevels(), chance) {
		throwErrorJob(ctx, client, job, NewZeebeBpmnError(ChanceUnacceptableError, err))
	}

	result := map[string]interface{}{
		"riskLevel": strings.ToLower(RiskLevel(chance).GetString()),
	}

	completeJob(ctx, client, job, result)
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
	stats.IncrementApproved()
	completeJob(ctx, client, job, map[string]interface{}{})
}

func handleRejectAppJob(client worker.JobClient, job entities.Job) {
	ctx, cancel := context.WithTimeout(newContext(job), time.Second*5)
	defer cancel()
	logger.Log(ctx, nil).WithField(logger.INPUTS, job.Variables).Debugf("reject job activated")
	stats.IncrementRejected()
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
	ctx := context.Background()
	s, err := extractStringVar(job, APP_ID)
	if err != nil {
		logger.Log(ctx, err).Debugf("")
	}
	ctx = context.WithValue(ctx, logger.APP_ID, s)
	return context.WithValue(ctx, logger.JOB_TYPE, job.Type)
}

func extractStringVar(job entities.Job, name string) (string, error) {
	vars, _ := job.GetVariablesAsMap()
	if v, ok := vars[name]; ok {
		if s, ok := v.(string); ok {
			return s, nil
		}
	}
	return "", fmt.Errorf("failed to extract '%s' field from job vars: %s", name, job.Variables)
}
