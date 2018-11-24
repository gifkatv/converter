# build stage
FROM golang:1.11-alpine3.8 AS build-env
MAINTAINER <terranisu@gmail.com>

RUN apk update && \
    apk add git

RUN go get -u github.com/golang/dep/cmd/dep && \
    go get github.com/codegangsta/gin

WORKDIR /go/src/app
ADD . /go/src/app
RUN cd /go/src/app && GOOS=linux go build -o converter

# final stage
FROM alpine:3.8
WORKDIR /app
COPY --from=build-env /go/src/app /app/
CMD ["./converter"]
