
NAME=vaultctl
AUTHOR=ukhomeofficedigital
REGISTRY=quay.io
HARDWARE=$(shell uname -m)
GODEPS=godep
VERSION=$(shell awk '/Version =/ { print $$3 }' doc.go | sed 's/"//g')
DEPS=$(shell go list -f '{{range .TestImports}}{{.}} {{end}}' ./...)
PACKAGES=$(shell go list ./...)
VETARGS?=-asmdecl -atomic -bool -buildtags -copylocks -methods -nilfunc -printf -rangeloops -shift -structtags -unsafeptr

.PHONY: test authors changelog build docker static release lint cover vet

default: build

build:
	@echo "--> Compiling the project"
	mkdir -p bin
	${GODEPS} go build -o bin/${NAME}

static:
	@echo "--> Compiling the static binary"
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux ${GODEPS} go build -a -tags netgo -ldflags '-w' -o bin/${NAME}

docker-build:
	@echo "--> Compiling the project"
	sudo docker run --rm -v ${ROOT_DIR}:/go/src/github.com/gambol99/${NAME} \
		-w /go/src/github.com/gambol99/${NAME} -e GOOS=linux golang:${GOVERSION} make static

docker: static
	@echo "--> Building the docker image"
	sudo docker build -t ${REGISTRY}/${AUTHOR}/${NAME}:${VERSION} .

release: static
	mkdir -p release
	gzip -c bin/${NAME} > release/${NAME}_${VERSION}_linux_${HARDWARE}.gz
	rm -f release/${NAME}

clean:
	rm -rf ./bin 2>/dev/null
	rm -rf ./release 2>/dev/null

authors:
	@echo "--> Updating the AUTHORS"
	git log --format='%aN <%aE>' | sort -u > AUTHORS

deps:
	@echo "--> Installing build dependencies"
	@go get -d -v ./... $(DEPS)

vet:
	@echo "--> Running go vet $(VETARGS) ."
	@go tool vet 2>/dev/null ; if [ $$? -eq 3 ]; then \
		go get golang.org/x/tools/cmd/vet; \
	fi
	@go tool vet $(VETARGS) *.go

lint:
	@echo "--> Running golint"
	@which golint 2>/dev/null ; if [ $$? -eq 1 ]; then \
		go get -u github.com/golang/lint/golint; \
	fi
	@golint .

format:
	@echo "--> Running go fmt"
	@go fmt $(PACKAGES)

cover:
	@echo "--> Running go cover"
	@go test --cover

test: deps
	@echo "--> Running the tests"
	go test -v
	@$(MAKE) vet
	@$(MAKE) cover

changelog: release
	git log $(shell git tag | tail -n1)..HEAD --no-merges --format=%B > changelog
