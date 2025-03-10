# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS base

WORKDIR /build

RUN apk add git

ARG DOCKER_VERSION=v27.5.1+incompatible

COPY go.mod go.sum ./

RUN go get github.com/docker/docker@${DOCKER_VERSION}

RUN go mod download

COPY ./src ./src

RUN go build -o docker-reload ./src


FROM alpine

COPY --from=base /build/docker-reload /bin/docker-reload

ENTRYPOINT [ "/bin/docker-reload" ]
