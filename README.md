# go-mailtester

![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)
[![Go Report Card](https://goreportcard.com/badge/github.com/btafoya/go-mailtester)](https://goreportcard.com/report/github.com/btafoya/go-mailtester)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A Go CLI tool for testing SMTP and IMAP servers. For SMTP, it exercises every layer: raw TCP, STARTTLS, implicit TLS, authentication, manual MAIL/RCPT/DATA pipelines, and high-level `SendMail` helpers. For IMAP, it tests connection, STARTTLS, implicit TLS, authentication, mailbox listing, status, and message fetching. Built on [`github.com/emersion/go-smtp`](https://github.com/emersion/go-smtp) and [`github.com/emersion/go-imap`](https://github.com/emersion/go-imap).

## Install

```bash
go install github.com/btafoya/go-mailtester@latest
```

Or clone and build:

```bash
git clone https://github.com/btafoya/go-mailtester.git
cd go-mailtester
go build -o mailtester .
```

## Quick Start

```bash
# Connectivity check
mailtester -host smtp.gmail.com -port 587 -mode connection

# Full SMTP test suite with STARTTLS + auth + send
mailtester -host smtp.gmail.com -port 587 -starttls \
  -from me@example.com -to you@example.net \
  -user me -pass secret -mode all

# Implicit TLS on port 465
mailtester -host smtp.gmail.com -port 465 -tls \
  -from me@example.com -to you@example.net \
  -user me -pass secret -mode send

# IMAP connectivity check
mailtester -host imap.gmail.com -port 993 -tls -mode imap-connection

# Full IMAP test suite
mailtester -host imap.gmail.com -port 993 -tls \
  -user me -pass secret -mode imap-all
```

## Test Modes

Use `-mode` to select which layer to exercise. Default is `all` (SMTP only). Use `imap-all` for IMAP.

| Mode | What it tests |
|------|---------------|
| `connection` | TCP connectivity, EHLO, and server extension discovery (STARTTLS, AUTH, SIZE, etc.) |
| `starttls` | Plaintext dial → check STARTTLS extension → upgrade to TLS → inspect TLS state |
| `ssl` | Implicit TLS (port 465): `tls.Dial` → inspect TLS state → EHLO → optional auth |
| `auth` | Dial with selected security, EHLO, then authenticate with PLAIN or LOGIN |
| `send` | Full manual pipeline: `MAIL FROM` → `RCPT TO` → `DATA` → message body → `QUIT` |
| `sendmail` | High-level `smtp.SendMail()` convenience function |
| `sendmailtls` | High-level `smtp.SendMailTLS()` convenience function (implicit TLS) |
| `raw` | Raw `net/textproto` SMTP session, bypassing the library entirely |
| `imap-connection` | Plaintext IMAP dial + greeting + capability listing |
| `imap-starttls` | IMAP STARTTLS upgrade + capability listing |
| `imap-ssl` | Implicit TLS IMAP (port 993) + greeting + capabilities + optional auth |
| `imap-auth` | IMAP connection + authenticate with PLAIN or LOGIN |
| `imap-list` | List all mailboxes |
| `imap-status` | Select a mailbox and report message count, UIDNext, UIDValidity |
| `imap-fetch` | Fetch the first message envelope and flags |
| `all` | Sequentially runs all SMTP tests |
| `imap-all` | Sequentially runs all IMAP tests |

## Flags

```
-host        SMTP/IMAP server hostname (default: localhost)
-port        SMTP/IMAP server port (default: 25)
-from        Sender email address
-to          Recipient(s), comma-separated
-subject     Email subject (default: "SMTP Test")
-body        Email body (default: "This is a test email from go-mailtester.")
-user        SMTP/IMAP username
-pass        SMTP/IMAP password
-auth        Auth mechanism: plain, login, none (default: plain)
-tls         Use implicit TLS (port 465 for SMTP, 993 for IMAP)
-starttls    Use STARTTLS
-skip-verify Skip TLS certificate verification
-timeout     Connection timeout (default: 30s)
-helo        EHLO/HELO hostname (default: go-mailtester)
-mode        Test mode (default: all)
-mailbox     IMAP mailbox to test (default: INBOX)
```

## Examples

**Test connectivity only:**
```bash
mailtester -host smtp.example.com -port 25 -mode connection
```

**Test STARTTLS negotiation:**
```bash
mailtester -host smtp.example.com -port 587 -mode starttls
```

**Test authentication:**
```bash
mailtester -host smtp.example.com -port 587 -starttls \
  -user alice -pass secret -mode auth
```

**Send a single test email via STARTTLS:**
```bash
mailtester -host smtp.example.com -port 587 -starttls \
  -from alice@example.com -to bob@example.net \
  -user alice -pass secret -mode send
```

**Send via implicit TLS (port 465):**
```bash
mailtester -host smtp.example.com -port 465 -tls \
  -from alice@example.com -to bob@example.net \
  -user alice -pass secret -mode send
```

**Use LOGIN auth instead of PLAIN:**
```bash
mailtester -host smtp.example.com -port 587 -starttls -auth login \
  -from alice@example.com -to bob@example.net \
  -user alice -pass secret -mode send
```

**Skip certificate verification (self-signed certs):**
```bash
mailtester -host mail.local -port 465 -tls -skip-verify \
  -from test@local -to admin@local -mode connection
```

**Custom timeout:**
```bash
mailtester -host slow.server.com -timeout 60s -mode connection
```

**Raw SMTP session (see server responses line-by-line):**
```bash
mailtester -host smtp.example.com -port 25 -mode raw
```

**Test IMAP connectivity:**
```bash
mailtester -host imap.example.com -port 993 -tls -mode imap-connection
```

**Test IMAP STARTTLS:**
```bash
mailtester -host imap.example.com -port 143 -mode imap-starttls
```

**Test IMAP authentication:**
```bash
mailtester -host imap.example.com -port 993 -tls \
  -user alice -pass secret -mode imap-auth
```

**List IMAP mailboxes:**
```bash
mailtester -host imap.example.com -port 993 -tls \
  -user alice -pass secret -mode imap-list
```

**Check IMAP mailbox status:**
```bash
mailtester -host imap.example.com -port 993 -tls \
  -user alice -pass secret -mode imap-status -mailbox INBOX
```

**Fetch first IMAP message:**
```bash
mailtester -host imap.example.com -port 993 -tls \
  -user alice -pass secret -mode imap-fetch -mailbox INBOX
```

**Full IMAP diagnostic suite:**
```bash
mailtester -host imap.example.com -port 993 -tls \
  -user alice -pass secret -mode imap-all
```

## Dependencies

- [`github.com/emersion/go-smtp`](https://github.com/emersion/go-smtp)
- [`github.com/emersion/go-imap`](https://github.com/emersion/go-imap)
- [`github.com/emersion/go-sasl`](https://github.com/emersion/go-sasl) (transitive)

## License

[MIT](LICENSE)
