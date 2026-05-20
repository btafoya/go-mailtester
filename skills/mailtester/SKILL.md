# mailtester

AI agent skill for using the `go-mailtester` CLI SMTP testing tool.

## Purpose

This skill enables Claude Code to diagnose SMTP connectivity, authentication, and delivery issues using the `mailtester` binary built from this repository.

## Building

From the project root:

```bash
go build -o mailtester .
```

## Test Modes

| Mode | What it tests |
|------|---------------|
| `connection` | TCP connectivity + EHLO + extension listing |
| `starttls` | STARTTLS negotiation + TLS state inspection |
| `ssl` | Implicit TLS (port 465) connection + TLS state + EHLO + auth |
| `auth` | Authentication with configured credentials |
| `send` | Full manual MAIL FROM → RCPT TO → DATA → QUIT pipeline |
| `sendmail` | High-level `smtp.SendMail()` helper |
| `sendmailtls` | High-level `smtp.SendMailTLS()` helper (implicit TLS) |
| `raw` | Raw textproto SMTP session (no library abstractions) |
| `all` | Runs connection, starttls, ssl (if `-tls`), auth, send, sendmail, raw sequentially |

## Common Flags

```
-host        SMTP server hostname (default: localhost)
-port        SMTP server port (default: 25)
-from        Sender email address
-to          Recipient(s), comma-separated
-subject     Email subject
-body        Email body text
-user        SMTP username
-pass        SMTP password
-auth        Auth mechanism: plain, login, none (default: plain)
-tls         Use implicit TLS (port 465)
-starttls    Use STARTTLS
-skip-verify Skip TLS certificate verification
-timeout     Connection timeout (default: 30s)
-helo        EHLO/HELO hostname (default: go-mailtester)
-mode        Test mode (default: all)
```

## Typical Workflows

**Quick connectivity check:**
```bash
./mailtester -host smtp.example.com -port 587 -mode connection
```

**Test authentication:**
```bash
./mailtester -host smtp.example.com -port 587 -user alice -pass secret -mode auth
```

**Send a test message via STARTTLS + PLAIN auth:**
```bash
./mailtester -host smtp.example.com -port 587 -starttls \
  -from alice@example.com -to bob@example.net \
  -user alice -pass secret -mode send
```

**Test implicit TLS (port 465):**
```bash
./mailtester -host smtp.example.com -port 465 -tls \
  -from alice@example.com -to bob@example.net \
  -user alice -pass secret -mode ssl
```

**Run full diagnostic suite:**
```bash
./mailtester -host smtp.example.com -port 587 -starttls \
  -from alice@example.com -to bob@example.net \
  -user alice -pass secret -mode all
```

## Notes for AI Agents

- Always specify `-mode` explicitly when the user describes a specific test; default `all` requires `-from` and `-to` for send phases.
- If the user mentions "port 465", add `-tls` and suggest `ssl` mode. If they mention "port 587", add `-starttls`.
- `-skip-verify` is useful for self-signed certificates but should only be suggested after a normal TLS failure.
- `-timeout` applies to all connection paths: `net.DialTimeout` for plaintext/STARTTLS, `tls.DialWithDialer` for implicit TLS.
- The tool uses `github.com/emersion/go-smtp` under the hood; errors come from that library.
