package workers

import (
	"context"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"log"
	"time"
)

func StartJobWorkers(client zbc.Client) {
	jobWorker := client.NewJobWorker().
		JobType("service-task").
		Handler(handleJob).
		Open()
	defer jobWorker.Close()
	jobWorker.AwaitClose()
}

func handleJob(client worker.JobClient, job entities.Job) {
	log.Printf("Handling job: %s, input: %s", job.Type, job.Variables)

	variables := map[string]interface{}{
		"result": "Yes",
	}
	request, err := client.NewCompleteJobCommand().
		JobKey(job.Key).
		VariablesFromMap(variables)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_, err = request.Send(ctx)
	if err != nil {
		panic(err)
	}

	log.Printf("Successfully completed job: %s, result: %v", job.Type, variables)
}
