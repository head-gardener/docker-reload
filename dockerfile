# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS base

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build ./src -o docker-reload


FROM alpine

COPY --from=base /build/docker-reload /bin/docker-reload

ENTRYPOINT [ "/bin/docker-reload" ]
