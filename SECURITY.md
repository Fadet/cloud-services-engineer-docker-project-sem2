# Работа с секретами

## Где хранятся секреты

| Тип                   | Где хранится                                      | Как используется                                                |
| --------------------- | ------------------------------------------------- | --------------------------------------------------------------- |
| Креды Docker Registry | GitHub Secrets (`DOCKER_USER`, `DOCKER_PASSWORD`) | используются в CI через `docker/login-action`                   |
| Runtime-секреты       | файлы в `secrets/`                                | монтируются через Docker secrets в `/run/secrets/`              |
| Обычная конфигурация  | `.env`                                            | используется Compose для портов и других несекретных параметров |

Секреты не копируются в Docker-образы и передаются контейнерам только при запуске.

## Основные правила

### Не хранить секреты в `.env`

`.env` используется только для обычной конфигурации:

```env
FRONTEND_PORT=80
BACKEND_PORT=8081
```

Пароли, токены и ключи хранятся отдельно в `secrets/`.

### Не передавать секреты через `ARG` и `ENV`

Секреты не должны передаваться в Docker build через:

```dockerfile
ARG TOKEN
ENV TOKEN=...
```

Если секрет требуется во время сборки, используется BuildKit:

```dockerfile
RUN --mount=type=secret,id=npmrc,target=/root/.npmrc npm ci
```

```bash
docker build --secret id=npmrc,src=$HOME/.npmrc .
```

### Не использовать `VUE_APP_*` для секретов

`VUE_APP_*` попадает в собранный JavaScript и доступен пользователю в браузере.

Через такие переменные можно передавать только публичную конфигурацию, например адрес API.

### Исключать локальные секреты из build context

В `.dockerignore` исключены:

```text
.env*
.npmrc
secrets/
*.pem
*.key
*.crt
```

Это предотвращает случайное попадание локальных секретов в Docker-слои и build cache.

## Runtime-защита

В production-контейнерах используются:

* запуск без `root`;
* `read_only: true`;
* `cap_drop: [ALL]`;
* `no-new-privileges`;
* отдельная `internal`-сеть для backend.

Эти настройки ограничивают возможности контейнера в случае его компрометации.

## Basic Auth для `/metrics`

Доступ к `/metrics` закрыт Basic Auth.

Создать htpasswd:

```bash
docker run --rm httpd:2.4-alpine \
  htpasswd -nbB metrics '<пароль>' > secrets/metrics.htpasswd

chmod 0644 secrets/metrics.htpasswd
```

Запуск:

```bash
METRICS_HTPASSWD_FILE=./secrets/metrics.htpasswd docker compose up -d
```

Файл передаётся балансировщику как Docker secret.

При замене секрета контейнер необходимо пересоздать:

```bash
docker compose up -d --force-recreate lb
```

Внутри htpasswd хранится хеш пароля, а не сам пароль.

## Секрет для идентификаторов заказов

Идентификаторы заказов генерируются как:

```text
hex(HMAC-SHA256(secret, instanceNonce || counter))
```

`instanceNonce` генерируется отдельно для каждого backend-процесса, поэтому реплики могут создавать ID независимо друг от друга.

Создать секрет:

```bash
openssl rand -hex 32 > secrets/order_id.secret
chmod 0644 secrets/order_id.secret
```

Запустить backend с ним:

```bash
ORDER_ID_SECRET_FILE=./secrets/order_id.secret \
  docker compose up -d --force-recreate backend
```

Секрет монтируется в:

```text
/run/secrets/order_id_secret
```

и не входит в Docker-образ.

Если секрет не задан, backend создаёт временный ключ при запуске и выводит предупреждение в лог.