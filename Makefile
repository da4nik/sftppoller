BINARY=sftppoller
VERSION=1.0.0
BUILD_TIME=`date +%FT%T%z`
LDFLAGS=-ldflags "-linkmode external -s -w -extldflags -static -X github.com/da4nik/sftppoller/main.version=${VERSION} -X github.com/da4nik/sftppoller/main.buildTime=${BUILD_TIME}"

SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')

.PHONY: build run install clean image
.DEFAULT_GOAL: $(BINARY)

$(BINARY): $(SOURCES)
	# glide install --skip-test
	go build ${LDFLAGS} -o ${BINARY} sftppoller.go

build: $(BINARY)

run:
	@go run ${BINARY}.go

install:
	@go install ${LDFLAGS} ./...

image:
	docker build -t sftppoller .

clean:
	@if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi
