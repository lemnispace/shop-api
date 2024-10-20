.PHONY: build-%

build-%:
	@echo "Building $*..."
	GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./build/$*/bootstrap ./cmd/$*

test:
	go test -v ./...