FROM golang:1.10.1 AS builder
WORKDIR $GOPATH/src/github.com/travelaudience/aerospike-operator/
COPY . .
RUN go get -u github.com/golang/dep/cmd/dep
RUN make dep
RUN make gen
RUN CGO_ENABLED=0 go build -a -v -ldflags="-d -s -w" -tags=netgo -installsuffix=netgo -o=/aerospike-operator ./cmd/operator/main.go

FROM alpine:3.7
COPY --from=builder /aerospike-operator /usr/local/bin/aerospike-operator
CMD ["aerospike-operator", "-h"]
