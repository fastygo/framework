# 10. Ошибки и отладка

[← Интеграция webmail](09-integraciya-webmail.md) · [Оглавление](INDEX.md)

---

## Тип mail.Error

```go
type Error struct {
    Op   string      // "jmap: Email/query"
    Code ErrorCode
    Err  error       // unwrap chain
}
```

### ErrorCode — когда что делать

| Code | Смысл | Действие UI |
|------|--------|-------------|
| `auth` | Неверный пароль / forbidden | Login again |
| `not_found` | Письмо/папка не найдены | 404 страница |
| `too_large` | Лимит upload/message | Сообщение пользователю |
| `rate_limit` | 429 / server limit | Retry backoff |
| `unavailable` | Сеть, 5xx | Retry, offline banner |
| `protocol` | Баг/несовместимость | Log + support |
| `unsupported` | Нет capability | Disable feature |

### Branching

```go
if err != nil {
    switch mail.CodeOf(err) {
    case mail.CodeAuth:
        redirectLogin(w, r)
    case mail.CodeNotFound:
        http.NotFound(w, r)
    case mail.CodeUnavailable:
        renderRetryPage(w, err)
    default:
        http.Error(w, "mail error", http.StatusBadGateway)
    }
}
```

### Unwrap

```go
var me *mail.Error
if errors.As(err, &me) {
    log.Error("mail", "op", me.Op, "code", me.Code, "err", me.Err)
}
```

---

## Типичные ошибки при подключении

| Симптом | Причина | Решение |
|---------|---------|---------|
| `SessionURL is required` | Пустой URL | Discover или config |
| `Auth is required` | Nil Authenticator | BasicAuth / TokenAuth |
| `server lacks urn:...mail` | Не JMAP mail server | Проверить Stalwart JMAP enabled |
| `no primary mail account` | Session без mail account | AccountID явно |
| `status 401` | Неверные credentials | App password |
| `connection refused` | Сервер down / wrong port | Docker, firewall |
| TLS handshake error | Self-signed | InsecureTLS dev only |

---

## JMAP method errors

Method-level `"error"` в batch мапится на `mail.Error`:

| JMAP type | Code |
|-----------|------|
| `forbidden`, `accountReadOnly` | auth |
| `notFound` | not_found |
| `overQuota`, `tooLarge` | too_large |
| `rateLimit` | rate_limit |
| `serverUnavailable` | unavailable |

---

## Отладка HTTP

Временно логируйте **без секретов**:

```go
// custom Transport wrapper — log URL, status, duration; NOT Authorization header
```

Или используйте `httputil.DumpResponse` в dev на fake server tests.

---

## Тесты в framework

```bash
cd @Framework
go test ./pkg/mail/...
go test ./pkg/mail/jmap/... -v
```

- **Fake server:** `harness_test.go` — httptest без сети.
- **Audit tests:** `jmap_audit_test.go` — subject не в логах.
- **Leak tests:** goleak в `leak_test.go`.

---

## Smoke test со Stalwart (manual)

```bash
docker run -d --name stalwart -p 8080:8080 stalwartlabs/stalwart:latest
# создать user + password в admin UI
```

```go
// +build stalwart_integration

func TestStalwartLive(t *testing.T) {
    client, err := jmap.New(context.Background(), jmap.Options{
        SessionURL: "https://127.0.0.1:8080/.well-known/jmap",
        Auth: mail.BasicAuth{Username: "...", Password: "..."},
        InsecureTLS: true,
    })
    // Mailboxes, Messages, ...
}
```

Не включать в CI по умолчанию — только локально.

---

## FAQ

**Q: Можно ли один Client на всех пользователей?**  
A: Нет. У каждого аккаунта свои credentials и session.

**Q: Почему Attachment делает лишний Message()?**  
A: Проверка, что partID принадлежит письму — защита от угадывания blobId.

**Q: Delete vs Move в Trash?**  
A: UX «удалить» = Move(Trash). Delete = empty trash.

**Q: Когда IMAP?**  
A: ADR 0005, тот же `mail.Client`. Следите за changelog framework.

**Q: Discover не работает с http://localhost**  
A: By design — передайте SessionURL явно или HTTPS+mkcert.

---

## Полезные команды

```bash
# все тесты mail
go test ./pkg/mail/...

# vet + no-root-imports (как CI)
go vet ./pkg/mail/...
go run ./scripts/check-no-root-imports.go

# godoc локально
go doc github.com/fastygo/framework/pkg/mail Client
go doc github.com/fastygo/framework/pkg/mail/jmap New
```

---

## Карта «проблема → глава»

| Проблема | Глава |
|----------|--------|
| Не понимаю архитектуру | [2](02-arhitektura.md) |
| Что такое MessageSummary | [3](03-domennye-tipy.md) |
| Как Mark read | [4](04-kontrakt-client.md), [7](07-osnovnye-operacii.md) |
| Stalwart local | [6](06-jmap-podklyuchenie.md) |
| Search / Thread / Push | [8](08-rasshirennye-vozmozhnosti.md) |
| Templ + session | [9](09-integraciya-webmail.md) |
| CodeAuth | [10](10-oshibki-otladka.md) |

---

[← Интеграция webmail](09-integraciya-webmail.md) · [Оглавление](INDEX.md)
