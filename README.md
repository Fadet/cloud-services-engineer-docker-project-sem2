# Momo Store — контейнеризация

Учебный проект с Vue-фронтендом, Go-бэкендом и nginx-балансировщиком.

## Архитектура

```text
браузер ──:80────> frontend (nginx, SPA)                 сеть edge
       └──:8081──> lb (nginx) ──> backend x3 (:8081)    сети edge + internal
                                    └── orders_data:/data
```

Фронтенд обращается к API через `VUE_APP_API_URL`.

Бэкенд находится во внутренней сети `internal` и недоступен напрямую с хоста. Внешний доступ к нему идёт через nginx-балансировщик.

## Запуск

Перед первым запуском создайте секреты:

```bash
openssl rand -hex 32 > secrets/order_id.secret
docker run --rm httpd:2.4-alpine htpasswd -nbB metrics '<пароль>' > secrets/metrics.htpasswd
chmod 0644 secrets/order_id.secret secrets/metrics.htpasswd
```

Production:

```bash
docker compose up -d --build
```

Development:

```bash
docker compose -f docker-compose.dev.yml up -d --build
```

Остановка:

```bash
docker compose down
```

После запуска:

* frontend — http://localhost/
* API — http://localhost:8081/

Для production используется `docker-compose.yml`, для development — отдельный `docker-compose.dev.yml`.

## Конфигурация

Переменные задаются через `.env` или окружение. Пример находится в `.env.example`.

Внутренние порты контейнеров фиксированы:

* backend — `8081`
* load balancer — `8081`
* frontend — `8080`

Переменные окружения управляют опубликованными портами и путями к секретам.

| Переменная              | Назначение                      | По умолчанию                 |
| ----------------------- | ------------------------------- | ---------------------------- |
| `FRONTEND_PORT`         | порт frontend на хосте          | `80`                         |
| `BACKEND_PORT`          | порт API через load balancer    | `8081`                       |
| `BACKEND_DEBUG_PORT`    | прямой доступ к backend в dev   | `18081`                      |
| `METRICS_HTPASSWD_FILE` | htpasswd для `/metrics`         | `./secrets/metrics.htpasswd` |
| `ORDER_ID_SECRET_FILE`  | секрет для генерации ID заказов | `./secrets/order_id.secret`  |

### Build arguments

| Аргумент          | Где используется      | Назначение |
| ----------------- | --------------------- | ---------- |
| `VUE_APP_API_URL` | `frontend/Dockerfile` | адрес API  |

`VUE_APP_API_URL` попадает в frontend bundle во время сборки, поэтому не должен содержать секретные данные.

## Docker-образы

Все образы используют multi-stage сборку: компиляторы, зависимости и другие инструменты сборки не попадают в финальные образы.

| Образ                 |  Размер | Базовый образ                        |
| --------------------- | ------: | ------------------------------------ |
| `momo-store-backend`  | 37.3 MB | `alpine:3.23`                        |
| `momo-store-frontend` |  113 MB | `nginx-unprivileged:1.29-alpine3.23` |
| `momo-store-lb`       |  111 MB | `nginx-unprivileged:1.29-alpine3.23` |

Основные оптимизации:

* multi-stage сборка;
* статическая сборка Go-бинарника через `CGO_ENABLED=0`;
* `-trimpath -ldflags="-s -w"` для уменьшения бинарника;
* отдельные слои для установки зависимостей;
* `.dockerignore` для исключения лишних файлов и локальных секретов;
* обновление пакетов Alpine в финальных образах.

Load balancer собирается отдельным образом и также проверяется Trivy.

## Масштабирование

Бэкенд запускается в трёх репликах:

```text
nginx
  ├── backend
  ├── backend
  └── backend
```

nginx использует встроенный DNS Docker (`127.0.0.11`) и периодически обновляет адреса backend-контейнеров.

При недоступности одной реплики запрос может быть отправлен на другую через `proxy_next_upstream`.

Количество реплик можно менять без дополнительной координации. ID заказов генерируются независимо:

```text
hex(HMAC(secret, instanceNonce || counter))
```

`instanceNonce` уникален для каждого процесса.

## Хранение данных

Заказы сохраняются в:

```text
/data/orders.jsonl
```

Каталог подключён через named volume:

```text
orders_data
```

Поэтому данные сохраняются при пересоздании контейнеров.

Все backend-реплики используют одно хранилище и дописывают заказы в общий файл.

## Безопасность

В production применяются:

* запуск контейнеров без `root`;
* `read_only: true`;
* отдельные `tmpfs` для временных файлов;
* `cap_drop: [ALL]`;
* `no-new-privileges:true`;
* ограничения CPU, памяти, количества процессов и файловых дескрипторов;
* ротация логов;
* изолированная `internal`-сеть для backend;
* Docker secrets для чувствительных данных;
* basic-auth для `/metrics`;
* Trivy-сканирование образов и репозитория в CI.

Секреты монтируются в `/run/secrets/` и не входят в Docker-образы.

Подробнее — в [SECURITY.md](SECURITY.md).