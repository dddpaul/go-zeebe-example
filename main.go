package main

import (
	"flag"
	"fmt"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"github.com/dddpaul/go-zeebe-example/pkg/handlers"
	"github.com/dddpaul/go-zeebe-example/pkg/logger"
	"github.com/dddpaul/go-zeebe-example/pkg/zeebe"
	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"time"
)

var (
	verbose         bool
	trace           bool
	zeebeBrokerAddr string
	port            string
)

func main() {
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&trace, "trace", false, "Enable network tracing")
	flag.StringVar(&port, "port", ":8080", "Port to listen (prepended by colon), i.e. :8080")
	flag.StringVar(&zeebeBrokerAddr, "zeebe-broker-addr", LookupEnvOrString("ZEEBE_BROKER_ADDR", ""), "Zeebe broker address")

	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	flag.Parse()
	log.Printf("Configuration %v, timezone %v", getConfig(flag.CommandLine), time.Local)

	if len(zeebeBrokerAddr) == 0 {
		panic("Zeebe broker address has to be specified")
	}

	if verbose {
		log.SetLevel(log.DebugLevel)
	}

	if trace {
		log.SetLevel(log.TraceLevel)
	}

	r := chi.NewRouter()

	zbClient := zeebe.NewClient(zeebeBrokerAddr)
	defer func(z zbc.Client) {
		err := z.Close()
		if err != nil {
			panic(err)
		}
	}(zbClient)

	// Deploy process and start job workers
	zeebe.DeployProcessDefinition(zbClient)
	go zeebe.StartJobWorkers(zbClient)

	r.Post("/sync", func(w http.ResponseWriter, r *http.Request) {
		handlers.Sync(zbClient, w, r)
	})

	r.Post("/callback", func(w http.ResponseWriter, r *http.Request) {
		handlers.Callback(zbClient, w, r)
	})

	log.Printf("Start HTTP service on port %s with Zeebe broker %s", port, zeebeBrokerAddr)
	err := http.ListenAndServe(port, logger.NewMiddleware(r))
	if err != nil {
		panic(err)
	}
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
