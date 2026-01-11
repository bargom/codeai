# Task: Email Notifications

## Overview
Implement email notification sending with template support and multiple provider backends.

## Phase
Phase 3: Workflows and Jobs

## Priority
Medium - Important for user notifications.

## Dependencies
- Phase 1 complete

## Description
Create an email notification module supporting templated emails, multiple SMTP/API providers, and async delivery.

## Detailed Requirements

### 1. Email Types (internal/modules/email/types.go)

```go
package email

type Email struct {
    To          []string
    CC          []string
    BCC         []string
    From        string
    ReplyTo     string
    Subject     string
    TextBody    string
    HTMLBody    string
    Attachments []Attachment
    Headers     map[string]string
}

type Attachment struct {
    Filename    string
    ContentType string
    Data        []byte
    Inline      bool
    ContentID   string
}

type Template struct {
    ID       string
    Name     string
    Subject  string
    TextBody string
    HTMLBody string
}

type EmailConfig struct {
    Provider    string // "smtp", "sendgrid", "ses", "mailgun"
    From        string
    FromName    string

    // SMTP settings
    SMTPHost     string
    SMTPPort     int
    SMTPUsername string
    SMTPPassword string
    SMTPTLS      bool

    // API providers
    APIKey   string
    Domain   string
    Region   string
}
```

### 2. Email Module Interface (internal/modules/email/module.go)

```go
package email

import (
    "context"
)

type EmailModule interface {
    Module
    Send(ctx context.Context, email *Email) error
    SendTemplate(ctx context.Context, templateID string, to []string, data map[string]any) error
    RegisterTemplate(template *Template) error
}

type emailModule struct {
    provider  EmailProvider
    templates map[string]*Template
    config    EmailConfig
    queue     chan *Email
    logger    *slog.Logger
}

func NewEmailModule(config EmailConfig) (EmailModule, error) {
    var provider EmailProvider
    var err error

    switch config.Provider {
    case "smtp":
        provider, err = NewSMTPProvider(config)
    case "sendgrid":
        provider = NewSendGridProvider(config)
    case "ses":
        provider = NewSESProvider(config)
    default:
        provider, err = NewSMTPProvider(config)
    }

    if err != nil {
        return nil, err
    }

    m := &emailModule{
        provider:  provider,
        templates: make(map[string]*Template),
        config:    config,
        queue:     make(chan *Email, 100),
        logger:    slog.Default().With("module", "email"),
    }

    return m, nil
}

func (m *emailModule) Name() string { return "email" }

func (m *emailModule) Initialize(config *Config) error {
    return nil
}

func (m *emailModule) Start(ctx context.Context) error {
    // Start async sender
    go m.sender(ctx)
    return nil
}

func (m *emailModule) Stop(ctx context.Context) error {
    close(m.queue)
    return nil
}

func (m *emailModule) Health() HealthStatus {
    return HealthStatus{Status: "healthy"}
}

func (m *emailModule) Send(ctx context.Context, email *Email) error {
    if email.From == "" {
        email.From = m.formatFrom()
    }

    return m.provider.Send(ctx, email)
}

func (m *emailModule) SendTemplate(ctx context.Context, templateID string, to []string, data map[string]any) error {
    template, ok := m.templates[templateID]
    if !ok {
        return fmt.Errorf("template not found: %s", templateID)
    }

    email := &Email{
        To:      to,
        From:    m.formatFrom(),
        Subject: m.renderTemplate(template.Subject, data),
    }

    if template.TextBody != "" {
        email.TextBody = m.renderTemplate(template.TextBody, data)
    }

    if template.HTMLBody != "" {
        email.HTMLBody = m.renderTemplate(template.HTMLBody, data)
    }

    return m.Send(ctx, email)
}

func (m *emailModule) RegisterTemplate(template *Template) error {
    m.templates[template.ID] = template
    return nil
}

func (m *emailModule) formatFrom() string {
    if m.config.FromName != "" {
        return fmt.Sprintf("%s <%s>", m.config.FromName, m.config.From)
    }
    return m.config.From
}

func (m *emailModule) renderTemplate(tmpl string, data map[string]any) string {
    t, err := template.New("").Parse(tmpl)
    if err != nil {
        return tmpl
    }

    var buf bytes.Buffer
    if err := t.Execute(&buf, data); err != nil {
        return tmpl
    }

    return buf.String()
}

func (m *emailModule) sender(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case email, ok := <-m.queue:
            if !ok {
                return
            }
            if err := m.provider.Send(ctx, email); err != nil {
                m.logger.Error("async email failed", "to", email.To, "error", err)
            }
        }
    }
}
```

### 3. SMTP Provider (internal/modules/email/smtp.go)

```go
package email

import (
    "context"
    "crypto/tls"
    "fmt"
    "net/smtp"
    "strings"
)

type SMTPProvider struct {
    host     string
    port     int
    username string
    password string
    useTLS   bool
}

func NewSMTPProvider(config EmailConfig) (*SMTPProvider, error) {
    return &SMTPProvider{
        host:     config.SMTPHost,
        port:     config.SMTPPort,
        username: config.SMTPUsername,
        password: config.SMTPPassword,
        useTLS:   config.SMTPTLS,
    }, nil
}

func (p *SMTPProvider) Send(ctx context.Context, email *Email) error {
    addr := fmt.Sprintf("%s:%d", p.host, p.port)

    // Build message
    msg := p.buildMessage(email)

    // Connect
    var client *smtp.Client
    var err error

    if p.useTLS {
        tlsConfig := &tls.Config{ServerName: p.host}
        conn, err := tls.Dial("tcp", addr, tlsConfig)
        if err != nil {
            return fmt.Errorf("TLS dial error: %w", err)
        }
        client, err = smtp.NewClient(conn, p.host)
        if err != nil {
            return fmt.Errorf("SMTP client error: %w", err)
        }
    } else {
        client, err = smtp.Dial(addr)
        if err != nil {
            return fmt.Errorf("dial error: %w", err)
        }

        // Try STARTTLS if available
        if ok, _ := client.Extension("STARTTLS"); ok {
            config := &tls.Config{ServerName: p.host}
            if err = client.StartTLS(config); err != nil {
                return fmt.Errorf("STARTTLS error: %w", err)
            }
        }
    }
    defer client.Close()

    // Authenticate
    if p.username != "" {
        auth := smtp.PlainAuth("", p.username, p.password, p.host)
        if err = client.Auth(auth); err != nil {
            return fmt.Errorf("auth error: %w", err)
        }
    }

    // Set sender
    if err = client.Mail(extractEmail(email.From)); err != nil {
        return fmt.Errorf("mail error: %w", err)
    }

    // Set recipients
    recipients := append(append(email.To, email.CC...), email.BCC...)
    for _, rcpt := range recipients {
        if err = client.Rcpt(rcpt); err != nil {
            return fmt.Errorf("rcpt error: %w", err)
        }
    }

    // Send message
    w, err := client.Data()
    if err != nil {
        return fmt.Errorf("data error: %w", err)
    }

    _, err = w.Write(msg)
    if err != nil {
        return fmt.Errorf("write error: %w", err)
    }

    err = w.Close()
    if err != nil {
        return fmt.Errorf("close error: %w", err)
    }

    return client.Quit()
}

func (p *SMTPProvider) buildMessage(email *Email) []byte {
    var msg strings.Builder

    msg.WriteString(fmt.Sprintf("From: %s\r\n", email.From))
    msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(email.To, ", ")))

    if len(email.CC) > 0 {
        msg.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(email.CC, ", ")))
    }

    msg.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))

    if email.ReplyTo != "" {
        msg.WriteString(fmt.Sprintf("Reply-To: %s\r\n", email.ReplyTo))
    }

    // Custom headers
    for k, v := range email.Headers {
        msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
    }

    if email.HTMLBody != "" {
        msg.WriteString("MIME-Version: 1.0\r\n")
        msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
        msg.WriteString("\r\n")
        msg.WriteString(email.HTMLBody)
    } else {
        msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
        msg.WriteString("\r\n")
        msg.WriteString(email.TextBody)
    }

    return []byte(msg.String())
}

func extractEmail(addr string) string {
    if idx := strings.Index(addr, "<"); idx >= 0 {
        end := strings.Index(addr, ">")
        if end > idx {
            return addr[idx+1 : end]
        }
    }
    return addr
}
```

### 4. Email Provider Interface

```go
// internal/modules/email/provider.go
package email

import "context"

type EmailProvider interface {
    Send(ctx context.Context, email *Email) error
}
```

### 5. SendGrid Provider (stub)

```go
// internal/modules/email/sendgrid.go
package email

type SendGridProvider struct {
    apiKey string
}

func NewSendGridProvider(config EmailConfig) *SendGridProvider {
    return &SendGridProvider{apiKey: config.APIKey}
}

func (p *SendGridProvider) Send(ctx context.Context, email *Email) error {
    // Implementation using SendGrid API
    return nil
}
```

## Acceptance Criteria
- [ ] SMTP email sending
- [ ] TLS/STARTTLS support
- [ ] Template-based emails
- [ ] Multiple recipient support (To, CC, BCC)
- [ ] Attachments support
- [ ] Provider abstraction
- [ ] Async email queue

## Testing Strategy
- Unit tests with mock SMTP server
- Integration tests with MailHog
- Template rendering tests

## Files to Create
- `internal/modules/email/types.go`
- `internal/modules/email/module.go`
- `internal/modules/email/provider.go`
- `internal/modules/email/smtp.go`
- `internal/modules/email/sendgrid.go`
- `internal/modules/email/email_test.go`
