# 5. Аутентификация и безопасность

[← Контракт Client](04-kontrakt-client.md) · [Оглавление](INDEX.md) · [Далее: JMAP подключение →](06-jmap-podklyuchenie.md)

---

## Принцип: credentials injected, never stored

`pkg/mail` **никогда**:

- не хранит пароли и токены;
- не пишет их в лог;
- не кладёт в доменные типы.

Учётные данные передаются через **`mail.Authenticator`**, который вызывается **на каждый HTTP-запрос**.

---

## BasicAuth — Stalwart, app password

```go
auth := mail.BasicAuth{
    Username: "user@example.com",
    Password: "app-password-from-stalwart",
}
```

Подходит для:

- Stalwart с паролем приложения;
- локальной разработки поверх HTTPS.

Username обычно = полный email.

---

## TokenAuth — OAuth2 / OIDC

```go
auth := mail.TokenAuth{
    Token: func() (string, error) {
        // вернуть свежий access token
        // refresh — ответственность вызывающего (pkg/auth/oidc)
        return tokenStore.AccessToken(), nil
    },
}
```

`Token()` вызывается **перед каждым запросом** — удобно для expiring tokens.

Связка с Framework:

```text
pkg/auth/oidc  →  получить token при login
ваша session →  хранить encrypted refresh material
TokenAuth    →  mail/jmap client
```

---

## InsecureTLS — только dev

```go
jmap.New(ctx, jmap.Options{
    SessionURL:  url,
    Auth:        auth,
    InsecureTLS: true,  // self-signed localhost
})
```

- Работает только если **не** передан свой `HTTPClient`.
- Эмитит Warn `mail.audit` event: `insecure_tls_enabled`.
- В production **всегда** false + валидный TLS на reverse proxy.

---

## Bounded reads — защита от OOM

| Лимит | Default | Options |
|-------|---------|---------|
| JMAP API response | 16 MiB | `MaxResponseBytes` |
| Тело письма (bodyValues) | 1 MiB | `MaxBodyBytes` |
| EventSource line | 1 MiB | (константа в push.go) |

Вложения идут **stream** через download endpoint — не попадают в лимит API response целиком.

---

## HTMLBody — XSS-контракт

```go
msg.HTMLBody // string, untrusted
```

**pkg/mail не санитизирует HTML.** Контракт для приложения:

1. Рендерить в **sandboxed iframe** (`srcdoc` + CSP), или
2. Прогонять через HTML sanitizer **перед** вставкой в DOM, или
3. Показывать только `TextBody`, HTML — по кнопке «показать оригинал».

Никогда не использовать `template.HTML(msg.HTMLBody)` без очистки.

---

## Redaction — безопасные логи

```go
log.Info("mail", "from", mail.RedactEmail(addr.Email))
log.Info("mail", "subject", mail.RedactSubject(msg.Envelope.Subject))
// RedactEmail("ada@example.com") → "a***@example.com"
// RedactSubject("anything") → "(redacted)"
```

Используйте в **своём** HTTP-слое; pkg/mail/jmap уже redact'ит audit events.

---

## mail.audit — structured slog

События безопасности (уровень Info/Warn):

| event | Когда |
|-------|--------|
| `session_loaded` | Успешное подключение |
| `auth_failed` | 401/403 |
| `insecure_tls_enabled` | InsecureTLS: true |
| `message_sent` | Успешный Send (без subject/content) |
| `push_started` / `push_stopped` | EventSource lifecycle |

Фильтр в SIEM: `msg="mail.audit"` AND `level>=WARN`.

---

## Threat model (кратко)

| Угроза | Митигация |
|--------|-----------|
| Утечка пароля в лог | Authenticator + RedactEmail |
| Утечка содержимого писем | RedactSubject; audit без body |
| XSS через HTML mail | Контракт на приложение |
| Memory exhaustion | LimitReader на responses |
| MITM (dev misuse) | InsecureTLS explicit + audited |

Полная таблица: [ADR 0004](../../docs/adr/0004-mail-package.md).

---

## Хранение credentials в webmail-приложении

**Рекомендуемый паттерн** (не в pkg/mail, в AppMail):

```text
Login form
    → проверка jmap.New (probe)
    → AES-GCM encrypt(password) в CookieSession payload
    → при каждом request: decrypt → BasicAuth → jmap.Client
```

Или: хранить только session token OIDC + TokenAuth.

**Не делайте:** plaintext password в cookie, логирование Draft.Subject на Info.

---

[← Контракт Client](04-kontrakt-client.md) · [Оглавление](INDEX.md) · [Далее: JMAP подключение →](06-jmap-podklyuchenie.md)
