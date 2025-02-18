FROM golang:1.23 as builder
LABEL authors="liun"

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY templates ./templates
RUN CGO_ENABLED=0 GOOS=linux go build -o /aqua

FROM alpine:3.21
WORKDIR /
COPY --from=builder /aqua /aqua
EXPOSE 8080
ENTRYPOINT ["/aqua"]

