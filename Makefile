vendor:
	go mod tidy && go mod vendor

.PHONY: codegen
codegen: vendor
	go generate ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: test
test: vendor
	go test ./... -v

.PHONY: clean
clean:
	git clean -dfx
