# Issue #12: исправление парсинга SAP OData metadata с несколькими Schema

**Документ:** 005  
**Дата:** 2026-04-23  
**Статус:** исправлено

---

## Проблема

Для части SAP OData сервисов bridge не генерировал tools по EntitySet и показывал только общий service-info инструмент.

Симптом:

- на Northwind всё работает
- на SAP сервисе tools почти отсутствуют

---

## Причина

Некоторые SAP OData metadata documents используют несколько `Schema`:

- в одной описаны `EntityType`
- в другой лежит `EntityContainer` и `EntitySet`

Старый парсер ожидал только одну `Schema`, из-за чего bridge видел лишь часть metadata.

Следствие:

- `EntitySet` не мог связаться с `EntityType`
- генерация entity tools пропускалась

---

## Что было исправлено

Исправление заключалось в переходе от single-schema parsing к обработке массива `Schema`.

После этого parser:

- собирает все схемы
- объединяет найденные `EntityType`
- корректно находит `EntityContainer`
- строит полноценную карту `EntitySet -> EntityType`

---

## Почему это особенно важно для SAP

Northwind и похожие демо-сервисы часто используют более простую структуру metadata.

SAP OData сервисы чаще используют:

- несколько schema blocks
- SAP-specific attributes
- дополнительные capability markers

Поэтому ошибка долго могла не проявляться на публичных тестовых сервисах, но воспроизводилась на реальных SAP landscapes.

---

## Результат после фикса

После исправления:

- SAP OData entity sets снова обнаруживаются корректно
- tools генерируются в обычном режиме
- read/write capability определяется по metadata, а не по случайному неполному набору schema blocks

---

## Практическая диагностика

Если похожий симптом возникнет снова, проверьте:

1. доступен ли `$metadata`
2. содержит ли документ несколько `Schema`
3. видит ли bridge `EntityContainer`
4. не пропал ли namespace mapping между `EntitySet` и `EntityType`

---

## Связанные файлы

- `internal/metadata/parser.go`
- `internal/bridge/bridge.go`

---

## Связанные документы

- [README.md](../README.md)
- [006-testing-strategy.md](006-testing-strategy.md)
