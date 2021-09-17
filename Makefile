TAG = dev
IMG ?= ghcr.io/xenitab/gitops-promotion:$(TAG)
TEST_ENV_FILE = tmp/test_env

ifneq (,$(wildcard $(TEST_ENV_FILE)))
    include $(TEST_ENV_FILE)
    export
endif

.SILENT: lint
.PHONY: lint
lint:
	golangci-lint run ./...

.SILENT: fmt
.PHONY: fmt
fmt:
	go fmt ./...

.SILENT: vet
.PHONY: vet
vet:
	go vet ./...

.SILENT: test
.PHONY: test
test: fmt vet
	go test -timeout 2m ./... -cover

cover:
	mkdir -p tmp
	go test -timeout 5m -coverpkg=./... -coverprofile=tmp/coverage.out ./...
	go tool cover -html=tmp/coverage.out

docker-build:
	docker build -t ${IMG} .
