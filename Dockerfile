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


FROM scratch

COPY --from=builder /app/app /

ENV HTTP_PORT=80
ENV GIN_MODE=release

HEALTHCHECK --interval=5s --timeout=2s --start-period=1s CMD ["/app", "healthcheck"]

# TODO: non-root user with access to docker socket file.
ENTRYPOINT ["/app"]
