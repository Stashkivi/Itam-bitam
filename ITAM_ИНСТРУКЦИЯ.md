# ITAM — Руководство по развёртыванию и эксплуатации

> Версия: 1.0 · Дата: 2026-05-26

---

## Оглавление

1. [Архитектура системы](#1-архитектура-системы)
2. [Требования к инфраструктуре](#2-требования-к-инфраструктуре)
3. [Подготовка сервера](#3-подготовка-сервера)
4. [Первый запуск — Docker Compose](#4-первый-запуск--docker-compose)
5. [Настройка переменных окружения](#5-настройка-переменных-окружения)
6. [Регистрация первого агента](#6-регистрация-первого-агента)
7. [Работа с дашбордом](#7-работа-с-дашбордом)
8. [Панель администратора](#8-панель-администратора)
9. [Настройка алертов](#9-настройка-алертов)
10. [Аномалии и базовые линии](#10-аномалии-и-базовые-линии)
11. [Диагностика и логи](#11-диагностика-и-логи)
12. [TLS-сертификаты для продакшна](#12-tls-сертификаты-для-продакшна)
13. [Резервное копирование](#13-резервное-копирование)
14. [Масштабирование](#14-масштабирование)
15. [Обновление компонентов](#15-обновление-компонентов)
16. [Мониторинг самой ITAM-системы](#16-мониторинг-самой-itam-системы)
17. [Частые проблемы (FAQ)](#17-частые-проблемы-faq)

---

## 1. Архитектура системы

```
┌──────────────────────────────────────────────────────┐
│                  ITAM Platform                        │
│                                                      │
│  ┌─────────┐     HTTPS/mTLS      ┌──────────────┐   │
│  │  Agent  │ ──────────────────► │ itam-server  │   │
│  │ (Go)    │                     │  (Go + chi)  │   │
│  └─────────┘                     └──────┬───────┘   │
│                                         │            │
│              ┌──────────────────────────┤            │
│              ▼          ▼          ▼    ▼            │
│         Postgres    Memgraph    Kafka  Redis          │
│          (история)  (граф)   (очереди) (WebSocket)   │
│                                                      │
│  ┌─────────────────┐   WebSocket   ┌──────────────┐  │
│  │   itam-ui       │ ◄──────────── │ itam-server  │  │
│  │  (React + nginx)│               │  /v1/ws      │  │
│  └─────────────────┘               └──────────────┘  │
└──────────────────────────────────────────────────────┘
```

**Компоненты:**

| Компонент | Технология | Назначение |
|-----------|-----------|-----------|
| Agent | Go 1.22, gopsutil | Сбор данных с хоста каждые N минут |
| itam-server | Go, chi, pgxpool | API, Kafka-воркеры, аномалии, алерты |
| PostgreSQL 16 | Реляционная БД | Хосты, снапшоты, дифы, токены, алерты |
| Memgraph 2.17 | Граф (Bolt) | Живой граф зависимостей |
| Kafka (KRaft) | Confluent 7.7 | Буфер для 10K+ агентов |
| Redis 7 | KV + PubSub | Дедупликация алертов, WebSocket |
| itam-ui | React 18, Vite, nginx | Дашборд и панель администратора |

---

## 2. Требования к инфраструктуре

### Минимальные требования (до 100 агентов)

| Ресурс | Значение |
|--------|---------|
| CPU | 4 vCPU |
| RAM | 8 GB |
| Диск | 100 GB SSD |
| ОС | Ubuntu 22.04 / Debian 12 / RHEL 9 |
| Docker | 24.x+ |
| Docker Compose | v2.20+ |

### Рекомендуемые (до 5000 агентов)

| Ресурс | Значение |
|--------|---------|
| CPU | 16 vCPU |
| RAM | 32 GB |
| Диск | 1 TB NVMe |
| Сеть | 1 Gbps |

### Сетевые порты

| Порт | Назначение |
|------|-----------|
| 80 | HTTP → itam-ui (nginx) |
| 443 | HTTPS → itam-ui (после настройки TLS) |
| 8443 | API-сервер (только внутренняя сеть или reverse proxy) |
| 7687 | Memgraph Bolt (только localhost) |

---

## 3. Подготовка сервера

```bash
# Установка Docker (Ubuntu/Debian)
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
newgrp docker

# Проверка версий
docker --version           # должен быть 24.x+
docker compose version     # должен быть 2.20+

# Создание директории проекта
mkdir -p /opt/itam
cd /opt/itam

# Клонирование репозиториев (или копирование папок)
# Если используете Desktop-папки:
cp -r ~/Desktop/itam        ./itam
cp -r ~/Desktop/itam-server ./itam-server
cp -r ~/Desktop/itam-ui     ./itam-ui
cp    ~/Desktop/docker-compose.yml .
cp    ~/Desktop/.env.example .env
```

---

## 4. Первый запуск — Docker Compose

### 4.1 Настройка .env

```bash
# Открыть файл и заполнить секреты
nano /opt/itam/.env
```

Пример заполненного `.env`:

```env
POSTGRES_DB=itam
POSTGRES_USER=itam
POSTGRES_PASSWORD=Str0ng_Passw0rd_Here!

# Сгенерировать кластер-ID для Kafka KRaft
# docker run --rm confluentinc/cp-kafka:7.7.1 kafka-storage random-uuid
KAFKA_CLUSTER_ID=abc123XYZ-uuid-here

JWT_SECRET=super-secret-jwt-key-at-least-32-characters
ENROLLMENT_SECRET=super-secret-enrollment-hmac-key-32ch
ADMIN_TOKEN=my-admin-api-token-for-itam-admin-panel
```

### 4.2 Инициализация Kafka KRaft

Перед первым запуском Kafka нужно форматировать хранилище:

```bash
# Запустить временный контейнер для форматирования
docker run --rm \
  -e KAFKA_CLUSTER_ID=$(grep KAFKA_CLUSTER_ID .env | cut -d= -f2) \
  -v itam_kafka_data:/var/lib/kafka/data \
  confluentinc/cp-kafka:7.7.1 \
  kafka-storage format \
    --config /etc/kafka/kraft/server.properties \
    --cluster-id $(grep KAFKA_CLUSTER_ID .env | cut -d= -f2) \
    --ignore-formatted
```

### 4.3 Сборка и запуск

```bash
cd /opt/itam

# Собрать все образы
docker compose build

# Запустить стек (в фоне)
docker compose up -d

# Проверить статус контейнеров
docker compose ps

# Ожидаемый вывод:
# NAME           STATUS          PORTS
# itam-postgres  Up (healthy)    5432/tcp
# itam-memgraph  Up (healthy)    7687/tcp
# itam-kafka     Up (healthy)    9092/tcp
# itam-redis     Up (healthy)    6379/tcp
# itam-server    Up (healthy)    0.0.0.0:8443->8443/tcp
# itam-ui        Up              0.0.0.0:80->80/tcp
```

### 4.4 Проверка работоспособности

```bash
# Health-check сервера
curl http://localhost:8443/healthz
# {"status":"ok"}

# Проверить дашборд
curl -I http://localhost:80
# HTTP/1.1 200 OK
```

---

## 5. Настройка переменных окружения

Все переменные можно задать двумя способами:

1. **Файл `.env`** — рекомендуется для Docker Compose
2. **Env-переменные напрямую** — для Kubernetes / systemd

| Переменная | Описание | Пример |
|-----------|---------|-------|
| `POSTGRES_PASSWORD` | Пароль PostgreSQL | `Strong_Pass_Here` |
| `JWT_SECRET` | Ключ подписи JWT (мин. 32 символа) | `random_32_char_string` |
| `ENROLLMENT_SECRET` | HMAC-ключ для токенов агентов | `another_32_char_string` |
| `ADMIN_TOKEN` | Токен для Admin API и UI | `admin_api_token` |
| `KAFKA_CLUSTER_ID` | UUID кластера Kafka KRaft | сгенерировать один раз |

### Генерация безопасных секретов

```bash
# JWT_SECRET и ENROLLMENT_SECRET
openssl rand -hex 32

# ADMIN_TOKEN
openssl rand -hex 24
```

---

## 6. Регистрация первого агента

### 6.1 Создание токена через Admin UI

1. Открыть `http://YOUR_SERVER/admin`
2. Ввести `ADMIN_TOKEN`
3. Перейти на вкладку **Tokens**
4. Заполнить:
   - **Org ID** — идентификатор вашей организации (например, `acme-corp`)
   - **Label** — метка для удобства (например, `prod-web-01`)
5. Нажать **Create**
6. **Скопировать токен** — он показывается только один раз!

### 6.2 Создание токена через curl (альтернатива)

```bash
curl -X POST http://localhost:8443/v1/admin/tokens \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"org_id":"acme-corp","label":"prod-web-01"}'

# Ответ:
# {"token":"a1b2c3d4...64hex_chars","note":"share with agent; single-use only"}
```

### 6.3 Настройка агента (Linux)

```bash
# Скопировать бинарник на целевой хост
scp agent user@10.0.0.5:/usr/local/bin/itam-agent
chmod +x /usr/local/bin/itam-agent

# Создать конфиг
cat > /etc/itam/config.yaml << 'EOF'
server_url: "https://itam.your-company.com:8443"
enrollment_token: "a1b2c3d4...ваш_токен_здесь"
org_id: "acme-corp"
scan_interval: "@every 5m"
EOF

# Создать systemd-юнит
cat > /etc/systemd/system/itam-agent.service << 'EOF'
[Unit]
Description=ITAM Agent
After=network.target

[Service]
ExecStart=/usr/local/bin/itam-agent -config /etc/itam/config.yaml
Restart=always
RestartSec=30
User=root

[Install]
WantedBy=multi-user.target
EOF

systemctl enable --now itam-agent
systemctl status itam-agent
```

### 6.4 Агент на Windows

```powershell
# Скопировать agent.exe в C:\Program Files\ITAM\

# Создать конфиг
New-Item -Force -Path "C:\ProgramData\ITAM" -ItemType Directory
@"
server_url: "https://itam.your-company.com:8443"
enrollment_token: "a1b2c3d4...ваш_токен_здесь"
org_id: "acme-corp"
scan_interval: "@every 5m"
"@ | Out-File -Encoding utf8 "C:\ProgramData\ITAM\config.yaml"

# Установить как Windows Service
sc.exe create ITAM-Agent `
  binPath= "\"C:\Program Files\ITAM\agent.exe\" -config \"C:\ProgramData\ITAM\config.yaml\"" `
  start= auto

sc.exe start ITAM-Agent
```

### 6.5 Процесс автоматической регистрации

При первом запуске агент:
1. Генерирует Ed25519-ключевую пару
2. Отправляет `POST /v1/enroll` с токеном и публичным ключом
3. Получает подписанный клиентский сертификат от внутреннего CA
4. Сохраняет сертификат, ключ и CA в `~/.itam/` (или `C:\ProgramData\ITAM\`)
5. Дальнейшие запросы используют mTLS + JWT

---

## 7. Работа с дашбордом

### 7.1 Вход

Откройте `http://YOUR_SERVER` и введите JWT-токен доступа. Токен можно получить, вызвав endpoint регистрации с валидным агентским токеном или временно через API (для администраторского доступа к UI используйте `/admin`).

### 7.2 Граф зависимостей

- **Синий узел** — хост (физический или виртуальный)
- **Зелёный узел** — systemd/Windows-сервис
- **Фиолетовый узел** — процесс (кликнуть для просмотра PID, CPU, RAM)
- **Жёлтый узел** — открытый порт (🌐 = слушает на 0.0.0.0, 🔒 = только localhost)

**Навигация:**
- Колесо мыши — масштаб
- Перетаскивание — панорама
- Двойной клик по ProcessNode — разворачивает/сворачивает детали
- Мини-карта (нижний правый угол) — быстрая навигация

### 7.3 Аномалии в реальном времени

Когда агент отправляет скан:
1. Аномальные узлы загораются пульсирующим ореолом (цвет = severity)
2. Тост-уведомление появляется в правом верхнем углу (6 сек для LOW/MEDIUM, остаётся для HIGH/CRITICAL)
3. Панель **Anomaly Feed** слева показывает историю

Цвета severity:
- 🔴 **CRITICAL** — красный пульс
- 🟠 **HIGH** — оранжевый пульс
- 🟡 **MEDIUM** — жёлтый пульс
- 🔵 **LOW** — синий пульс

---

## 8. Панель администратора

Доступна по адресу `http://YOUR_SERVER/admin`.

### 8.1 Вкладка Tokens

Управление токенами регистрации агентов:

| Поле | Описание |
|------|---------|
| Org ID | Идентификатор организации (произвольная строка) |
| Label | Метка для идентификации агента |
| Used | Был ли токен использован |

Токен однократный — после использования агентом помечается как `used`.

### 8.2 Вкладка Channels

Настройка каналов для уведомлений об аномалиях:

**Slack:**
- Создать Incoming Webhook в `api.slack.com/apps`
- Вставить URL в поле **Slack Webhook URL**

**Telegram:**
- Создать бота через `@BotFather`
- Получить **Bot Token** и **Chat ID**
- Chat ID можно найти через `https://api.telegram.org/bot<TOKEN>/getUpdates`

### 8.3 Вкладка Hosts

Список всех зарегистрированных хостов. Кнопка **Approve baseline** фиксирует текущий снапшот как эталонное состояние (базовую линию) — после этого любые отклонения будут генерировать аномалии.

---

## 9. Настройка алертов

### 9.1 Slack

```bash
# Создать канал через API
curl -X POST http://localhost:8443/v1/admin/channels \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "org_id": "acme-corp",
    "channel_type": "slack",
    "config": {
      "webhook_url": "https://hooks.slack.com/services/T.../B.../..."
    }
  }'
```

Пример уведомления в Slack:
```
🔴 ITAM Alert — CRITICAL
Host: prod-web-01 (acme-corp)
Rule: PORT_BIND_CHANGE
Details: Port 22/tcp changed bind from 127.0.0.1 to 0.0.0.0
Time: 2026-05-26 14:23:05 UTC
```

### 9.2 Telegram

```bash
curl -X POST http://localhost:8443/v1/admin/channels \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "org_id": "acme-corp",
    "channel_type": "telegram",
    "config": {
      "bot_token": "1234567890:AAF...",
      "chat_id": "-1001234567890"
    }
  }'
```

### 9.3 Правила аномалий

| Правило | Severity | Описание |
|---------|---------|---------|
| `PORT_BIND_CHANGE` | CRITICAL | Порт изменил адрес прослушивания |
| `UNKNOWN_PORT` | HIGH | Порт, не входящий в базовую линию |
| `SERVICE_CRASHED` | HIGH | Сервис упал (status = failed) |
| `NEW_SERVICE` | MEDIUM | Появился новый systemd-сервис |
| `UNKNOWN_PROCESS` | MEDIUM | Неизвестный процесс под root |

**Дедупликация:** одна и та же аномалия не будет повторно рассылаться в течение 15 минут (Redis TTL). WebSocket-событие всё равно доставляется.

---

## 10. Аномалии и базовые линии

### 10.1 Логика работы

```
Первый скан → AutoApproveFirstScan → базовая линия создана
Следующий скан → сравнение с базовой линией → выявление отклонений
```

### 10.2 Ручное утверждение базовой линии

```bash
# Через API
curl -X PUT http://localhost:8443/v1/admin/hosts/{HOST_UUID}/baseline \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Через Admin UI: вкладка Hosts → "Approve baseline"
```

### 10.3 Сброс базовой линии

```sql
-- Деактивировать все базовые линии хоста
UPDATE baselines SET is_active = false
WHERE host_id = 'ваш-host-uuid';

-- После следующего скана агента база будет создана заново
```

---

## 11. Диагностика и логи

### 11.1 Просмотр логов

```bash
# Все компоненты
docker compose logs -f

# Только сервер
docker compose logs -f itam-server

# Только агент (если запущен через systemd)
journalctl -u itam-agent -f
```

### 11.2 Проверка подключения Kafka

```bash
# Список топиков
docker compose exec kafka \
  kafka-topics --bootstrap-server localhost:9092 --list

# Количество сообщений в очереди
docker compose exec kafka \
  kafka-consumer-groups \
    --bootstrap-server localhost:9092 \
    --describe --group itam-pg-writer
```

### 11.3 Состояние PostgreSQL

```bash
# Подключиться к БД
docker compose exec postgres psql -U itam -d itam

# Количество хостов
SELECT COUNT(*) FROM hosts WHERE status = 'active';

# Последние аномалии
SELECT h.hostname, a.rule_id, a.severity, a.detected_at
FROM anomalies a
JOIN hosts h ON h.id = a.host_id
ORDER BY a.detected_at DESC
LIMIT 20;
```

### 11.4 Граф Memgraph

```bash
# Открыть Memgraph Lab (если порт пробросить)
docker compose exec memgraph mgconsole

# Количество узлов
MATCH (n) RETURN labels(n), count(n);

# Граф конкретного хоста
MATCH (h:Host {uuid: "your-host-uuid"})-[*1..3]-(n)
RETURN h, n LIMIT 100;
```

---

## 12. TLS-сертификаты для продакшна

### 12.1 Let's Encrypt через Caddy (рекомендуется)

```bash
# Установить Caddy как reverse proxy перед itam-ui
cat > /opt/itam/Caddyfile << 'EOF'
itam.your-company.com {
    reverse_proxy itam-ui:80
}

api.itam.your-company.com {
    reverse_proxy itam-server:8443
}
EOF

# Добавить Caddy в docker-compose.yml:
# caddy:
#   image: caddy:2-alpine
#   ports: ["80:80", "443:443"]
#   volumes:
#     - ./Caddyfile:/etc/caddy/Caddyfile
#     - caddy_data:/data
#   depends_on: [itam-ui, itam-server]
```

### 12.2 Nginx + certbot (альтернатива)

```bash
# На хосте (вне Docker)
certbot certonly --standalone -d itam.your-company.com

# Смонтировать сертификаты в nginx
# volumes:
#   - /etc/letsencrypt:/etc/letsencrypt:ro
```

### 12.3 Сертификат для mTLS агента

Внутренний CA создаётся автоматически при первом запуске сервера и хранится в Docker-томе `ca_data`. Агент получает клиентский сертификат в процессе enrollment — дополнительной настройки не требуется.

Для просмотра CA-сертификата:

```bash
docker compose exec itam-server cat /data/ca/ca.crt
```

---

## 13. Резервное копирование

### 13.1 PostgreSQL

```bash
# Дамп базы данных
docker compose exec postgres pg_dump -U itam itam | \
  gzip > backup_$(date +%Y%m%d_%H%M%S).sql.gz

# Восстановление
gunzip -c backup_20260526_120000.sql.gz | \
  docker compose exec -T postgres psql -U itam -d itam
```

### 13.2 Docker volumes

```bash
# Резервная копия всех томов
for vol in pg_data mg_data redis_data ca_data; do
  docker run --rm \
    -v itam_${vol}:/data:ro \
    -v $(pwd)/backups:/backup \
    alpine tar czf /backup/${vol}_$(date +%Y%m%d).tar.gz -C /data .
done
```

### 13.3 Рекомендуемый schedule

| Что | Как часто | Retention |
|-----|---------|---------|
| PostgreSQL full dump | Ежедневно | 30 дней |
| CA volume | Еженедельно | 1 год |
| Memgraph snapshot | Еженедельно | 7 дней |

---

## 14. Масштабирование

### 14.1 Горизонтальное масштабирование itam-server

```yaml
# docker-compose.yml
itam-server:
  deploy:
    replicas: 3
```

Серверы без состояния — можно запустить сколько угодно экземпляров. Kafka-партиции (32 шт.) позволяют 32 параллельных воркера.

### 14.2 PostgreSQL connection pooling (PgBouncer)

```yaml
pgbouncer:
  image: edoburu/pgbouncer:latest
  environment:
    DB_USER: itam
    DB_PASSWORD: ${POSTGRES_PASSWORD}
    DB_HOST: postgres
    POOL_MODE: transaction
    MAX_CLIENT_CONN: 1000
    DEFAULT_POOL_SIZE: 50
```

### 14.3 Автоматическое создание партиций

Для автоматической ежемесячной партиционирования таблиц:

```sql
-- Установить pg_partman
CREATE EXTENSION IF NOT EXISTS pg_partman;

SELECT partman.create_parent(
  'public.host_snapshots',
  'captured_at',
  'native',
  'monthly',
  p_start_partition := '2026-01-01'
);

-- Добавить в crontab сервера:
-- 0 0 1 * * docker compose exec postgres \
--   psql -U itam -c "SELECT partman.run_maintenance();"
```

---

## 15. Обновление компонентов

### 15.1 Обновление itam-server

```bash
cd /opt/itam

# Обновить код
git pull  # или скопировать новые файлы

# Пересобрать и перезапустить только сервер
docker compose build itam-server
docker compose up -d --no-deps itam-server

# Проверить версию
docker compose logs itam-server | grep "listening"
```

### 15.2 Обновление itam-ui

```bash
docker compose build itam-ui
docker compose up -d --no-deps itam-ui
```

### 15.3 Обновление агентов

```bash
# Собрать бинарник
cd itam && go build -o agent ./cmd/agent

# Распространить через Ansible
ansible all -m copy -a "src=agent dest=/usr/local/bin/itam-agent mode=0755"
ansible all -m systemd -a "name=itam-agent state=restarted"
```

### 15.4 Миграции БД

Миграции применяются автоматически при старте itam-server. Для ручного применения:

```bash
docker compose exec postgres psql -U itam -d itam -f /migrations/003_new_feature.sql
```

---

## 16. Мониторинг самой ITAM-системы

### 16.1 Health checks

```bash
# Простой мониторинг через cron
*/5 * * * * curl -sf http://localhost:8443/healthz || \
  echo "ITAM server DOWN" | mail -s "ALERT" ops@company.com
```

### 16.2 Метрики (Prometheus-ready)

Добавьте в `itam-server` эндпоинт `/metrics` для Prometheus:

```bash
# Ключевые метрики для алертов:
# - itam_kafka_consumer_lag > 1000 → воркеры не справляются
# - itam_anomalies_total{severity="CRITICAL"} — тренд аномалий
# - itam_agents_last_seen_age_seconds > 3600 → агент не выходит на связь
```

### 16.3 Grafana дашборд

```bash
# Запустить Grafana
docker run -d \
  --name grafana \
  --network itam_default \
  -p 3000:3000 \
  -v grafana_data:/var/lib/grafana \
  grafana/grafana:latest
```

Импортировать дашборды для PostgreSQL, Kafka, Redis из Grafana Labs (IDs: 9628, 7589, 11835).

### 16.4 Алерты на сам стек

В `alertmanager` настройте:

| Условие | Action |
|---------|--------|
| itam-server не отвечает на `/healthz` | Page on-call |
| Kafka consumer lag > 5000 | Slack #ops |
| PostgreSQL connections > 80% | Slack #dba |
| Redis memory > 90% | Slack #ops |

---

## 17. Частые проблемы (FAQ)

### Агент не регистрируется

```bash
# Проверить токен
curl -s http://localhost:8443/v1/admin/tokens \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.[] | select(.used==false)'

# Проверить доступность сервера с агента
curl -v https://itam.your-company.com:8443/healthz

# Проверить логи агента
journalctl -u itam-agent --since "5 min ago"
```

**Частые причины:**
- Токен уже использован (однократный)
- Неверный `server_url` в конфиге агента
- Блокировка фаерволом порта 8443

### Граф не обновляется

```bash
# Проверить подключение к Memgraph
docker compose exec memgraph mgconsole <<< "MATCH (n) RETURN count(n);"

# Проверить graph-writer воркер
docker compose logs itam-server | grep "graph-writer"
```

### Аномалии не приходят в Slack/Telegram

```bash
# Проверить каналы
curl http://localhost:8443/v1/admin/channels \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .

# Проверить Kafka топик алертов
docker compose exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic agent.alert --from-beginning --max-messages 5
```

### WebSocket не подключается

```bash
# Проверить nginx конфиг — должен быть proxy_set_header Upgrade
docker compose exec itam-ui nginx -t

# Проверить CORS и токен в браузере
# F12 → Network → WS → смотреть статус 101 Switching Protocols
```

### PostgreSQL переполняется

```bash
# Посмотреть размер таблиц
docker compose exec postgres psql -U itam -d itam -c "
SELECT tablename, pg_size_pretty(pg_total_relation_size(tablename::text))
FROM pg_tables WHERE schemaname='public'
ORDER BY pg_total_relation_size(tablename::text) DESC LIMIT 10;
"

# Очистить старые снапшоты (старше 90 дней)
docker compose exec postgres psql -U itam -d itam -c "
DELETE FROM host_snapshots WHERE captured_at < now() - interval '90 days';
"
```

---

## Контакты и поддержка

- Репозиторий: `https://github.com/your-org/itam`
- Issues: открывать тикеты с тегом `bug` или `question`
- Логи для отчёта об ошибке: `docker compose logs --tail=100 itam-server`
