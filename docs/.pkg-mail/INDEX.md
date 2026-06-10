# pkg/mail — путеводитель 101

Исчерпывающий вводный курс по пакету `github.com/fastygo/framework/pkg/mail` и транспорту `pkg/mail/jmap`. Цель: после прочтения вы понимаете **зачем** пакет существует, **как устроен**, **как подключиться** к серверу (Stalwart / JMAP) и **как пользоваться** API в своём webmail-приложении.

> **Аудитория:** Go-разработчик, который строит webmail или почтовый модуль на FastyGo.  
> **Уровень:** 101 — без предположения, что вы уже знаете JMAP.  
> **Код:** актуален для текущей реализации в `@Framework/pkg/mail`.

---

## Быстрый старт (5 минут)

```go
import (
    "context"
    "log"

    "github.com/fastygo/framework/pkg/mail"
    "github.com/fastygo/framework/pkg/mail/jmap"
)

func main() {
    ctx := context.Background()

    sessionURL, _ := jmap.Discover("mail.example.com")

    client, err := jmap.New(ctx, jmap.Options{
        SessionURL: sessionURL,
        Auth: mail.BasicAuth{
            Username: "user@example.com",
            Password: "app-password",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    boxes, _ := client.Mailboxes(ctx)
    inbox := mail.FindByRole(boxes, mail.RoleInbox)

    page, _ := client.Messages(ctx, inbox.ID, mail.ListOptions{Limit: 20})
    for _, msg := range page.Items {
        log.Println(msg.Envelope.Subject)
    }
}
```

Дальше — по главам ниже.

---

## Карта документации

| № | Глава | О чём |
|---|--------|--------|
| 1 | [Введение](01-vvedenie.md) | Задача пакета, границы ответственности, что внутри / снаружи |
| 2 | [Архитектура](02-arhitektura.md) | Слои, контракт `Client`, транспорты, поток данных |
| 3 | [Доменные типы](03-domennye-tipy.md) | Address, Message, Mailbox, Draft, флаги, пагинация |
| 4 | [Контракт Client](04-kontrakt-client.md) | Все методы, опциональные интерфейсы, идентификаторы |
| 5 | [Аутентификация и безопасность](05-autentifikaciya-bezopasnost.md) | Authenticator, audit, redaction, HTML, лимиты |
| 6 | [JMAP: подключение](06-jmap-podklyuchenie.md) | Discover, Options, Stalwart, локальная разработка |
| 7 | [Основные операции](07-osnovnye-operacii.md) | Папки, список, чтение, вложения, флаги, перемещение, отправка |
| 8 | [Расширенные возможности](08-rasshirennye-vozmozhnosti.md) | Поиск, цепочки (Thread), push (EventSource) |
| 9 | [Интеграция webmail](09-integraciya-webmail.md) | Framework + Templ, сессии, кэш, типичный HTTP-слой |
| 10 | [Ошибки и отладка](10-oshibki-otladka.md) | ErrorCode, типичные сбои, тестирование, smoke с Docker |

---

## Связанные материалы в репозитории

| Документ | Назначение |
|----------|------------|
| [docs/adr/0004-mail-package.md](../../docs/adr/0004-mail-package.md) | ADR: решения, threat model, отложенный IMAP |
| [docs/API_REFERENCE.md](../../docs/API_REFERENCE.md) | Краткая справка по символам API |
| [docs/SECURITY.md](../../docs/SECURITY.md) | Безопасность framework, включая mail.audit |
| `pkg/mail/doc.go` | Godoc пакета (англ.) |
| `pkg/mail/jmap/example_test.go` | Пример подключения (компилируется, не запускается в CI) |

---

## Рекомендуемый порядок чтения

```text
Новичок в JMAP     → 1 → 2 → 6 → 7 → 4 → 5
Пишете webmail UI  → 1 → 3 → 4 → 9 → 7 → 8
DevOps / Stalwart  → 6 → 10 → 5
```

---

## Что **не** входит в pkg/mail

| Компонент | Где живёт |
|-----------|-----------|
| HTTP-маршруты, login-форма | Ваше приложение (отдельный repo webmail) |
| Хранение пароля в cookie/session | `pkg/auth` + слой приложения |
| Templ / UI inbox | `github.com/fastygo/templ` + `internal/views` |
| IMAP/SMTP (maddy, Dovecot) | Запланировано (ADR 0005), пока только JMAP |

---

*Последнее обновление гида: 2026-06-10 — синхронизировано с реализацией Phase 0–3 pkg/mail.*
