# sap-odata-mcp-universal: сквозной сценарий запуска и проверки

**Документ:** 003  
**Дата:** 2026-04-23  
**Статус:** актуально

---

## Цель

Этот документ описывает рабочий end-to-end сценарий:

1. собрать бинарь
2. поднять HTTP-сервер
3. добавить SAP OData подключение через dashboard
4. подключить MCP-клиент
5. проверить, что MCP endpoint и bridge работают корректно

---

## Предпосылки

Нужно иметь:

- собранный бинарь `sap-odata-mcp-universal`
- доступный SAP OData service
- логин и пароль
- свободный локальный порт, например `8080`

---

## Сценарий 1. Рекомендуемый режим: HTTP + dashboard

### Шаг 1. Сборка

```bash
go build -o sap-odata-mcp-universal ./cmd/sap-odata-mcp-universal
```

### Шаг 2. Старт сервера

```bash
./sap-odata-mcp-universal \
  --transport streamable-http \
  --http-addr localhost:8080 \
  --mcp-token dev-token
```

Ожидаемый результат:

- dashboard доступен на `http://localhost:8080/dashboard`
- MCP endpoint доступен на `http://localhost:8080/mcp`
- health check доступен на `http://localhost:8080/health`

### Шаг 3. Добавление SAP-системы

Откройте `http://localhost:8080/dashboard` и заполните:

- `Имя соединения`
- `Имя системы`
- `URL сервиса`
- `SAP клиент`, если нужен
- `Логин`
- `Пароль`
- переключатель `Разрешить запись`, если хотите write-capable режим

Нажмите `Подключить`.

Ожидаемый результат:

- профиль сохраняется
- bridge загружает metadata
- MCP tools перестраиваются под выбранную SAP-систему

### Шаг 4. Подключение Codex

```bash
export SAP_ODATA_MCP_TOKEN=dev-token

codex mcp add sap-odata-universal \
  --url http://localhost:8080/mcp \
  --bearer-token-env-var SAP_ODATA_MCP_TOKEN
```

### Шаг 5. Smoke-check

Проверки:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/status
curl http://localhost:8080/api/databases
```

Ожидаемо:

- `/health` отвечает `ok`
- `/api/status` показывает активное подключение
- `/api/databases` возвращает список сохранённых профилей

---

## Сценарий 2. Один сервис через stdio

Если dashboard не нужен, используйте прямой запуск:

```bash
./sap-odata-mcp-universal \
  --service https://host/sap/opu/odata/sap/API_SALES_ORDER_SRV/ \
  --user YOUR_LOGIN \
  --password YOUR_PASSWORD
```

Пример подключения в Codex:

```bash
codex mcp add sap-odata-universal \
  -- /absolute/path/to/sap-odata-mcp-universal \
  --service https://host/sap/opu/odata/sap/API_SALES_ORDER_SRV/ \
  --user YOUR_LOGIN \
  --password YOUR_PASSWORD
```

---

## Проверка корректности bridge

### Проверка metadata

Если при подключении bridge не может построить tools, проверьте:

- корректность корневого OData URL
- нужны ли `sap-client`
- проходят ли логин/пароль
- действительно ли сервис отдаёт `$metadata`

### Проверка режима доступа

Если нет write-tools:

- dashboard-профиль может быть в `restricted`
- сервис может запрещать create/update/delete в metadata
- bridge может быть запущен с `--read-only`

### Проверка HTTP security

Если сервер не стартует:

- на `localhost` проверьте наличие `--mcp-token`
- вне `localhost` проверьте наличие `--mcp-token` и `--tls`
- на `0.0.0.0` проверьте `--allow-all-interfaces`

---

## Полезные режимы диагностики

### Вывести список tools и завершиться

```bash
./sap-odata-mcp-universal \
  --trace \
  --service https://host/sap/opu/odata/sap/API_SALES_ORDER_SRV/
```

### Включить трассировку MCP

```bash
./sap-odata-mcp-universal \
  --trace-mcp \
  --service https://host/sap/opu/odata/sap/API_SALES_ORDER_SRV/
```

### Подробные ошибки

```bash
./sap-odata-mcp-universal \
  --verbose \
  --verbose-errors \
  --service https://host/sap/opu/odata/sap/API_SALES_ORDER_SRV/
```

---

## Production notes

Для реального сетевого доступа:

- не оставляйте `dev-token`
- используйте `--mcp-token-file`
- публикуйте HTTP endpoint только с TLS
- ограничивайте права на state file
- используйте `restricted` по умолчанию для production landscape

---

## Связанные документы

- [README.md](../README.md)
- [004-security-analysis-http-transport.md](004-security-analysis-http-transport.md)
- [006-testing-strategy.md](006-testing-strategy.md)
