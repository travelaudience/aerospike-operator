FROM golang:1.12.5 AS builder
RUN go get -u github.com/golang/dep/cmd/dep
WORKDIR $GOPATH/src/github.com/travelaudience/aerospike-operator/
COPY . .
RUN make build BIN=operator OUT=/aerospike-operator

FROM alpine:3.9

RUN apk add -U ca-certificates
COPY --from=builder /aerospike-operator /usr/local/bin/aerospike-operator
CMD ["aerospike-operator", "-h"]
