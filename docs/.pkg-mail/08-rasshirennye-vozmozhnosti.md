# 8. Расширенные возможности

[← Основные операции](07-osnovnye-operacii.md) · [Оглавление](INDEX.md) · [Далее: Интеграция webmail →](09-integraciya-webmail.md)

---

## Поиск (Searcher)

```go
searcher, ok := client.(mail.Searcher)
if !ok || !client.Capabilities().Search {
    // fallback: client-side filter по кэшу
    return
}

page, err := searcher.Search(ctx, mail.SearchQuery{
    Text:      "invoice",
    MailboxID: inboxID,       // пусто = весь аккаунт
    From:      "billing@",
    After:     time.Now().AddDate(0, -1, 0),
}, mail.ListOptions{Limit: 25})
```

### Поля SearchQuery

| Поле | JMAP filter |
|------|-------------|
| `Text` | full-text по subject/body/names |
| `From` | from header |
| `To` | to/cc |
| `Subject` | subject |
| `MailboxID` | inMailbox |
| `HasAttachment` | hasAttachment: true |
| `After` / `Before` | date range |

Несколько полей → AND.

---

## Цепочки писем (Threader)

```go
threader, ok := client.(mail.Threader)
if !ok {
    return
}

// threadID из MessageSummary при клике на conversation
msgs, err := threader.Thread(ctx, summary.ThreadID)
// msgs отсортированы от старых к новым
```

### UI-паттерн «conversation view»

```text
1. Messages(inbox) → группировка по ThreadID на клиенте (optional)
2. Клик → Thread(threadID) → render timeline
3. Reply → Draft с InReplyTo/References последнего в thread
```

JMAP нативно даёт `threadId` — JWZ-алгоритм на клиенте **не нужен** для JMAP-транспорта.

---

## Push-уведомления (Pusher / EventSource)

```go
if !client.Capabilities().Push {
    return // polling fallback
}

pusher, ok := client.(mail.Pusher)
if !ok {
    return
}

ctx, cancel := context.WithCancel(appCtx)
defer cancel()

events, err := pusher.Watch(ctx)
if err != nil {
    return err
}

for ev := range events {
    _ = ev // ChangeEvent — без содержимого
    // invalidate cache, SSE to browser, refresh inbox count
    refreshInbox(client)
}
```

### Важные детали реализации jmap.Watch

| Аспект | Поведение |
|--------|-----------|
| Goroutine | Одна на Watch; завершится при `ctx.Done()` |
| Reconnect | Автоматически через ~5s при обрыве |
| HTTP timeout | Stream использует client **без** global timeout |
| Cleanup | `cancel()` на context достаточно |

### Webmail + browser notifications

```text
jmap.Watch → server-side event
    → ваш SSE/WebSocket endpoint
    → browser Notification API (с permission)
```

`ChangeEvent` не говорит *какое* письмо пришло — сделайте лёгкий `Messages(inbox, Limit:1)` или сравните unread count.

---

## Capabilities — матрица UI

| Capability | UI feature |
|------------|------------|
| `Threads: true` | Conversation grouping |
| `Search: true` | Search box |
| `Push: true` | Live refresh / badge |
| `MaxUploadSize` | Max attach size hint |
| `MaxMessageSize` | Compose validation |

---

## Polling fallback (без Push)

```go
ticker := time.NewTicker(60 * time.Second)
defer ticker.Stop()

for {
    select {
    case <-ctx.Done():
        return
    case <-ticker.C:
        page, _ := client.Messages(ctx, inboxID, mail.ListOptions{Limit: 1})
        _ = page
    }
}
```

Используйте только если `!caps.Push` или EventSource блокируется proxy.

---

## Batch operations (несколько id)

Все mutation-методы принимают `[]string ids`:

```go
client.SetFlags(ctx, selectedIDs, mail.NewFlagSet(mail.FlagSeen), nil)
client.Move(ctx, selectedIDs, archiveID)
client.Delete(ctx, selectedIDs)
```

JMAP: один `Email/set` с несколькими ключами в `update`/`destroy`.

---

[← Основные операции](07-osnovnye-operacii.md) · [Оглавление](INDEX.md) · [Далее: Интеграция webmail →](09-integraciya-webmail.md)
