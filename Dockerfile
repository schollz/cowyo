FROM node:24-alpine AS frontend

WORKDIR /src/web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/index.html web/vite.config.js ./
COPY web/public ./public
COPY web/src ./src
RUN npm run build

FROM golang:1.26.5-alpine AS backend

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /src/internal/site/build ./internal/site/build
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/cowyo2 \
    ./cmd/cowyo2

FROM alpine:3.23

RUN apk add --no-cache ca-certificates \
    && addgroup -S cowyo \
    && adduser -S -G cowyo cowyo \
    && mkdir -p /data \
    && chown cowyo:cowyo /data

COPY --from=backend /out/cowyo2 /usr/local/bin/cowyo2

ENV SQLITE_PATH=/data/cowyo2.sqlite3

USER cowyo

EXPOSE 8001

CMD ["/usr/local/bin/cowyo2"]
