FROM node:16-alpine AS builder

WORKDIR /app

# Ставим зависимости отдельным слоем — кэшируется, пока не меняются lock-файлы
COPY package.json package-lock.json ./
RUN npm ci

COPY . .

ARG VUE_APP_API_URL=http://localhost:8081
ENV VUE_APP_API_URL=${VUE_APP_API_URL}

RUN npm run build

# Образ уже работает под непривилегированным uid 101
FROM nginxinc/nginx-unprivileged:1.29-alpine3.23

USER root
RUN apk --no-cache upgrade
USER 101

COPY --chown=101:0 default.conf.template /etc/nginx/templates/default.conf.template
COPY --chown=101:0 --from=builder /app/dist /usr/share/nginx/html/momo-store

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null "http://127.0.0.1:8080/healthz" || exit 1
