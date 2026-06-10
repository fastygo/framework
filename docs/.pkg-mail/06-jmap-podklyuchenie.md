# 6. JMAP: подключение

[← Аутентификация](05-autentifikaciya-bezopasnost.md) · [Оглавление](INDEX.md) · [Далее: Основные операции →](07-osnovnye-operacii.md)

---

## Зависимости

```go
import (
    "github.com/fastygo/framework/pkg/mail"
    "github.com/fastygo/framework/pkg/mail/jmap"
)
```

Дополнительных `go get` **не нужно** — JMAP на stdlib.

---

## Шаг 1: URL сессии

### Вариант A — Discover (из hostname)

```go
sessionURL, err := jmap.Discover("mail.example.com")
// → "https://mail.example.com/.well-known/jmap"
```

Принимает:

- `mail.example.com`
- `mail.example.com:8443`
- `https://mail.example.com/anything` (path/query отбрасываются)

**Только HTTPS.** HTTP отклоняется.

**SRV autodiscovery нет** — host должен быть известен приложению (config, login form).

### Вариант B — явный URL

```go
sessionURL := "https://mail.example.com/.well-known/jmap"
```

Удобно для Stalwart за reverse proxy с нестандартным path.

---

## Шаг 2: jmap.Options

```go
type Options struct {
    SessionURL       string              // обязательно
    Auth             mail.Authenticator  // обязательно
    HTTPClient       *http.Client        // опционально
    AccountID        string              // опционально, default = primary mail
    MaxResponseBytes int64               // 0 → 16 MiB
    MaxBodyBytes     int64               // 0 → 1 MiB
    InsecureTLS      bool                // dev only
}
```

---

## Шаг 3: New

```go
ctx := context.Background() // или request context

client, err := jmap.New(ctx, jmap.Options{
    SessionURL: sessionURL,
    Auth: mail.BasicAuth{
        Username: "ada@example.com",
        Password: "secret",
    },
})
if err != nil {
    // см. главу 10 — CodeAuth, CodeUnsupported, ...
    return err
}
defer client.Close()
```

При успехе:

1. GET session resource;
2. Проверка `urn:ietf:params:jmap:core` и `urn:ietf:params:jmap:mail`;
3. Выбор `primaryAccounts[mail]` или `AccountID`;
4. Заполнение `Capabilities`.

---

## Stalwart — локальная разработка

### Docker

```bash
docker run -d --name stalwart \
  -p 8080:8080 -p 443:443 \
  -v stalwart-data:/opt/stalwart \
  stalwartlabs/stalwart:latest
```

1. Откройте web-admin (порт из документации Stalwart).
2. Создайте аккаунт `user@localhost.test` (или ваш домен).
3. Создайте app password / пароль.

### Подключение из Go

```go
sessionURL := "http://127.0.0.1:8080/.well-known/jmap" // если Stalwart на HTTP локально
```

**Проблема:** `Discover` требует HTTPS. Для localhost:

```go
// Явный SessionURL + InsecureTLS если TLS self-signed:
client, err := jmap.New(ctx, jmap.Options{
    SessionURL:  "https://127.0.0.1:8080/.well-known/jmap",
    Auth:        mail.BasicAuth{Username: "...", Password: "..."},
    InsecureTLS: true,
})
```

Или передайте `HTTPClient` с кастомным Transport для вашего dev-окружения.

### /etc/hosts (опционально)

```text
127.0.0.1  mail.test mx.mail.test
```

И `Discover("mail.test")` с mkcert-сертификатом.

---

## Кастомный HTTPClient

Для production за reverse proxy, tracing, connection pooling:

```go
httpClient := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConnsPerHost: 10,
        // TLSConfig: ...
    },
}

client, err := jmap.New(ctx, jmap.Options{
    SessionURL: sessionURL,
    Auth:       auth,
    HTTPClient: httpClient,
})
```

`InsecureTLS` **игнорируется**, если передан `HTTPClient` — настраивайте TLS в Transport.

---

## Проверка подключения (health probe)

Минимальный probe при login:

```go
func probeMail(ctx context.Context, sessionURL, user, pass string) error {
    c, err := jmap.New(ctx, jmap.Options{
        SessionURL: sessionURL,
        Auth:       mail.BasicAuth{Username: user, Password: pass},
    })
    if err != nil {
        return err
    }
    defer c.Close()
    _, err = c.Mailboxes(ctx)
    return err
}
```

---

## Session resource — что внутри (концептуально)

После `New` клиент хранит:

| Поле | Использование |
|------|----------------|
| `apiUrl` | POST все method calls |
| `downloadUrl` | Attachment stream |
| `uploadUrl` | Send attachments |
| `eventSourceUrl` | Watch push |
| `accounts` | AccountID |
| `capabilities` | лимиты upload |

Вам не нужно читать wire.Session — всё через методы Client.

---

## Мульти-аккаунт

Если в session несколько mail-accounts:

```go
client, err := jmap.New(ctx, jmap.Options{
    SessionURL: sessionURL,
    Auth:       auth,
    AccountID:  "specific-account-id-from-session",
})
```

Default: primary mail account из session.

---

## Graceful shutdown

```go
// при logout или shutdown app:
client.Close()
```

Для активного `Watch` — **отмените context**, переданный в `Watch`, затем `Close()`.

---

[← Аутентификация](05-autentifikaciya-bezopasnost.md) · [Оглавление](INDEX.md) · [Далее: Основные операции →](07-osnovnye-operacii.md)
