package main

import (
	"flag"
	"fmt"
	"github.com/dddpaul/go-zeebe-example/pkg/service"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
	"time"
)

var (
	verbose               bool
	trace                 bool
	port                  string
	zbBrokerAddr          string
	zbProcessID           string
	zbWorkerMaxJobsActive int
	zbWorkerConcurrency   int
	zbStreamEnabled       bool
	redisAddr             string
)

func main() {
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&trace, "trace", false, "Enable network tracing")
	flag.StringVar(&port, "port", LookupEnvOrString("SERVER_PORT", ":8080"), "Port to listen (prepended by colon), i.e. :8080")
	flag.StringVar(&zbBrokerAddr, "zeebe-broker-addr", LookupEnvOrString("ZEEBE_BROKER_ADDR", "127.0.0.1:26500"), "Zeebe broker address")
	flag.StringVar(&zbProcessID, "zeebe-process-id", LookupEnvOrString("ZEEBE_PROCESS_ID", "diagram_1"), "BPMN process ID")
	flag.IntVar(&zbWorkerMaxJobsActive, "zeebe-worker-max-jobs-active", LookupEnvOrInt("ZEEBE_WORKER_MAX_JOBS_ACTIVE", 0), "Max amount of active jobs for worker, if 0 then Zeebe client default is used (32)")
	flag.IntVar(&zbWorkerConcurrency, "zeebe-worker-concurrency", LookupEnvOrInt("ZEEBE_WORKER_CONCURRENCY", 0), "Worker concurrency, if 0 then Zeebe client default is used (4)")
	flag.BoolVar(&zbStreamEnabled, "zeebe-worker-job-streaming", false, "Enable Zeebe worker job streaming")
	flag.StringVar(&redisAddr, "redis-addr", LookupEnvOrString("REDIS_ADDR", ""), "Redis cluster/server address")

	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	flag.Parse()
	log.Printf("Configuration %v, timezone %v", getConfig(flag.CommandLine), time.Local)

	if verbose {
		log.SetLevel(log.DebugLevel)
	}

	if trace {
		log.SetLevel(log.TraceLevel)
	}

	s := service.New(
		service.WithHttpPort(port),
		service.WithZeebe(zbBrokerAddr, zbProcessID, zbWorkerMaxJobsActive, zbWorkerConcurrency, zbStreamEnabled),
		service.WithRedis(redisAddr))

	s.Start()
}

func LookupEnvOrString(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func LookupEnvOrInt(key string, defaultVal int) int {
	val, err := strconv.Atoi(LookupEnvOrString(key, strconv.Itoa(defaultVal)))
	if err != nil {
		panic(err)
	}
	return val
}

func getConfig(fs *flag.FlagSet) []string {
	cfg := make([]string, 0, 10)
	fs.VisitAll(func(f *flag.Flag) {
		cfg = append(cfg, fmt.Sprintf("%s:%q", f.Name, f.Value.String()))
	})
	return cfg
}
