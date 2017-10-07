FROM golang:1.9 AS builder

RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.3.1/dep-linux-amd64 && chmod +x /usr/local/bin/dep

RUN mkdir -p /go/src/github.com/jshirley/slack-ask

COPY ./cmd /go/src/github.com/jshirley/slack-ask/cmd
COPY ./asker /go/src/github.com/jshirley/slack-ask/asker
WORKDIR /go/src/github.com/jshirley/slack-ask

COPY Gopkg.toml Gopkg.lock main.go ./
# copies the Gopkg.toml and Gopkg.lock to WORKDIR

RUN dep ensure -vendor-only
# install the dependencies without checking for go code

RUN go build

