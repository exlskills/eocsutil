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

COPY --from=gobuilder /go/src/github.com/exlskills/eocsutil/*.md /go/src/github.com/exlskills/eocsutil/eocsutil /go/src/github.com/exlskills/eocsutil/package.json /go/src/github.com/exlskills/eocsutil/yarn.lock ./
COPY --from=gobuilder /go/src/github.com/exlskills/eocsutil/showdownjs/ ./showdownjs/
COPY --from=gobuilder /go/src/github.com/exlskills/eocsutil/vendor/ ./vendor/

RUN yarn install

EXPOSE 3344
ENTRYPOINT ["./eocsutil"]
CMD ["serve-gh-hook"]

# Example use:
# RUN mkdir /course && git clone https://github.com/exlskills/course-python-introduction.git /course
# ENV MGO_DB_NAME exldev
# ENV MGO_DB_URI mongodb://172.17.0.1:27017
# ./eocsutil convert --from-format eocs --from-uri /course --to-format eocs --to-uri $MGO_DB_URI

