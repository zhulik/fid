FROM golang:1.24 AS builder

ARG COMPONENT
WORKDIR /app

COPY go.mod go.sum ./

COPY pkg/ ./pkg
COPY internal/ ./internal
COPY cmd/$COMPONENT .

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN --mount=type=cache,target=/root/.cache/go-build  \
    --mount=type=cache,target=/go  \
    go build -ldflags="-w -s" -o app


FROM alpine

RUN apk add --no-cache curl

COPY --from=builder /app/app /

ENV HTTP_PORT=80
ENV GIN_MODE=release

HEALTHCHECK --interval=10s --timeout=5s --start-period=1s CMD curl --fail http://127.0.0.1/health || exit 1

# TODO: non-root user with access to docker socket file.
CMD ["/app"]
