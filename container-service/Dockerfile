FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/app .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /out/app ./app

# Expose app port (your API)
EXPOSE 8081

# Expose Prometheus metrics port
EXPOSE 2112

ENV SPARROW_ORIGIN="http://localhost:5173" \
    CLONE_BASE_DIR="/temp"

CMD ["./app"]
