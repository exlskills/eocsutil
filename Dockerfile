FROM golang:1.11-alpine3.8 as gobuilder

MAINTAINER EXL INC

WORKDIR /go/src/github.com/exlskills/eocsutil/
ENV GOPATH /go
RUN apk add git
RUN go get -u github.com/golang/dep/cmd/dep

COPY . .

RUN go get -d -v ./...
RUN dep ensure -v
RUN go build

FROM node:8.12-alpine
WORKDIR /app/
RUN apk add git
COPY --from=gobuilder /go/src/github.com/exlskills/eocsutil .
RUN yarn install

