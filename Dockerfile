FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o copium ./cmd/server

FROM alpine:latest

WORKDIR /root/

RUN apk --no-cache add ca-certificates && \
    addgroup -g 1000 appgroup && \
    adduser -D -u 1000 -G appgroup appuser

COPY --from=builder /app/copium .
RUN chown -R appuser:appgroup /root/

USER appuser
EXPOSE 8081

CMD ["./copium"]
