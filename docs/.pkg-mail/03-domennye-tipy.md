# 3. Доменные типы

[← Архитектура](02-arhitektura.md) · [Оглавление](INDEX.md) · [Далее: Контракт Client →](04-kontrakt-client.md)

---

## Address — адрес участника

```go
type Address struct {
    Name  string // "Ada Lovelace" — опционально
    Email string // "ada@example.com" — обязательно
}
```

`String()` форматирует как `Ada Lovelace <ada@example.com>` или просто email.

---

## Envelope — заголовки письма

Общая часть для списка и полного письма:

| Поле | Смысл |
|------|--------|
| `From`, `To`, `Cc`, `Bcc`, `ReplyTo` | `[]Address` |
| `Subject` | Тема (декодированная) |
| `Date` | `time.Time` — из SentAt или ReceivedAt |
| `MessageID` | RFC 5322 Message-ID **без** угловых скобок |
| `InReplyTo`, `References` | Для цепочек / reply |

---

## MessageSummary — строка в списке

Лёгкая форма для inbox-таблицы:

```go
type MessageSummary struct {
    ID             string
    ThreadID       string      // ID беседы (JMAP Thread)
    MailboxIDs     []string    // письмо может быть в нескольких папках (JMAP)
    Envelope       Envelope
    Preview        string      // короткий текст для списка
    Flags          FlagSet
    Size           int64
    HasAttachments bool
}
```

---

## Message — полное письмо

```go
type Message struct {
    MessageSummary
    TextBody     string
    HTMLBody     string              // ⚠ untrusted — см. гл. 5
    Attachments  []AttachmentInfo    // метаданные, не содержимое
}
```

---

## AttachmentInfo и скачивание

```go
type AttachmentInfo struct {
    PartID      string // для Client.Attachment — у JMAP это blobId
    Filename    string
    ContentType string
    Size        int64
    Inline      bool   // Content-Disposition: inline + CID
}
```

Содержимое **никогда** не загружается в `Message` целиком — только через:

```go
rc, info, err := client.Attachment(ctx, messageID, partID)
defer rc.Close()
// io.Copy(dst, rc)
```

---

## Mailbox и Role

```go
type Mailbox struct {
    ID             string
    ParentID       string
    Name           string
    Role           Role
    TotalMessages  int64  // -1 если сервер не сообщил
    UnreadMessages int64
    SortOrder      int
}
```

### Специальные роли (Role)

| Константа | JMAP role | Назначение |
|-----------|-----------|------------|
| `RoleInbox` | inbox | Входящие |
| `RoleSent` | sent | Отправленные |
| `RoleDrafts` | drafts | Черновики |
| `RoleTrash` | trash | Корзина |
| `RoleArchive` | archive | Архив |
| `RoleJunk` | junk | Спам |
| `RoleNone` | — | Обычная пользовательская папка |

**Не ищите папку по локализованному имени** («Входящие», «Korb»). Используйте:

```go
inbox := mail.FindByRole(boxes, mail.RoleInbox)
trash := mail.FindByRole(boxes, mail.RoleTrash)
```

---

## FlagSet — ключевые слова

JMAP и будущий IMAP используют **единые** имена:

| Константа | Значение | Смысл |
|-----------|----------|--------|
| `FlagSeen` | `$seen` | Прочитано |
| `FlagFlagged` | `$flagged` | Помечено / star |
| `FlagAnswered` | `$answered` | Отвечено |
| `FlagForwarded` | `$forwarded` | Переслано |
| `FlagDraft` | `$draft` | Черновик |

```go
flags := mail.NewFlagSet(mail.FlagSeen, mail.FlagFlagged)
if msg.Flags.Has(mail.FlagSeen) { ... }
names := msg.Flags.Names()
```

---

## Draft и OutgoingAttachment — отправка

```go
type Draft struct {
    From, To, Cc, Bcc
    Subject, TextBody, HTMLBody
    InReplyTo, References []string  // для Reply
    Attachments []OutgoingAttachment
}

type OutgoingAttachment struct {
    Filename, ContentType string
    Open func() (io.ReadCloser, error)  // stream при Send
}
```

**Требования валидации** (до сетевого вызова):

- `From.Email` не пустой
- хотя бы один получатель в To/Cc/Bcc
- хотя бы одно из TextBody / HTMLBody
- у каждого вложения: Filename, ContentType, Open

---

## SendResult

```go
type SendResult struct {
    MessageID     string // RFC Message-ID отправленного
    SentMessageID string // ID копии в Sent (если сервер сохранил)
}
```

---

## ListOptions — пагинация списка

```go
type ListOptions struct {
    Offset     int64      // с какой позиции (0-based)
    Limit      int        // 0 → default 50, max 500
    SortBy     SortField  // SortDate, SortSize, SortFrom, SortSubject
    Ascending  bool       // false = новые сверху (default)
    UnreadOnly bool       // только без $seen
}
```

---

## Page[T] — страница результатов

```go
type Page[T any] struct {
    Items  []T
    Total  int64  // -1 если неизвестно
    Offset int64
}
```

Пример обхода всех страниц (осторожно с большими ящиками):

```go
var offset int64
for {
    page, err := client.Messages(ctx, inboxID, mail.ListOptions{
        Offset: offset,
        Limit:  50,
    })
    // обработать page.Items...
    if len(page.Items) == 0 {
        break
    }
    offset += int64(len(page.Items))
    if page.Total >= 0 && offset >= page.Total {
        break
    }
}
```

---

## SearchQuery — для Searcher

```go
type SearchQuery struct {
    Text, From, To, Subject string
    MailboxID               string
    HasAttachment           bool
    After, Before           time.Time
}
```

Пустые поля игнорируются; непустые комбинируются через AND.

---

## Capabilities — что умеет сервер

```go
type Capabilities struct {
    Threads, Push, Search bool
    MaxUploadSize, MaxMessageSize int64
}
```

Заполняется при `jmap.New` из session resource. Статична на lifetime клиента.

---

[← Архитектура](02-arhitektura.md) · [Оглавление](INDEX.md) · [Далее: Контракт Client →](04-kontrakt-client.md)
