# Стратегия тестирования sap-odata-mcp-universal

**Документ:** 006  
**Дата:** 2026-04-23  
**Статус:** актуально

---

## Цель

Тестовая стратегия проекта должна покрывать одновременно:

- внутреннюю логику bridge
- генерацию MCP tools
- HTTP transport и dashboard
- интеграционные особенности реальных OData сервисов

Поэтому тестирование разделено на несколько уровней.

---

## Уровень 1. Локальные unit и integration tests

Базовый запуск:

```bash
go test ./...
```

Этот уровень покрывает:

- парсинг metadata
- форматирование запросов и ключей
- генерацию инструментов
- обработку MCP protocol сообщений
- dashboard provider/server
- transport layer

Это основной обязательный набор для каждого изменения.

---

## Уровень 2. Тесты с mock server

Часть тестов использует встроенный mock OData server и не требует внешних зависимостей.

Что это даёт:

- стабильность
- быстрый прогон
- воспроизводимость в CI

Именно этот уровень должен подтверждать, что базовая функциональность проекта не ломается.

---

## Уровень 3. Тесты на публичных OData сервисах

Для реальной проверки полезно использовать:

- Northwind v2
- Northwind v4

Это позволяет проверить:

- реальный HTTP обмен
- поведение против внешнего OData сервиса
- различия между v2 и v4

Такие тесты полезны, но их нельзя считать единственным источником истины для SAP-специфики.

---

## Уровень 4. Тесты на реальном SAP landscape

Это самый ценный уровень для проектных изменений, связанных с SAP:

- metadata parsing
- CSRF
- `sap-client`
- GUID formatting
- ограничения create/update/delete
- различия между DEV/QA/PROD landscapes

Такие проверки обычно требуют:

- `ODATA_URL`
- `ODATA_USERNAME` или `ODATA_USER`
- `ODATA_PASSWORD` или `ODATA_PASS`

---

## Переменные окружения для интеграционных прогонов

Используются:

- `ODATA_URL`
- `ODATA_SERVICE_URL`
- `ODATA_USERNAME`
- `ODATA_PASSWORD`
- `ODATA_USER`
- `ODATA_PASS`

Пример:

```bash
export ODATA_URL="https://host/sap/opu/odata/sap/API_SALES_ORDER_SRV/"
export ODATA_USERNAME="YOUR_LOGIN"
export ODATA_PASSWORD="YOUR_PASSWORD"

go test ./...
```

---

## Проверка race conditions

Для изменения transport, registry, dashboard provider и bridge state обязательно полезно запускать:

```bash
go test -race ./...
```

Это особенно важно для:

- HTTP server
- dashboard connection registry
- hot reconfigure активной SAP-системы

---

## Что проверять после изменений dashboard

После любых изменений dashboard желательно проверить:

1. root redirect на `/dashboard`
2. рендер `/dashboard`
3. рендер `/dashboard/docs`
4. `GET /api/status`
5. `GET /api/databases`
6. `POST /api/connect`
7. `POST /api/edit`
8. `POST /api/switch`
9. `POST /api/disconnect`

И отдельно руками:

- переключение RU/EN
- отсутствие дефолтного логина/пароля
- корректность state file после connect/edit/disconnect

---

## Минимальный validation pipeline перед commit

```bash
gofmt -w $(rg --files -g '*.go')
go test ./...
go test -race ./...
go build -o sap-odata-mcp-universal ./cmd/sap-odata-mcp-universal
```

Если менялась документация dashboard, дополнительно полезно проверить вручную:

```bash
./sap-odata-mcp-universal \
  --transport streamable-http \
  --http-addr localhost:8080 \
  --mcp-token dev-token
```

и открыть:

- `http://localhost:8080/dashboard`
- `http://localhost:8080/dashboard/docs`

---

## Связанные документы

- [README.md](../README.md)
- [003-odata-mcp-e2e-documentation.md](003-odata-mcp-e2e-documentation.md)
