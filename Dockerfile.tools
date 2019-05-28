FROM golang:1.12.5 AS builder
WORKDIR /src/aerospike-operator/
COPY go.mod go.sum ./
RUN go mod download
COPY hack/tools/go.mod hack/tools/go.sum ./hack/tools/
RUN cd ./hack/tools/ && go mod download
COPY . .
RUN make build BIN=backup OUT=/backup
RUN make build BIN=asinit OUT=/asinit
WORKDIR $GOPATH/src/github.com/alicebob/asprom
RUN git clone https://github.com/alicebob/asprom .
RUN CGO_ENABLED=0 go build \
    -a \
    -v \
    -ldflags="-d -s -w" \
    -tags=netgo \
    -installsuffix=netgo \
    -o=/asprom *.go

FROM aerospike/aerospike-tools:3.15.3.14 AS astools

FROM ubuntu:16.04
RUN apt update && \
    apt install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*
COPY --from=builder /asinit /usr/local/bin/asinit
COPY --from=builder /asprom /usr/local/bin/asprom
COPY --from=builder /backup /usr/local/bin/backup
COPY --from=astools /usr/bin/asbackup /usr/local/bin/asbackup
COPY --from=astools /usr/bin/asrestore /usr/local/bin/asrestore
