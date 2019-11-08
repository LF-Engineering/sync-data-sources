FROM golang:1.13 AS builder
RUN apt-get update -y && apt-get upgrade -y && apt-get install -y ca-certificates openssh-client git curl make
RUN go get -u github.com/LF-Engineering/sync-data-sources
WORKDIR /go/src/github.com/LF-Engineering/sync-data-sources/sources
RUN make -f ./Makefile install
FROM alpine
RUN apk add git bash
COPY --from=builder /bins/* /usr/bin/
