FROM golang:1.15.3-alpine3.12 AS builder

RUN apk add -U make

WORKDIR /usr/src/bootkube

COPY . .

ARG GOOS=linux
ARG GOARCH=amd64
ARG CMD=bootkube
ARG VERSION=undefined

RUN make GOARCH=$GOARCH CMD=$CMD VERSION=$VERSION

FROM scratch

COPY --from=builder /usr/src/bootkube/$CMD /$CMD
