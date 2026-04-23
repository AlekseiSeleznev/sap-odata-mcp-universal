# sap-odata-mcp-universal

Единый MCP-сервер для работы с SAP OData из AI-ассистентов и любых MCP-клиентов

Один endpoint `http://localhost:8080/mcp` вместо ручного запуска отдельного bridge под каждую SAP-систему. В HTTP-режиме сервер поднимает двуязычный dashboard на `http://localhost:8080/dashboard`, где можно сохранять несколько SAP OData-подключений по логину и паролю, переключать активную систему без рестарта и сразу отдавать актуальный набор MCP-инструментов.

`sap-odata-mcp-universal` поддерживает:

- SAP OData v2 и OData v4
- `stdio`, legacy `http`/SSE и современный `streamable-http`
- базовую аутентификацию, cookie auth и header forwarding
- read-only и write-capable профили
- обычный per-entity режим и `--universal` для больших сервисов
- двуязычный web dashboard в стиле `postgres-mcp-universal`

---

## Содержание

- [Возможности](#возможности)
- [Архитектура](#архитектура)
- [Требования](#требования)
- [Быстрый старт](#быстрый-старт)
  - [Рекомендуемый режим: HTTP + dashboard](#рекомендуемый-режим-http--dashboard)
  - [Режим одного сервиса: stdio](#режим-одного-сервиса-stdio)
- [Установка](#установка)
  - [Сборка из исходников](#сборка-из-исходников)
  - [Makefile](#makefile)
  - [Docker](#docker)
- [Подключение к AI-ассистенту](#подключение-к-ai-ассистенту)
  - [Codex](#codex)
  - [Claude Desktop](#claude-desktop)
  - [Cursor / Windsurf / другие HTTP MCP-клиенты](#cursor--windsurf--другие-http-mcp-клиенты)
- [Веб-дашборд](#веб-дашборд)
  - [Что умеет dashboard](#что-умеет-dashboard)
  - [Поля подключения](#поля-подключения)
  - [Режимы доступа](#режимы-доступа)
  - [Хранение состояния](#хранение-состояния)
  - [HTTP endpoints](#http-endpoints)
- [Режимы работы MCP-инструментов](#режимы-работы-mcp-инструментов)
  - [Per-entity режим](#per-entity-режим)
  - [Universal mode](#universal-mode)
- [Аутентификация и конфигурация](#аутентификация-и-конфигурация)
  - [Переменные окружения](#переменные-окружения)
  - [Ключевые CLI-флаги](#ключевые-cli-флаги)
- [Безопасность](#безопасность)
- [Разработка и тестирование](#разработка-и-тестирование)
- [Диагностика](#диагностика)
- [Лицензия](#лицензия)

---

## Возможности

| Категория | Что есть |
|---|---|
| **SAP OData** | Автозагрузка metadata, генерация MCP tools, поддержка SAP-специфики |
| **Транспорты** | `stdio`, `http` (SSE), `streamable-http` |
| **Dashboard** | Несколько сохранённых SAP систем, переключение активной системы, RU/EN UI и docs |
| **Аутентификация** | Basic auth, cookie file, cookie string, optional header forwarding |
| **Ограничение операций** | `--read-only`, `--read-only-but-functions`, `--enable`, `--disable` |
| **Фильтрация инструментов** | `--entities`, `--functions`, `--tool-shrink`, custom prefix/postfix |
| **Ответы и диагностика** | `--pagination-hints`, `--response-metadata`, `--verbose-errors`, `--trace`, `--trace-mcp` |
| **Большие сервисы** | `--universal` для одного универсального инструмента вместо сотен entity-tools |
| **HTTP эксплуатация** | `/mcp`, `/health`, `/dashboard`, `/dashboard/docs`, API управления подключениями |

Ключевые особенности:

- **Dashboard-first сценарий**. HTTP-сервер может стартовать вообще без `--service`, а активная SAP-система выбирается потом через web UI.
- **Горячее переключение системы**. При смене активного подключения bridge пересобирает MCP tools без перезапуска процесса.
- **Персистентный реестр подключений**. Сохранённые профили лежат на диске и могут автоматически восстанавливаться после рестарта.
- **Режим доступа на уровне профиля**. Для каждого SAP-подключения можно включать запись или принудительно оставлять только read-only инструменты.
- **Совместимость с большими SAP сервисами**. `--universal` резко снижает число инструментов и токеновую нагрузку на MCP-клиент.

---

## Архитектура

```text
MCP clients / AI assistants
        |
        | stdio
        | или HTTP POST /mcp
        v
+--------------------------------------------------+
| sap-odata-mcp-universal                          |
|                                                  |
|  Transport layer                                 |
|  - stdio                                         |
|  - HTTP/SSE                                      |
|  - Streamable HTTP                               |
|                                                  |
|  Dashboard layer (/dashboard)                    |
|  - список SAP OData подключений                  |
|  - connect / edit / switch / disconnect          |
|  - RU/EN UI и документация                       |
|                                                  |
|  Bridge layer                                    |
|  - загрузка metadata                             |
|  - SAP auth / cookies / CSRF                     |
|  - генерация MCP tools                           |
|  - hot reconfigure активной системы              |
|                                                  |
|  State file                                      |
|  - active connection                             |
|  - saved profiles                                |
|  - access mode                                   |
+--------------------------------------------------+
        |
        v
 SAP OData service
 https://host/sap/opu/odata/sap/...
```

В `stdio` режиме процесс обычно обслуживает одну конкретную SAP OData систему.

В `streamable-http` или `http` режиме процесс можно использовать как локальный MCP gateway:

- открываете `http://localhost:8080/dashboard`
- сохраняете несколько SAP-систем
- выбираете активную
- MCP-клиент работает через один и тот же `/mcp`

---

## Требования

- **Go 1.21+** для локальной сборки
- **Linux, macOS или Windows**
- доступ к SAP OData service
- логин/пароль или cookie-аутентификация
- для HTTP-режима:
  - локально: обязателен `--mcp-token`
  - вне `localhost`: обязательны `--mcp-token` и `--tls`

---

## Быстрый старт

### Рекомендуемый режим: HTTP + dashboard

Это основной сценарий для нескольких SAP-систем.

```bash
go build -o sap-odata-mcp-universal ./cmd/sap-odata-mcp-universal

./sap-odata-mcp-universal \
  --transport streamable-http \
  --http-addr localhost:8080 \
  --mcp-token dev-token
```

После старта:

- dashboard: `http://localhost:8080/dashboard`
- документация: `http://localhost:8080/dashboard/docs`
- MCP endpoint: `http://localhost:8080/mcp`
- health check: `http://localhost:8080/health`

Дальше:

1. Откройте dashboard.
2. Добавьте SAP OData подключение.
3. Нажмите `Подключить`.
4. Выберите его как активное.
5. Подключите AI-клиент к `/mcp`.

Важно: в HTTP-режиме токен обязателен даже на `localhost`.

### Режим одного сервиса: stdio

Это простой вариант, если вам не нужен dashboard и вы работаете с одной SAP-системой.

```bash
./sap-odata-mcp-universal \
  --service https://host/sap/opu/odata/sap/API_SALES_ORDER_SRV/ \
  --user YOUR_LOGIN \
  --password YOUR_PASSWORD
```

Также можно передавать URL позиционным аргументом:

```bash
./sap-odata-mcp-universal \
  --user YOUR_LOGIN \
  --password YOUR_PASSWORD \
  https://host/sap/opu/odata/sap/API_SALES_ORDER_SRV/
```

---

## Установка

### Сборка из исходников

```bash
git clone https://github.com/AlekseiSeleznev/sap-odata-mcp-universal.git
cd sap-odata-mcp-universal
go build -o sap-odata-mcp-universal ./cmd/sap-odata-mcp-universal
```

Кросс-компиляция вручную:

```bash
GOOS=linux GOARCH=amd64 go build -o sap-odata-mcp-universal-linux-amd64 ./cmd/sap-odata-mcp-universal
GOOS=windows GOARCH=amd64 go build -o sap-odata-mcp-universal-windows-amd64.exe ./cmd/sap-odata-mcp-universal
GOOS=darwin GOARCH=amd64 go build -o sap-odata-mcp-universal-darwin-amd64 ./cmd/sap-odata-mcp-universal
GOOS=darwin GOARCH=arm64 go build -o sap-odata-mcp-universal-darwin-arm64 ./cmd/sap-odata-mcp-universal
```

### Makefile

Основные команды:

```bash
make build
make build-all
make test
make test-race
make fmt
make docker
make clean
```

Полезные таргеты:

- `make build` — локальная сборка
- `make build-all` — Linux, Windows, macOS
- `make build-windows-wsl` — Windows binary + копирование в `/mnt/c/bin`
- `make test-all` — полный прогон с regression/E2E
- `make version` — версия по git tag / commit count

### Docker

```bash
docker build -t sap-odata-mcp-universal .

docker run --rm -it \
  -p 8080:8080 \
  sap-odata-mcp-universal \
  --transport streamable-http \
  --http-addr 0.0.0.0:8080 \
  --allow-all-interfaces \
  --mcp-token CHANGE_ME \
  --tls \
  --tls-cert /path/to/cert.pem \
  --tls-key /path/to/key.pem
```

Если контейнеру доступен каталог `/data`, состояние dashboard по умолчанию хранится в `/data/odata_state.json`.

---

## Подключение к AI-ассистенту

### Codex

Есть два практических сценария.

`stdio` для одной системы:

```bash
codex mcp add sap-odata-universal \
  -- /absolute/path/to/sap-odata-mcp-universal \
  --service https://host/sap/opu/odata/sap/API_SALES_ORDER_SRV/ \
  --user YOUR_LOGIN \
  --password YOUR_PASSWORD
```

`HTTP` для dashboard-сценария:

```bash
export SAP_ODATA_MCP_TOKEN=dev-token

codex mcp add sap-odata-universal \
  --url http://localhost:8080/mcp \
  --bearer-token-env-var SAP_ODATA_MCP_TOKEN
```

Условия:

- сервер поднят с `--transport streamable-http`
- `SAP_ODATA_MCP_TOKEN` совпадает с `--mcp-token`
- активная SAP-система уже выбрана в dashboard

Если ваш MCP-клиент не умеет передавать HTTP token, используйте `stdio` или промежуточный прокси.

### Claude Desktop

Пример `claude_desktop_config.json` для одной SAP-системы:

```json
{
  "mcpServers": {
    "sap-odata-universal": {
      "command": "/usr/local/bin/sap-odata-mcp-universal",
      "args": [
        "--service",
        "https://host/sap/opu/odata/sap/API_SALES_ORDER_SRV/",
        "--user",
        "YOUR_LOGIN",
        "--password",
        "YOUR_PASSWORD",
        "--tool-shrink"
      ]
    }
  }
}
```

Если нужен именно dashboard-сценарий с несколькими системами, используйте HTTP MCP-клиент или отдельную локальную схему подключения к `/mcp`.

### Cursor / Windsurf / другие HTTP MCP-клиенты

Рекомендуемый transport:

```bash
./sap-odata-mcp-universal \
  --transport streamable-http \
  --http-addr localhost:8080 \
  --mcp-token dev-token
```

Подключение:

- MCP URL: `http://localhost:8080/mcp`
- dashboard: `http://localhost:8080/dashboard`
- token: значение `--mcp-token`

Для старых клиентов можно использовать `--transport http`, но основной режим для новых MCP-клиентов — `streamable-http`.

---

## Веб-дашборд

### Что умеет dashboard

Dashboard открывается на:

- `GET /dashboard` — основной интерфейс
- `GET /dashboard/docs` — встроенная подробная документация

Он нужен для сценария, где один процесс обслуживает несколько SAP OData систем.

Через UI можно:

- сохранить несколько SAP OData профилей
- подключить новый профиль и сразу сделать его активным
- переключить активную систему без рестарта процесса
- отредактировать профиль
- удалить профиль
- переключить язык интерфейса `RU/EN`

Dashboard запускается в стиле `postgres-mcp-universal`:

- слева список сохранённых подключений
- справа форма нового или редактируемого подключения
- сверху переключение языка, документация и refresh
- UI не подставляет логин/пароль по умолчанию

### Поля подключения

| Поле | Назначение |
|---|---|
| `Имя соединения` | внутреннее имя профиля в dashboard |
| `Имя системы` | человекочитаемое имя SAP-ландшафта, например `S4HANA QA` |
| `URL сервиса` | корневой SAP OData URL |
| `SAP клиент` | optional `sap-client`, добавляется в query string автоматически |
| `Логин` | basic auth username |
| `Пароль` | basic auth password |
| `Разрешить запись` | включает write-capable инструменты, если metadata это допускает |

Пример URL:

```text
https://host/sap/opu/odata/sap/API_SALES_ORDER_SRV/
```

### Режимы доступа

Dashboard хранит режим доступа на уровне профиля:

- `restricted` — принудительный read-only
- `unrestricted` — разрешает операции записи, если SAP metadata помечает их как доступные

Если вы выключаете запись в UI, bridge скрывает модифицирующие MCP tools даже при поддержке со стороны сервиса.

### Хранение состояния

Состояние dashboard сохраняется на диск.

По умолчанию:

- в контейнере с каталогом `/data`: `/data/odata_state.json`
- локально: файл в user config directory, внутри каталога `sap-odata-mcp-universal`
- можно переопределить через `ODATA_MCP_STATE_FILE`

В state file хранятся:

- активное подключение
- список сохранённых SAP OData профилей
- режим доступа каждого профиля
- логины и пароли для автопереподключения после рестарта

Пароли в state file **не шифруются**. Это сознательное эксплуатационное ограничение текущей реализации.

### HTTP endpoints

Основные маршруты:

| Endpoint | Метод | Назначение |
|---|---|---|
| `/dashboard` | `GET` | UI dashboard |
| `/dashboard/docs` | `GET` | встроенная документация |
| `/api/status` | `GET` | статус активного подключения |
| `/api/databases` | `GET` | список сохранённых подключений |
| `/api/connect` | `POST` | создать и сразу активировать подключение |
| `/api/edit` | `POST` | изменить сохранённое подключение |
| `/api/switch` | `POST` | переключить активную систему |
| `/api/disconnect` | `POST` | удалить подключение |
| `/health` | `GET` | health endpoint |
| `/mcp` | `POST` | MCP transport endpoint в `streamable-http` |

Root `/` редиректит на `/dashboard`.

---

## Режимы работы MCP-инструментов

### Per-entity режим

Это стандартный режим.

Bridge создаёт отдельные MCP-инструменты под entity sets и function imports, например:

- `search_*`
- `filter_*`
- `get_*`
- `create_*`
- `upd_*`
- `del_*`

Плюсы:

- высокая discoverability
- AI-клиент явно видит структуру сервиса
- удобно для небольших OData-сервисов

Минусы:

- для крупных SAP сервисов инструментов может быть очень много

### Universal mode

Флаг:

```bash
--universal
```

Вместо большого набора entity-tools сервер отдаёт один универсальный OData tool.

Используйте его, если:

- сервис очень большой
- MCP-клиент захлёбывается от числа инструментов
- нужно снизить токеновую нагрузку
- вы подключаете несколько SAP систем и не хотите раздувать toolset

Пример:

```bash
./sap-odata-mcp-universal \
  --transport streamable-http \
  --mcp-token dev-token \
  --universal
```

---

## Аутентификация и конфигурация

### Переменные окружения

Поддерживаются:

- `ODATA_URL`
- `ODATA_SERVICE_URL`
- `ODATA_USERNAME`
- `ODATA_PASSWORD`
- `ODATA_COOKIE_FILE`
- `ODATA_COOKIE_STRING`
- `ODATA_MCP_STATE_FILE`

Приоритет URL:

1. `--service`
2. positional argument
3. `ODATA_URL`
4. `ODATA_SERVICE_URL`

### Ключевые CLI-флаги

| Флаг | Назначение |
|---|---|
| `--service` | URL OData сервиса |
| `--user`, `--password` | basic auth |
| `--cookie-file`, `--cookie-string` | cookie auth |
| `--transport` | `stdio`, `http`, `streamable-http` |
| `--http-addr` | адрес HTTP сервера |
| `--mcp-token` | обязательный token для HTTP transport |
| `--mcp-token-file` | token из файла |
| `--tls`, `--tls-cert`, `--tls-key` | TLS для non-localhost |
| `--allow-all-interfaces` | разрешить bind на `0.0.0.0/::` |
| `--read-only`, `--read-only-but-functions` | ограничение записывающих операций |
| `--enable`, `--disable` | точечное включение/выключение типов операций |
| `--entities`, `--functions` | фильтрация генерируемых tools |
| `--tool-shrink` | сокращённые имена инструментов |
| `--tool-prefix`, `--tool-postfix`, `--no-postfix` | кастомизация имён tools |
| `--forward-mcp-headers` | прокидывать HTTP headers в OData service |
| `--pagination-hints` | подсказки по пагинации в ответах |
| `--response-metadata` | включать `__metadata` в ответы |
| `--verbose-errors` | подробные ошибки |
| `--trace` | вывести tools и завершиться |
| `--trace-mcp` | трассировка MCP transport |
| `--claude-code-friendly` | убрать `$` у параметров для Claude Code CLI |
| `--protocol-version` | override версии MCP протокола |
| `--universal` | один универсальный OData tool |

Примеры:

```bash
./sap-odata-mcp-universal --read-only --service https://host/sap/opu/odata/sap/API_BUSINESS_PARTNER/

./sap-odata-mcp-universal --disable "cud" --service https://host/sap/opu/odata/sap/API_SALES_ORDER_SRV/

./sap-odata-mcp-universal --entities "A_BusinessPartner,A_Address*" --tool-shrink --service https://host/sap/opu/odata/sap/API_BUSINESS_PARTNER/
```

---

## Безопасность

Текущая security-модель жёсткая и должна быть явно учтена в эксплуатации.

HTTP transport:

- на `localhost` обязательно указывать `--mcp-token`
- при bind не на `localhost` обязательно указывать:
  - `--mcp-token`
  - `--tls`
- при bind на `0.0.0.0` или `::` дополнительно нужен:
  - `--allow-all-interfaces`

Рекомендации:

- не публикуйте HTTP endpoint без TLS
- храните token вне shell history через `--mcp-token-file`
- ограничивайте права доступа к state file
- для продуктивных ландшафтов по умолчанию используйте `restricted`
- если включаете `--forward-mcp-headers`, считайте это доверенным контуром

---

## Разработка и тестирование

Локальный цикл:

```bash
go test ./...
go test -race ./...
go build -o sap-odata-mcp-universal ./cmd/sap-odata-mcp-universal
```

Через `make`:

```bash
make fmt
make test
make test-race
make build
make build-all
```

Smoke-check HTTP режима:

```bash
./sap-odata-mcp-universal \
  --transport streamable-http \
  --http-addr localhost:8080 \
  --mcp-token dev-token

curl http://localhost:8080/health
curl http://localhost:8080/dashboard
curl http://localhost:8080/api/status
```

---

## Диагностика

Типичные проблемы:

1. **HTTP сервер не стартует**

Причина:

- нет `--mcp-token`
- bind не на `localhost` без `--tls`
- bind на `0.0.0.0` без `--allow-all-interfaces`

2. **Подключение через dashboard не создаётся**

Проверьте:

- корневой OData URL
- логин/пароль
- доступность metadata endpoint
- нужен ли `sap-client`

3. **Инструменты записи не появились**

Причины:

- профиль сохранён как `restricted`
- указан `--read-only`
- SAP metadata помечает entity set как non-creatable/non-updatable/non-deletable

4. **После рестарта tools пустые**

Проверьте:

- существует ли state file
- валиден ли активный сохранённый профиль
- не истекли ли учётные данные

5. **AI-клиент не может работать через HTTP**

Проверьте:

- правильный `/mcp` URL
- совпадает ли token
- умеет ли клиент передавать token для HTTP MCP

Для дополнительной диагностики используйте:

```bash
./sap-odata-mcp-universal --trace --service https://host/sap/opu/odata/sap/API_SALES_ORDER_SRV/

./sap-odata-mcp-universal --trace-mcp --service https://host/sap/opu/odata/sap/API_SALES_ORDER_SRV/

./sap-odata-mcp-universal --verbose --verbose-errors --service https://host/sap/opu/odata/sap/API_SALES_ORDER_SRV/
```

Встроенная документация dashboard тоже доступна по адресу:

```text
http://localhost:8080/dashboard/docs
```

---

## Лицензия

MIT License

Проект: `https://github.com/AlekseiSeleznev/sap-odata-mcp-universal`
