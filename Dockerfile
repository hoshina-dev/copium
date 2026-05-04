# --- 1) build the SPA --------------------------------------------------------
FROM node:24-alpine AS webui

WORKDIR /webui

COPY webui/package.json webui/package-lock.json* ./
RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi

COPY webui/ ./
RUN npm run build

# --- 2) build the Go binary --------------------------------------------------
FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Replace the placeholder dist/ with the freshly built SPA so go:embed picks
# up real assets.
RUN rm -rf webui/dist
COPY --from=webui /webui/dist ./webui/dist

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o copium ./cmd/server

# --- 3) runtime image --------------------------------------------------------
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
