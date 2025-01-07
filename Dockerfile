FROM golang:1.23 AS builder

ARG COMPONENT
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY pkg/ ./pkg
COPY cmd/$COMPONENT .

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN go build -ldflags="-w -s" -o app


FROM alpine

RUN apk add --no-cache curl

COPY --from=builder /app/app /

ENV HTTP_PORT=80
ENV GIN_MODE=release

HEALTHCHECK --interval=10s --timeout=5s --start-period=2s CMD curl --fail http://127.0.0.1/pulse || exit 1

CMD ["/app"]
