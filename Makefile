TAG = dev
GITOPS_PROMOTION_IMAGE ?= ghcr.io/xenitab/gitops-promotion:$(TAG)
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
	go test -timeout 5m -coverpkg=./pkg/... -coverprofile=tmp/coverage.out ./pkg/...
	go tool cover -html=tmp/coverage.out

docker-build:
	docker build -t ${GITOPS_PROMOTION_IMAGE} .

verify:
	go test -timeout 2m -v ./tests/...