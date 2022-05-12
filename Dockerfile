FROM golang:1.18-buster as build
WORKDIR /src

ARG VERSION=latest

COPY . .
RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w -X main.version=${VERSION}" -o kucero cmd/kucero/*.go

FROM opensuse/leap:15.4
WORKDIR /usr/bin
COPY --from=build /src/kucero .
ENTRYPOINT ["/usr/bin/kucero"]
