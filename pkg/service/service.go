package service

import (
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"github.com/dddpaul/go-zeebe-example/pkg/handlers"
	"github.com/dddpaul/go-zeebe-example/pkg/logger"
	"github.com/dddpaul/go-zeebe-example/pkg/zeebe"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"net/http"
)

type Service struct {
	router      chi.Router
	zbClient    zbc.Client
	zbProcessID string
	zbCleanup   func(z zbc.Client)
	rdb         *redis.ClusterClient
	rdbClose    func(r *redis.ClusterClient)
	port        string
}

type Option func(s *Service)

func WithZeebe(zbBrokerAddr string, zbProcessID string) Option {
	return func(s *Service) {
		s.zbClient = zeebe.NewClient(zbBrokerAddr)
		s.zbProcessID = zbProcessID
		s.zbCleanup = func(z zbc.Client) {
			err := z.Close()
			if err != nil {
				panic(err)
			}
		}
	}
}

func WithRedis() Option {
	return func(s *Service) {
		s.rdb = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs: []string{":6379"},
		})
		s.rdbClose = func(r *redis.ClusterClient) {
			err := r.Close()
			if err != nil {
				panic(err)
			}
		}
	}
}

func WithHttpPort(port string) Option {
	return func(s *Service) {
		s.port = port
	}
}

func New(opts ...Option) *Service {
	s := &Service{
		router: chi.NewRouter(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Service) Start() {
	defer s.cleanup()

	// Deploy process and start job workers
	zeebe.DeployProcessDefinition(s.zbClient, s.zbProcessID)
	go zeebe.StartJobWorkers(s.zbClient)

	router := chi.NewRouter()
	router.Post("/sync", func(w http.ResponseWriter, r *http.Request) {
		handlers.Sync(s.zbClient, s.zbProcessID, w, r)
	})
	router.Post("/sync-with-result", func(w http.ResponseWriter, r *http.Request) {
		handlers.SyncWithResult(s.zbClient, s.zbProcessID, w, r)
	})
	router.Post("/callback", func(w http.ResponseWriter, r *http.Request) {
		handlers.Callback(s.zbClient, w, r)
	})

	err := http.ListenAndServe(s.port, logger.NewMiddleware(s.router))
	if err != nil {
		panic(err)
	}
}

func (s *Service) cleanup() {
	s.zbCleanup(s.zbClient)
	s.rdbClose(s.rdb)
}
