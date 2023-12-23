package main

import (
	"flag"
	"fmt"
	"github.com/dddpaul/go-zeebe-example/pkg/service"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

var (
	verbose      bool
	trace        bool
	port         string
	zbBrokerAddr string
	zbProcessID  string
	redisAddr    string
)

func main() {
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&trace, "trace", false, "Enable network tracing")
	flag.StringVar(&port, "port", ":8080", "Port to listen (prepended by colon), i.e. :8080")
	flag.StringVar(&zbBrokerAddr, "zeebe-broker-addr", LookupEnvOrString("ZEEBE_BROKER_ADDR", "127.0.0.1:26500"), "Zeebe broker address")
	flag.StringVar(&zbProcessID, "zeebe-process-id", LookupEnvOrString("ZEEBE_PROCESS_ID", "diagram_1"), "BPMN process ID")
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
		service.WithZeebe(zbBrokerAddr, zbProcessID),
		service.WithRedis(redisAddr))

	s.Start()
}

func LookupEnvOrString(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func getConfig(fs *flag.FlagSet) []string {
	cfg := make([]string, 0, 10)
	fs.VisitAll(func(f *flag.Flag) {
		cfg = append(cfg, fmt.Sprintf("%s:%q", f.Name, f.Value.String()))
	})

	return cfg
}
