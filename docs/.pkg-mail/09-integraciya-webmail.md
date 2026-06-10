# 9. Интеграция webmail-приложения

[← Расширенные возможности](08-rasshirennye-vozmozhnosti.md) · [Оглавление](INDEX.md) · [Далее: Ошибки и отладка →](10-oshibki-otladka.md)

---

## Рекомендуемая структура repo (AppMail)

```text
AppMail/
  cmd/server/main.go
  internal/
    delivery/http/       # handlers: login, inbox, message, compose, attach
    application/mail/    # use-cases: InboxService, ComposeService
    session/             # MailSession in CookieSession
    views/               # Templ: inbox.templ, message.templ
    ui/layout/           # shell (Blank/BuildY pattern)
  go.mod                 # require github.com/fastygo/framework
```

**pkg/mail** импортируется в `application` и `delivery`, **не** в `.templ` файлах напрямую.

---

## Слои и зависимости

```text
views (Templ)
    ↑ props: готовые строки + structs, без mail.Client
delivery/http
    ↑ вызывает application, рендерит views
application/mail
    ↑ mail.Client interface
jmap.New (composition root / session factory)
```

---

## Framework host

По образцу BuildY / AppCMS:

```go
import frameworkapp "github.com/fastygo/framework/pkg/app"
import "github.com/fastygo/framework/pkg/web"

// Feature регистрирует /mail/* routes
type MailFeature struct { /* ... */ }

func (f *MailFeature) Routes(mux *http.ServeMux) {
    mux.HandleFunc("GET /mail/", f.inbox)
    mux.HandleFunc("GET /mail/message/{id}", f.message)
    mux.HandleFunc("POST /mail/send", f.send)
}

func main() {
    app := frameworkapp.New(cfg).
        WithFeature(NewMailFeature(...)).
        Build()
    app.Run(ctx)
}
```

Platform BFF **не обязателен** для v1 webmail — достаточно `net/http` + Templ.

---

## Сессия webmail vs Authenticator

```go
type MailSession struct {
    SessionURL string
    Username   string
    Encrypted  string // AES-GCM blob password or token ref
}

// При request:
func clientFromSession(sess MailSession, key []byte) (mail.Client, error) {
    pass, err := decrypt(sess.Encrypted, key)
    if err != nil {
        return nil, err
    }
    return jmap.New(ctx, jmap.Options{
        SessionURL: sess.SessionURL,
        Auth:       mail.BasicAuth{Username: sess.Username, Password: pass},
    })
}
```

**Альтернатива:** держать `*jmap.Client` в server-side session store (memory) с TTL — меньше reconnect, но больше state.

---

## Login flow

```text
POST /login
    form: server URL (or preset), email, password
    → jmap.New (probe)
    → Mailboxes (verify)
    → Issue CookieSession[MailSession]
    → redirect /mail/
```

При неверном пароле: `mail.CodeOf(err) == mail.CodeAuth` → форма с ошибкой.

---

## Inbox handler (sketch)

```go
func (h *Handler) inbox(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    client, err := h.client(r)
    if err != nil {
        http.Redirect(w, r, "/login", http.StatusFound)
        return
    }
    defer client.Close()

    boxes, _ := client.Mailboxes(ctx)
    inbox := mail.FindByRole(boxes, mail.RoleInbox)
    page, _ := client.Messages(ctx, inbox.ID, mail.ListOptions{Limit: 50})

    data := InboxViewData{ /* map summaries to display strings */ }
    web.Render(ctx, w, views.MailShell(layout, views.InboxPage(data)))
}
```

View model **не** содержит `mail.Client` — только DTO для Templ.

---

## Кэширование (на усмотрение приложения)

pkg/mail **не** кэширует. Варианты:

| Стратегия | Когда |
|-----------|--------|
| No cache | MVP, всегда live |
| TTL cache list | `framework/pkg/cache` in-memory |
| ETag / If-None-Match | На HTTP layer для static-like fragments |
| Push invalidation | Watch → сброс cache inbox |

Не кэшируйте `HTMLBody` долго без invalidation — риск stale и XSS surface.

---

## Compose + CSRF

- Form POST с CSRF token (Framework security middleware).
- Validate Draft на server перед `Send`.
- Rate limit на `/mail/send` (security config).

---

## Мульти-сервер (Stalwart + будущий IMAP)

```go
func dial(profile ServerProfile, auth mail.Authenticator) (mail.Client, error) {
    switch profile.Transport {
    case "jmap":
        return jmap.New(ctx, jmap.Options{SessionURL: profile.URL, Auth: auth})
    // case "imap":
    //     return imap.New(...)
    default:
        return nil, fmt.Errorf("unknown transport")
    }
}
```

UI login: выбор preset «Stalwart (JMAP)» или custom URL.

---

## EN/RU и fixtures

Для BuildY-style mockup: fixtures в JSON, views props-only.  
Для live AppMail: i18n через `pkg/web/i18n` или fixtures site/*.json.

pkg/mail **не** локализует имена папок — показывайте `Mailbox.Name` с сервера + icon по `Role`.

---

## Тестирование приложения

| Уровень | Как |
|---------|-----|
| Unit | Mock `mail.Client` (interface) |
| Integration | httptest fake JMAP (см. `pkg/mail/jmap/*_test.go`) |
| E2E | Stalwart Docker + build tag |

---

## Checklist перед production

- [ ] TLS verified (`InsecureTLS: false`)
- [ ] Cookie Secure + HttpOnly + SameSite
- [ ] HTML sanitize / iframe sandbox
- [ ] RedactEmail в логах приложения
- [ ] Body limits на upload handler
- [ ] `client.Close()` on logout
- [ ] Cancel Watch context on logout

---

[← Расширенные возможности](08-rasshirennye-vozmozhnosti.md) · [Оглавление](INDEX.md) · [Далее: Ошибки и отладка →](10-oshibki-otladka.md)
