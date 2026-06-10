# 1. Введение

[← Оглавление](INDEX.md) · [Далее: Архитектура →](02-arhitektura.md)

---

## Зачем нужен pkg/mail

`pkg/mail` — это **библиотечное ядро** для построения webmail-клиентов на FastyGo. Оно отвечает на один вопрос:

> Как из Go-кода **единым способом** работать с почтовым сервером — списком папок, письмами, вложениями, отправкой — не привязывая UI и HTTP-слой к конкретному протоколу?

Сегодня реализован транспорт **JMAP** (приоритет — **Stalwart**). Завтра за тем же интерфейсом `mail.Client` может стоять IMAP+SMTP (maddy, Dovecot) — без переписывания экранов inbox/compose.

---

## Философия (три столпа Framework)

Пакет следует правилам из `docs/ARCHITECTURE.md`:

1. **Без лишнего кода** — один контракт `Client`, опциональные возможности через отдельные интерфейсы (`Threader`, `Pusher`, `Searcher`).
2. **Без лишних запросов и утечек** — батчинг JMAP-вызовов, streaming вложений, goroutine push привязана к `context`, goleak в тестах.
3. **Без лишних зависимостей** — JMAP-клиент на **stdlib** (`net/http`, `encoding/json`); в `go.mod` framework **не добавляются** новые модули.

---

## Что пакет **делает**

| Область | Описание |
|---------|----------|
| Домен | Типы `Message`, `Mailbox`, `Draft`, флаги, пагинация |
| Контракт | `mail.Client` — единый API для любого транспорта |
| JMAP | `pkg/mail/jmap` — полная реализация для RFC 8620/8621 |
| Безопасность | Инъекция credentials, redaction в логах, bounded reads |
| Ошибки | Типизированные `mail.Error` с `ErrorCode` |

## Что пакет **не делает**

| Область | Почему снаружи |
|---------|----------------|
| Web-страницы, Templ | UI — задача приложения |
| Сессии пользователя webmail | `pkg/auth` + ваш `internal/` |
| Кэш списка писем на диске | Политика кэша — приложение |
| Админка почтового сервера | Это продукт/server-side, не клиент |
| CalDAV, контакты | Вне scope mail-клиента v1 |

---

## Два уровня API

```text
pkg/mail          ←  вы импортируете в 99% случаев (типы + Client)
    │
    └── pkg/mail/jmap   ←  конкретный транспорт (Stalwart, другой JMAP)
            │
            └── internal/wire   ←  JSON JMAP (не импортировать!)
```

**Правило:** код webmail зависит от `mail.Client`, а `jmap.New(...)` вызывается только в composition root (`main`, wire-up слой).

---

## JMAP в двух словах

**JMAP** (JSON Meta Application Protocol, RFC 8620) — современный API почты поверх HTTPS+JSON вместо классической пары IMAP+SMTP.

| Аспект | IMAP (классика) | JMAP |
|--------|-----------------|------|
| Транспорт | TCP, бинарные команды | HTTPS POST, JSON |
| Сессия | Долгое соединение | Stateless HTTP (+ EventSource для push) |
| Батчинг | Ограничен | Несколько методов в одном запросе |
| Stalwart | Поддерживает | **Сильная сторона** |

`pkg/mail/jmap` реализует подмножество, нужное webmail-клиенту: Mailbox, Email, Identity, EmailSubmission, Blob, Thread, EventSource.

---

## Минимальный сценарий использования

```text
1. Discover / SessionURL  →  URL сессии JMAP
2. jmap.New               →  mail.Client (+ Threader, Pusher, Searcher)
3. Mailboxes              →  найти RoleInbox
4. Messages               →  список писем
5. Message                →  тело + метаданные вложений
6. Attachment             →  stream файла
7. Send                   →  исходящее письмо
8. Close                  →  освободить idle-соединения HTTP
```

---

## Связь с будущим webmail-приложением

```text
┌─────────────────────────────────────────┐
│  AppMail (отдельный repo)               │
│  Framework host + Templ + pkg/auth      │
│  internal/delivery/http  → handlers     │
│  internal/views          → inbox UI     │
└──────────────────┬──────────────────────┘
                   │ mail.Client
┌──────────────────▼──────────────────────┐
│  github.com/fastygo/framework/pkg/mail  │
│  github.com/fastygo/framework/pkg/mail/jmap
└──────────────────┬──────────────────────┘
                   │ HTTPS
┌──────────────────▼──────────────────────┐
│  Stalwart / другой JMAP-сервер          │
└─────────────────────────────────────────┘
```

LilMail и другие reference-проекты **не копируются** — только идеи UX. Протокольный слой написан с нуля под ADR 0004.

---

## Когда читать ADR

Если нужны **обоснования решений** (почему свой JMAP, почему без IMAP в v1, threat model) — см. [docs/adr/0004-mail-package.md](../../docs/adr/0004-mail-package.md).

---

[← Оглавление](INDEX.md) · [Далее: Архитектура →](02-arhitektura.md)
