# Task
#===============================================================
export GO111MODULE=on

fmt:
	go fmt $$(go list ./... | tr '\n' ' ')

test:
	go test $$(go list ./... | tr '\n' ' ')

build:
	go build -o ./bin/kangol cmd/kangol/main.go

update:
	go get -u=patch

update_test: update test

.PHONY: fmt test build update