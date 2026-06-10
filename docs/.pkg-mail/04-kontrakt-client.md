# 4. Контракт Client

[← Доменные типы](03-domennye-tipy.md) · [Оглавление](INDEX.md) · [Далее: Аутентификация →](05-autentifikaciya-bezopasnost.md)

---

## Интерфейс mail.Client — справочник методов

### Mailboxes

```go
Mailboxes(ctx context.Context) ([]Mailbox, error)
```

Возвращает **все** папки аккаунта. JMAP: `Mailbox/get` с `ids: null`.

**Типичное использование:** построить sidebar, найти Inbox/Sent/Trash через `FindByRole`.

---

### Messages

```go
Messages(ctx context.Context, mailboxID string, opts ListOptions) (Page[MessageSummary], error)
```

Страница писем **в одной папке**. Не включает тела — только summary + preview.

JMAP: batched `Email/query` + `Email/get` (summary properties).

---

### Message

```go
Message(ctx context.Context, id string) (*Message, error)
```

Полное письмо: текст, HTML, метаданные вложений.  
JMAP: `Email/get` с `fetchTextBodyValues`, `fetchHTMLBodyValues`, лимит `MaxBodyBytes` (default 1 MiB на тело).

`CodeNotFound` — если id не существует.

---

### Attachment

```go
Attachment(ctx context.Context, messageID, partID string) (io.ReadCloser, AttachmentInfo, error)
```

1. Сначала проверяет, что `partID` есть в `Message(messageID).Attachments` (защита от произвольного blobId).
2. Stream с download URL сессии.
3. **Вы обязаны** `Close()` reader.

---

### SetFlags

```go
SetFlags(ctx context.Context, ids []string, add, remove FlagSet) error
```

Добавляет ключевые слова из `add`, удаляет из `remove`.  
Пустой `ids` или оба пустых FlagSet — no-op без запроса.

Пример «прочитано»:

```go
client.SetFlags(ctx, []string{msgID}, mail.NewFlagSet(mail.FlagSeen), nil)
```

Пример «звёздочка»:

```go
client.SetFlags(ctx, []string{msgID}, mail.NewFlagSet(mail.FlagFlagged), nil)
```

---

### Move

```go
Move(ctx context.Context, ids []string, destMailboxID string) error
```

Перемещает в другую папку. JMAP: замена `mailboxIds`.

**«Удалить в корзину»** — это Move в `RoleTrash`, не Delete:

```go
trash := mail.FindByRole(boxes, mail.RoleTrash)
client.Move(ctx, []string{msgID}, trash.ID)
```

---

### Delete

```go
Delete(ctx context.Context, ids []string) error
```

**Безвозвратное** уничтожение (`Email/set destroy`). Для UX «корзина» используйте Move.

---

### Send

```go
Send(ctx context.Context, draft Draft) (*SendResult, error)
```

Отправка + (при поддержке сервера) сохранение копии в Sent. См. [главу 7](07-osnovnye-operacii.md#отправка).

---

### Capabilities / Close

```go
caps := client.Capabilities()
err := client.Close()
```

После `Close()` клиент использовать нельзя.

---

## Опциональные интерфейсы

### Threader

```go
Thread(ctx context.Context, threadID string) ([]MessageSummary, error)
```

Все письма беседы, **от старых к новым**.  
Берите `threadID` из `MessageSummary.ThreadID`.

### Pusher

```go
Watch(ctx context.Context) (<-chan ChangeEvent, error)
```

EventSource-поток изменений. Канал закрывается при отмене `ctx`.  
`ChangeEvent` **не содержит** содержимое писем — только сигнал «перечитайте inbox».

### Searcher

```go
Search(ctx context.Context, query SearchQuery, opts ListOptions) (Page[MessageSummary], error)
```

Полнотекстовый/фильтрованный поиск по аккаунту или папке.

---

## Type assertion — паттерн

```go
func enrichClient(c mail.Client) {
    if caps := c.Capabilities(); caps.Push {
        if p, ok := c.(mail.Pusher); ok {
            // подписаться на Watch
            _ = p
        }
    }
}
```

Для `jmap.Client` все три optional interface **всегда** доступны, но `Capabilities.Push` может быть `false`, если сервер не дал `eventSourceUrl`.

---

## Идентификаторы — правила

| ID | Правило |
|----|---------|
| `message id` | Opaque string от JMAP; храните как есть |
| `mailbox id` | Opaque string |
| `partID` для Attachment | BlobId из AttachmentInfo |
| `threadID` | Opaque string беседы |

**Не делайте:** парсинг, конкатенацию для UI, SQL LIKE по id.

---

## Конcurrency

`jmap.Client` безопасен для параллельных вызовов из разных HTTP-handlers.  
Исключение: не вызывайте `Close()` пока другие goroutine работают с клиентом.

Рекомендация для webmail: **один Client на залогиненную сессию** (или pool с mutex на уровне приложения).

---

[← Доменные типы](03-domennye-tipy.md) · [Оглавление](INDEX.md) · [Далее: Аутентификация →](05-autentifikaciya-bezopasnost.md)
