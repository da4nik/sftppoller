FROM golang:1.8.3 as build
COPY . /usr/local/go/src/github.com/da4nik/sftppoller
WORKDIR /usr/local/go/src/github.com/da4nik/sftppoller

RUN mkdir /app && \
    curl https://glide.sh/get | sh && \
    glide install && \
    make build




FROM alpine:3.5

RUN apk add --no-cache ca-certificates

COPY --from=build /usr/local/go/src/github.com/da4nik/sftppoller/sftppoller /app/
WORKDIR /app
CMD ["/app/sftppoller"]
