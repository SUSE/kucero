FROM golang:1.13 as build
WORKDIR /src

COPY . .
RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w -X main.version=0.0.1" -o kucero cmd/kucero/*.go

FROM alpine:3.11
RUN addgroup -g 1000 kucero && \
    adduser -u 1000 -h /kucero -G kucero -S kucero
USER kucero
WORKDIR /bin
COPY --from=build /src/kucero .
ENTRYPOINT ["/bin/kucero"]
