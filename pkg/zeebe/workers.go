package zeebe

import (
	"context"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"github.com/dddpaul/go-zeebe-example/pkg/logger"
	"github.com/dddpaul/go-zeebe-example/pkg/pubsub"
	"time"
)

const (
	SERVICE_TASK = "service-task"
	FINAL_TASK   = "final-task"
)

var pubSub pubsub.PubSub

func StartJobWorkers(client zbc.Client, ps pubsub.PubSub) {
	pubSub = ps
	go startJobWorker(client, SERVICE_TASK, handleJob)
	go startJobWorker(client, FINAL_TASK, handleFinalJob)
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
	logger.Log(ctx, nil).WithField(logger.INPUTS, job.Variables).Debugf("job activated")

	result := map[string]interface{}{
		"result": "final task success",
	}

	completeJob(ctx, client, job, result)

	// Send complete signal for waiting /sync method
	//ch := cache.Get(ctx.Value(logger.APP_ID).(string))
	//ch <- result["result"]

	err := pubSub.Publish(ctx, ctx.Value(logger.APP_ID).(string), "Success")
	if err != nil {
		logger.Log(ctx, err).WithField(logger.INPUTS, job.Variables).Error("error while publish to redis")
		return
	}
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

func newContext(job entities.Job) context.Context {
	vars, _ := job.GetVariablesAsMap()
	ctx := context.WithValue(context.Background(), logger.APP_ID, vars[APP_ID].(string))
	return context.WithValue(ctx, logger.JOB_TYPE, job.Type)
}
