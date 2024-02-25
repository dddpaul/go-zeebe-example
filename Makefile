.PHONY: all build release

IMAGE=dddpaul/go-zeebe-example

all: test

test:
	go clean -testcache
	go test -race -coverprofile=coverage.out ./...
	grep -v "_mock.go" coverage.out | grep -v mocks > coverage_no_mocks.out
	go tool cover -func=coverage_no_mocks.out
	rm coverage.out coverage_no_mocks.out

build-alpine:
	CGO_ENABLED=0 GOOS=linux go test
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./bin/go-zeebe-example ./main.go

build:
	@docker build --tag=${IMAGE} .

debug:
	@docker run -it --entrypoint=sh ${IMAGE}

release: build
	@echo "Tag image with version $(version)"
	@docker tag ${IMAGE} ${IMAGE}:$(version)

push: release
	@docker push ${IMAGE}
	@docker push ${IMAGE}:$(version)

worker:
	@go run main.go -zeebe-broker-addr 192.168.0.100:26500 -zeebe-worker-max-jobs-active 64 -zeebe-worker-concurrency 8

run: worker
