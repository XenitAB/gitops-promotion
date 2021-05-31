TAG = dev
IMG ?= ghcr.io/xenitab/gitops-promotion:$(TAG)
TEST_ENV_FILE = tmp/test_env

ifneq (,$(wildcard $(TEST_ENV_FILE)))
    include $(TEST_ENV_FILE)
    export
endif

lint:
	golangci-lint run -E misspell ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

.SILENT: test
.PHONY: test
test: fmt vet
	go test -timeout 1m ./... -cover

gosec:
	gosec ./...

cover:
	mkdir -p tmp
	go test -timeout 1m ./... -coverprofile=tmp/coverage.out
	go tool cover -html=tmp/coverage.out

docker-build:
	docker build -t ${IMG} .
