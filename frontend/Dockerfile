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
FROM nginxinc/nginx-unprivileged:1.27-alpine

ARG FRONTEND_PORT=8080
ENV FRONTEND_PORT=${FRONTEND_PORT}

COPY --chown=101:0 default.conf.template /etc/nginx/templates/default.conf.template
COPY --chown=101:0 --from=builder /app/dist /usr/share/nginx/html/momo-store

EXPOSE ${FRONTEND_PORT}

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null "http://127.0.0.1:${FRONTEND_PORT}/healthz" || exit 1
