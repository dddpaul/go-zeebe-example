package zeebe

import (
	"context"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"github.com/dddpaul/go-zeebe-example/pkg/cache"
	"github.com/dddpaul/go-zeebe-example/pkg/logger"
	"time"
)

func StartJobWorkers(client zbc.Client) {
	jobWorker := client.NewJobWorker().
		JobType("service-task").
		Handler(handleJob).
		Open()
	defer jobWorker.Close()

	finalWorker := client.NewJobWorker().
		JobType("final-task").
		Handler(handleFinalJob).
		Open()
	defer finalWorker.Close()

	jobWorker.AwaitClose()
	finalWorker.AwaitClose()
}

func handleJob(client worker.JobClient, job entities.Job) {
	ctx, cancel := context.WithTimeout(newContext(job), time.Second*5)
	defer cancel()
	logger.Log(ctx, nil).WithField(logger.INPUTS, job.Variables).Debugf("job activated")

	result := map[string]interface{}{
		"result": "service task success",
	}

	req, _ := client.NewCompleteJobCommand().JobKey(job.Key).VariablesFromMap(result)
	if _, err := req.Send(ctx); err != nil {
		logger.Log(ctx, err).Errorf("error")
	}
	logger.Log(ctx, nil).WithField(logger.OUTPUTS, result).Debugf("job completed")
}

func handleFinalJob(client worker.JobClient, job entities.Job) {
	ctx, cancel := context.WithTimeout(newContext(job), time.Second*5)
	defer cancel()
	logger.Log(ctx, nil).WithField(logger.INPUTS, job.Variables).Debugf("job activated")

	result := map[string]interface{}{
		"result": "final task success",
	}

	req, _ := client.NewCompleteJobCommand().JobKey(job.Key).VariablesFromMap(result)
	if _, err := req.Send(ctx); err != nil {
		logger.Log(ctx, err).Errorf("error")
	}
	logger.Log(ctx, nil).WithField(logger.OUTPUTS, result).Debugf("job completed")

	// Send complete signal for waiting /sync method
	ch := cache.Get(ctx.Value(logger.APP_ID).(string))
	ch <- result["result"]
}

func newContext(job entities.Job) context.Context {
	vars, _ := job.GetVariablesAsMap()
	ctx := context.WithValue(context.Background(), logger.APP_ID, vars[APP_ID].(string))
	return context.WithValue(ctx, logger.JOB_TYPE, job.Type)
}
