FROM golang:1.26.4-alpine3.23 AS builder

WORKDIR /src

# Копируем только go.mod и go.sum, чтобы использовать кэширование слоев Docker
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код и собираем бинарный файл
COPY . .

# Флаги -trimpath и -ldflags="-s -w" уменьшают размер бинарного файла, удаляя информацию о путях исходного кода и отладочную информацию.
RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.23

WORKDIR /app

# Порт задаётся на сборке и пробрасывается в окружение — его читает и сервер, и HEALTHCHECK.
ARG BACKEND_PORT=8081
ENV BACKEND_PORT=${BACKEND_PORT}

# Копируем бинарник и запускаем от непривилегированного пользователя с UID/GID 65532.
COPY --from=builder --chown=65532:65532 /out/api /app/api
USER 65532:65532

EXPOSE ${BACKEND_PORT}

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null "http://127.0.0.1:${BACKEND_PORT}/health" || exit 1

ENTRYPOINT ["/app/api"]
