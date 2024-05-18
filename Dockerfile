ARG GO_VERSION=1.22
FROM golang:${GO_VERSION} AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o main ./cmd/main.go

FROM golang:${GO_VERSION} AS build-release-stage
WORKDIR /app
COPY --from=builder /app/main .

EXPOSE 8080
ENTRYPOINT [ "./main" ]
