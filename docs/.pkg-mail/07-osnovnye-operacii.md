# 7. Основные операции

[← JMAP подключение](06-jmap-podklyuchenie.md) · [Оглавление](INDEX.md) · [Далее: Расширенные возможности →](08-rasshirennye-vozmozhnosti.md)

---

## Сценарий: Inbox end-to-end

```go
boxes, err := client.Mailboxes(ctx)
inbox := mail.FindByRole(boxes, mail.RoleInbox)
if inbox == nil {
    return fmt.Errorf("no inbox")
}

page, err := client.Messages(ctx, inbox.ID, mail.ListOptions{
    Limit:      50,
    UnreadOnly: false,
    SortBy:     mail.SortDate,
    Ascending:  false,
})

for _, summary := range page.Items {
    fmt.Println(summary.Envelope.Subject, summary.Flags.Has(mail.FlagSeen))
}
```

---

## Чтение полного письма

```go
msg, err := client.Message(ctx, summary.ID)
if err != nil {
    return err
}

plain := msg.TextBody
html := msg.HTMLBody  // sanitize before render!

// Автоматически «прочитано» (опционально):
_ = client.SetFlags(ctx, []string{msg.ID}, mail.NewFlagSet(mail.FlagSeen), nil)
```

---

## Скачивание вложения

```go
msg, _ := client.Message(ctx, messageID)

for _, att := range msg.Attachments {
    rc, info, err := client.Attachment(ctx, messageID, att.PartID)
    if err != nil {
        continue
    }
    // io.Copy(file, rc)
    rc.Close()
    _ = info
}
```

HTTP handler pattern:

```go
func downloadAttachment(w http.ResponseWriter, r *http.Request, client mail.Client, msgID, partID string) {
    rc, info, err := client.Attachment(r.Context(), msgID, partID)
    if err != nil {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }
    defer rc.Close()

    w.Header().Set("Content-Type", info.ContentType)
    w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, info.Filename))
    io.Copy(w, rc)
}
```

---

## Флаги: прочитано, звёздочка, ответ

```go
// Mark read
client.SetFlags(ctx, ids, mail.NewFlagSet(mail.FlagSeen), nil)

// Toggle star (упрощённо — только add)
client.SetFlags(ctx, ids, mail.NewFlagSet(mail.FlagFlagged), nil)

// Unstar
client.SetFlags(ctx, ids, nil, mail.NewFlagSet(mail.FlagFlagged))
```

---

## Перемещение и корзина

```go
boxes, _ := client.Mailboxes(ctx)
trash := mail.FindByRole(boxes, mail.RoleTrash)

err := client.Move(ctx, []string{msgID}, trash.ID)
```

Архивирование:

```go
archive := mail.FindByRole(boxes, mail.RoleArchive)
client.Move(ctx, ids, archive.ID)
```

---

## Безвозвратное удаление

```go
// Только после «очистить корзину» UX:
client.Delete(ctx, []string{msgID})
```

---

## Отправка

```go
result, err := client.Send(ctx, mail.Draft{
    From: mail.Address{
        Name:  "Ada",
        Email: "ada@example.com",
    },
    To: []mail.Address{
        {Email: "bob@example.com"},
    },
    Subject:  "Hello",
    TextBody: "Plain text body",
    HTMLBody: "<p>HTML body</p>", // optional
})
```

### С вложением (stream)

```go
draft := mail.Draft{
    From:     mail.Address{Email: "ada@example.com"},
    To:       []mail.Address{{Email: "bob@example.com"}},
    Subject:  "Report",
    TextBody: "See attachment",
    Attachments: []mail.OutgoingAttachment{
        {
            Filename:    "report.pdf",
            ContentType: "application/pdf",
            Open: func() (io.ReadCloser, error) {
                return os.Open("/path/to/report.pdf")
            },
        },
    },
}
result, err := client.Send(ctx, draft)
```

`Open` может вызываться повторно при retry — каждый раз новый reader с начала.

### Reply (In-Reply-To)

```go
original, _ := client.Message(ctx, replyToID)

draft := mail.Draft{
    From:      mail.Address{Email: "ada@example.com"},
    To:        original.Envelope.From, // simplified
    Subject:   "Re: " + original.Envelope.Subject,
    TextBody:  quotedReply,
    InReplyTo: []string{original.Envelope.MessageID},
    References: append(original.Envelope.References, original.Envelope.MessageID),
}
client.Send(ctx, draft)
```

---

## Пагинация «Load more»

```go
func loadPage(client mail.Client, mailboxID string, offset int64) (mail.Page[mail.MessageSummary], error) {
    return client.Messages(ctx, mailboxID, mail.ListOptions{
        Offset: offset,
        Limit:  50,
    })
}
```

UI: храните `offset += len(page.Items)`; кнопка «ещё» пока `offset < page.Total` или пока `len(page.Items) == limit`.

---

## Несколько папок в sidebar

```go
boxes, _ := client.Mailboxes(ctx)

// Сортировка по SortOrder, затем Name
sort.Slice(boxes, func(i, j int) bool {
    if boxes[i].SortOrder != boxes[j].SortOrder {
        return boxes[i].SortOrder < boxes[j].SortOrder
    }
    return boxes[i].Name < boxes[j].Name
})

for _, mb := range boxes {
    // render: mb.Name, mb.UnreadMessages, icon by mb.Role
}
```

---

## Проверка лимитов перед upload

```go
caps := client.Capabilities()
if caps.MaxUploadSize > 0 && fileSize > caps.MaxUploadSize {
    return fmt.Errorf("file too large for server")
}
```

---

[← JMAP подключение](06-jmap-podklyuchenie.md) · [Оглавление](INDEX.md) · [Далее: Расширенные возможности →](08-rasshirennye-vozmozhnosti.md)
