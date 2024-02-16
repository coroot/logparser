FROM golang:1.21-bullseye AS builder
WORKDIR /tmp/src
COPY . .
RUN go test ./...
RUN go build -mod=readonly -o ./logparser ./cmd/

FROM scratch
COPY --from=builder /tmp/src/logparser /usr/bin/logparser
ENTRYPOINT ["logparser"]
