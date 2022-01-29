.DEFAULT_GOAL = test

all: vet build test

vet:
	go vet

build:
	go build

test:
	go test -race -v

cover:
	go test -coverprofile=coverage.txt -covermode=atomic
	go tool cover -html=coverage.txt -o coverage.html

clean:
	go clean
	rm -f coverage.*

publish:
	GOPROXY=proxy.golang.org go list -m 'github.com/prantlf/go-multipart-composer@v$(VERSION)'

.PHONY: vet build test cover clean
