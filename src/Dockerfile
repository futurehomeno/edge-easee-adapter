# build dependencies
FROM golang:1.20-alpine as dependencies
WORKDIR /build

RUN apk add --no-cache ca-certificates git
ENV GO111MODULE=on
ENV GOPRIVATE=github.com/futurehomeno

COPY ./.netrc /root/.netrc
RUN chmod 600 /root/.netrc

COPY go.mod go.sum ./
RUN go mod download

# Build container
FROM golang:1.20-alpine as builder
WORKDIR /build
COPY . /build

#copy built dependencies
COPY --from=dependencies /go /go
RUN CGO_ENABLED=0 GOOS=linux go build -o app-bin .

# Run container
FROM alpine
WORKDIR /app

RUN apk update && apk add bash

COPY --from=builder /build/app-bin .
COPY ../testdata/defaults/config-local.json ./defaults/config.json
COPY  ../testdata/defaults/adapter.json ./defaults/adapter.json
COPY  ../testdata/defaults/app-manifes.json ./defaults/app-manifes.json
COPY ../testdata/testing/configured/data/adapter.json ./data/adapter.json

EXPOSE 8005

CMD ["./app-bin"]


