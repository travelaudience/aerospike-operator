FROM golang:1.12.5 AS builder
WORKDIR /src/aerospike-operator/
COPY go.mod go.sum ./
RUN go mod download
COPY hack/tools/go.mod hack/tools/go.sum ./hack/tools/
RUN cd ./hack/tools/ && go mod download
COPY . .
RUN make build BIN=operator OUT=/aerospike-operator

FROM gcr.io/distroless/static
COPY --from=builder /aerospike-operator /usr/local/bin/aerospike-operator
CMD ["aerospike-operator", "-h"]
