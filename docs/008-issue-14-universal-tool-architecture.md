# Universal Tool Architecture в sap-odata-mcp-universal

**Документ:** 008  
**Дата:** 2026-04-23  
**Статус:** актуально

---

## Проблема, которую решает `--universal`

У крупных SAP OData сервисов MCP toolset может разрастаться до сотен инструментов:

- по несколько операций на каждый `EntitySet`
- отдельные function import tools
- service info tools

Для AI-клиента это создаёт проблемы:

- слишком большой tool schema
- рост token consumption
- ухудшение выбора инструмента
- случаи, когда MCP-клиент перестаёт нормально видеть API

---

## Обычный режим

В стандартном режиме bridge генерирует N инструментов:

- `search_*`
- `filter_*`
- `get_*`
- `create_*`
- `upd_*`
- `del_*`
- function import tools

Это хороший режим, когда:

- сервис небольшой или средний
- важна discoverability
- AI должен видеть явную структуру OData модели

---

## Universal mode

При `--universal` bridge регистрирует один универсальный OData tool вместо большого набора per-entity tools.

Идея:

- metadata по-прежнему загружается
- но наружу публикуется один инструмент
- конкретная операция маршрутизируется внутренне по параметрам вызова

---

## Когда использовать

`--universal` стоит включать, если:

- сервис очень большой
- подключено несколько SAP-систем через один gateway
- MCP-клиент деградирует из-за числа tools
- нужно уменьшить token footprint

---

## Компромиссы

Плюсы:

- резкое уменьшение числа инструментов
- меньше токенов в schema
- лучше масштабируется на крупных SAP APIs

Минусы:

- ниже discoverability
- пользователю и AI нужно точнее формировать параметры вызова
- отладка вызова может быть менее наглядной, чем в per-entity режиме

---

## Практический выбор режима

Используйте обычный режим, если:

- вы работаете с одной системой
- сервис относительно компактный
- вам нужен максимальный список явных tools

Используйте `--universal`, если:

- сервис крупный
- bridge работает как HTTP gateway для нескольких landscape profiles
- MCP-клиенту тяжело переваривать сотни инструментов

---

## Примеры

Обычный режим:

```bash
./sap-odata-mcp-universal \
  --transport streamable-http \
  --http-addr localhost:8080 \
  --mcp-token dev-token
```

Universal mode:

```bash
./sap-odata-mcp-universal \
  --transport streamable-http \
  --http-addr localhost:8080 \
  --mcp-token dev-token \
  --universal
```

---

## Связь с dashboard

Dashboard и `--universal` хорошо сочетаются:

- один сервер
- несколько SAP-систем
- одна активная система в каждый момент времени
- минимальный MCP toolset для текущего профиля

Это особенно полезно, если сервер используется как единый локальный SAP OData gateway для Codex или другого HTTP MCP-клиента.

---

## Связанные документы

- [README.md](../README.md)
- [002-odata-mcp-september-2025-update.md](002-odata-mcp-september-2025-update.md)
