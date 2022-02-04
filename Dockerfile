FROM golang:1.16-buster AS builder
WORKDIR /go/src/logparser
COPY . .
RUN go build -mod=readonly -o /usr/bin/logparser ./cmd/
ENTRYPOINT ["logparser"]
