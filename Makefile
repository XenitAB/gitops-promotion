TAG = dev
IMG ?= ghcr.io/xenitab/gitops-promotion:$(TAG)

lint:
	golangci-lint run -E misspell ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

test: fmt vet
	go test ./...

cover:
	mkdir -p tmp
	go test -timeout 1m ./... -coverprofile=tmp/coverage.out
	go tool cover -html=tmp/coverage.out

docker-build:
	docker build -t ${IMG} .
