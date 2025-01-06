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


FROM scratch
COPY --from=builder /app/app /
CMD ["/app"]
