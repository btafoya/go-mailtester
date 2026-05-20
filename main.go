package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/textproto"
	"os"
	"strings"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

type Config struct {
	Host        string
	Port        int
	From        string
	To          []string
	Subject     string
	Body        string
	AuthType    string
	Username    string
	Password    string
	UseTLS      bool
	UseStartTLS bool
	SkipVerify  bool
	Timeout     time.Duration
	TestMode    string
	Helo        string
}

func main() {
	cfg := parseFlags()

	switch cfg.TestMode {
	case "connection":
		testConnection(cfg)
	case "auth":
		testAuth(cfg)
	case "send":
		testSend(cfg)
	case "sendmail":
		testSendMail(cfg)
	case "sendmailtls":
		testSendMailTLS(cfg)
	case "ssl":
		testSSL(cfg)
	case "starttls":
		testStartTLS(cfg)
	case "raw":
		testRawSession(cfg)
	case "all":
		runAllTests(cfg)
	default:
		log.Fatalf("Unknown test mode: %s", cfg.TestMode)
	}
}

func parseFlags() *Config {
	cfg := &Config{}

	var toStr string
	flag.StringVar(&cfg.Host, "host", "localhost", "SMTP server hostname")
	flag.IntVar(&cfg.Port, "port", 25, "SMTP server port")
	flag.StringVar(&cfg.From, "from", "", "Sender email address")
	flag.StringVar(&toStr, "to", "", "Recipient email address(es), comma-separated")
	flag.StringVar(&cfg.Subject, "subject", "SMTP Test", "Email subject")
	flag.StringVar(&cfg.Body, "body", "This is a test email from go-mailtester.", "Email body")
	flag.StringVar(&cfg.AuthType, "auth", "plain", "Auth type: plain, login, cram-md5, none")
	flag.StringVar(&cfg.Username, "user", "", "SMTP username")
	flag.StringVar(&cfg.Password, "pass", "", "SMTP password")
	flag.BoolVar(&cfg.UseTLS, "tls", false, "Use implicit TLS")
	flag.BoolVar(&cfg.UseStartTLS, "starttls", false, "Use STARTTLS")
	flag.BoolVar(&cfg.SkipVerify, "skip-verify", false, "Skip TLS certificate verification")
	flag.DurationVar(&cfg.Timeout, "timeout", 30*time.Second, "Connection timeout")
	flag.StringVar(&cfg.TestMode, "mode", "all", "Test mode: connection, auth, send, sendmail, sendmailtls, ssl, starttls, raw, all")
	flag.StringVar(&cfg.Helo, "helo", "go-mailtester", "HELO/EHLO hostname")

	flag.Parse()

	if toStr != "" {
		cfg.To = splitEmails(toStr)
	}

	return cfg
}

func splitEmails(s string) []string {
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func addr(cfg *Config) string {
	return fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
}

func tlsConfig(cfg *Config) *tls.Config {
	return &tls.Config{
		ServerName:         cfg.Host,
		InsecureSkipVerify: cfg.SkipVerify,
	}
}

func getAuth(cfg *Config) (sasl.Client, error) {
	switch strings.ToLower(cfg.AuthType) {
	case "plain", "":
		return sasl.NewPlainClient("", cfg.Username, cfg.Password), nil
	case "login":
		return sasl.NewLoginClient(cfg.Username, cfg.Password), nil
	case "cram-md5":
		return nil, fmt.Errorf("CRAM-MD5 not yet implemented")
	case "none":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported auth type: %s", cfg.AuthType)
	}
}

func dialWithTimeout(cfg *Config) (*smtp.Client, error) {
	if cfg.UseTLS {
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: cfg.Timeout}, "tcp", addr(cfg), tlsConfig(cfg))
		if err != nil {
			return nil, err
		}
		return smtp.NewClient(conn), nil
	}
	if cfg.UseStartTLS {
		conn, err := net.DialTimeout("tcp", addr(cfg), cfg.Timeout)
		if err != nil {
			return nil, err
		}
		return smtp.NewClientStartTLS(conn, tlsConfig(cfg))
	}
	conn, err := net.DialTimeout("tcp", addr(cfg), cfg.Timeout)
	if err != nil {
		return nil, err
	}
	return smtp.NewClient(conn), nil
}

func testConnection(cfg *Config) {
	fmt.Printf("Testing connection to %s...\n", addr(cfg))

	conn, err := net.DialTimeout("tcp", addr(cfg), cfg.Timeout)
	if err != nil {
		log.Fatalf("Connection failed: %v", err)
	}
	defer conn.Close()

	fmt.Println("TCP connection: OK")

	client := smtp.NewClient(conn)
	defer client.Close()

	fmt.Println("SMTP handshake: OK")

	if err := client.Hello(cfg.Helo); err != nil {
		log.Fatalf("EHLO failed: %v", err)
	}
	fmt.Println("EHLO: OK")

	extensions := []string{"STARTTLS", "AUTH", "8BITMIME", "PIPELINING", "SIZE"}
	for _, ext := range extensions {
		if ok, arg := client.Extension(ext); ok {
			fmt.Printf("Extension %s: %s\n", ext, arg)
		}
	}
}

func testAuth(cfg *Config) {
	fmt.Printf("Testing authentication to %s...\n", addr(cfg))

	client, err := dialWithTimeout(cfg)
	if err != nil {
		log.Fatalf("Dial failed: %v", err)
	}
	defer client.Close()

	if err := client.Hello(cfg.Helo); err != nil {
		log.Fatalf("EHLO failed: %v", err)
	}

	auth, err := getAuth(cfg)
	if err != nil {
		log.Fatalf("Auth setup failed: %v", err)
	}
	if auth == nil {
		fmt.Println("Auth type 'none' - skipping authentication")
		return
	}

	if err := client.Auth(auth); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}
	fmt.Println("Authentication: OK")
}

func testSend(cfg *Config) {
	if cfg.From == "" || len(cfg.To) == 0 {
		log.Fatal("-from and -to are required for send test")
	}

	fmt.Printf("Testing send to %s...\n", addr(cfg))

	client, err := dialWithTimeout(cfg)
	if err != nil {
		log.Fatalf("Dial failed: %v", err)
	}
	defer client.Close()

	if err := client.Hello(cfg.Helo); err != nil {
		log.Fatalf("EHLO failed: %v", err)
	}

	auth, err := getAuth(cfg)
	if err != nil {
		log.Fatalf("Auth setup failed: %v", err)
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			log.Fatalf("Authentication failed: %v", err)
		}
		fmt.Println("Authentication: OK")
	}

	if err := client.Mail(cfg.From, nil); err != nil {
		log.Fatalf("MAIL FROM failed: %v", err)
	}
	fmt.Printf("MAIL FROM %s: OK\n", cfg.From)

	for _, to := range cfg.To {
		if err := client.Rcpt(to, nil); err != nil {
			log.Fatalf("RCPT TO %s failed: %v", to, err)
		}
		fmt.Printf("RCPT TO %s: OK\n", to)
	}

	msg, err := client.Data()
	if err != nil {
		log.Fatalf("DATA failed: %v", err)
	}

	fmt.Fprintf(msg, "From: %s\r\n", cfg.From)
	fmt.Fprintf(msg, "To: %s\r\n", strings.Join(cfg.To, ", "))
	fmt.Fprintf(msg, "Subject: %s\r\n", cfg.Subject)
	fmt.Fprintf(msg, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(msg, "Message-Id: <%s@%s>\r\n", generateMsgID(), cfg.Host)
	fmt.Fprint(msg, "\r\n")
	fmt.Fprintf(msg, "%s\r\n", cfg.Body)

	if err := msg.Close(); err != nil {
		log.Fatalf("Message close failed: %v", err)
	}
	fmt.Println("Message sent: OK")

	if err := client.Quit(); err != nil {
		log.Fatalf("QUIT failed: %v", err)
	}
	fmt.Println("QUIT: OK")
}

func testSendMail(cfg *Config) {
	if cfg.From == "" || len(cfg.To) == 0 {
		log.Fatal("-from and -to are required for sendmail test")
	}

	fmt.Printf("Testing SendMail to %s...\n", addr(cfg))

	auth, err := getAuth(cfg)
	if err != nil {
		log.Fatalf("Auth setup failed: %v", err)
	}

	msg := buildMessage(cfg)
	if err := smtp.SendMail(addr(cfg), auth, cfg.From, cfg.To, strings.NewReader(msg)); err != nil {
		log.Fatalf("SendMail failed: %v", err)
	}
	fmt.Println("SendMail: OK")
}

func testSendMailTLS(cfg *Config) {
	if cfg.From == "" || len(cfg.To) == 0 {
		log.Fatal("-from and -to are required for sendmailtls test")
	}

	fmt.Printf("Testing SendMailTLS to %s...\n", addr(cfg))

	auth, err := getAuth(cfg)
	if err != nil {
		log.Fatalf("Auth setup failed: %v", err)
	}

	msg := buildMessage(cfg)
	if err := smtp.SendMailTLS(addr(cfg), auth, cfg.From, cfg.To, strings.NewReader(msg)); err != nil {
		log.Fatalf("SendMailTLS failed: %v", err)
	}
	fmt.Println("SendMailTLS: OK")
}

func testSSL(cfg *Config) {
	fmt.Printf("Testing implicit TLS to %s...\n", addr(cfg))

	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: cfg.Timeout}, "tcp", addr(cfg), tlsConfig(cfg))
	if err != nil {
		log.Fatalf("TLS dial failed: %v", err)
	}
	defer conn.Close()

	fmt.Println("TLS connection: OK")

	state := conn.ConnectionState()
	fmt.Printf("TLS Version: %x\n", state.Version)
	fmt.Printf("Cipher Suite: %x\n", state.CipherSuite)
	fmt.Printf("Server Name: %s\n", state.ServerName)
	fmt.Printf("Handshake Complete: %v\n", state.HandshakeComplete)

	client := smtp.NewClient(conn)
	defer client.Close()

	if err := client.Hello(cfg.Helo); err != nil {
		log.Fatalf("EHLO failed: %v", err)
	}
	fmt.Println("EHLO: OK")

	extensions := []string{"AUTH", "8BITMIME", "PIPELINING", "SIZE"}
	for _, ext := range extensions {
		if ok, arg := client.Extension(ext); ok {
			fmt.Printf("Extension %s: %s\n", ext, arg)
		}
	}

	auth, err := getAuth(cfg)
	if err != nil {
		log.Fatalf("Auth setup failed: %v", err)
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			log.Fatalf("Authentication failed: %v", err)
		}
		fmt.Println("Authentication: OK")
	}
}

func testStartTLS(cfg *Config) {
	fmt.Printf("Testing STARTTLS to %s...\n", addr(cfg))

	conn, err := net.DialTimeout("tcp", addr(cfg), cfg.Timeout)
	if err != nil {
		log.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	client := smtp.NewClient(conn)

	if err := client.Hello(cfg.Helo); err != nil {
		client.Close()
		log.Fatalf("EHLO failed: %v", err)
	}

	if ok, _ := client.Extension("STARTTLS"); !ok {
		client.Close()
		log.Fatal("STARTTLS not supported by server")
	}
	client.Close()

	conn2, err := net.DialTimeout("tcp", addr(cfg), cfg.Timeout)
	if err != nil {
		log.Fatalf("Re-dial for STARTTLS failed: %v", err)
	}
	defer conn2.Close()

	client, err = smtp.NewClientStartTLS(conn2, tlsConfig(cfg))
	if err != nil {
		log.Fatalf("STARTTLS failed: %v", err)
	}
	defer client.Close()
	fmt.Println("STARTTLS negotiation: OK")

	state, ok := client.TLSConnectionState()
	if !ok {
		log.Fatal("TLS connection state not available")
	}
	fmt.Printf("TLS Version: %x\n", state.Version)
	fmt.Printf("Cipher Suite: %x\n", state.CipherSuite)
	fmt.Printf("Server Name: %s\n", state.ServerName)
	fmt.Printf("Handshake Complete: %v\n", state.HandshakeComplete)
}

func testRawSession(cfg *Config) {
	fmt.Printf("Testing raw SMTP session to %s...\n", addr(cfg))

	conn, err := net.DialTimeout("tcp", addr(cfg), cfg.Timeout)
	if err != nil {
		log.Fatalf("Connection failed: %v", err)
	}
	defer conn.Close()

	text := textproto.NewConn(conn)
	defer text.Close()

	_, err = text.ReadLine()
	if err != nil {
		log.Fatalf("Failed to read greeting: %v", err)
	}
	fmt.Println("Server greeting received")

	if err := text.PrintfLine("EHLO %s", cfg.Helo); err != nil {
		log.Fatalf("EHLO write failed: %v", err)
	}

	for {
		line, err := text.ReadLine()
		if err != nil {
			log.Fatalf("EHLO read failed: %v", err)
		}
		fmt.Printf("  < %s\n", line)
		if len(line) < 4 || line[3] != '-' {
			break
		}
	}
	fmt.Println("EHLO complete")

	if err := text.PrintfLine("QUIT"); err != nil {
		log.Fatalf("QUIT failed: %v", err)
	}
	line, _ := text.ReadLine()
	fmt.Printf("  < %s\n", line)
	fmt.Println("QUIT complete")
}

func runAllTests(cfg *Config) {
	fmt.Println("=== Running all tests ===")
	testConnection(cfg)
	fmt.Println()
	testStartTLS(cfg)
	fmt.Println()
	if cfg.UseTLS {
		testSSL(cfg)
		fmt.Println()
	}
	testAuth(cfg)
	fmt.Println()
	if cfg.From != "" && len(cfg.To) > 0 {
		testSend(cfg)
		fmt.Println()
		testSendMail(cfg)
		fmt.Println()
		if cfg.UseTLS {
			testSendMailTLS(cfg)
			fmt.Println()
		}
	} else {
		fmt.Println("Skipping send tests: -from and -to required")
	}
	testRawSession(cfg)
	fmt.Println("=== All tests complete ===")
}

func buildMessage(cfg *Config) string {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", cfg.From)
	fmt.Fprintf(&b, "To: %s\r\n", strings.Join(cfg.To, ", "))
	fmt.Fprintf(&b, "Subject: %s\r\n", cfg.Subject)
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-Id: <%s@%s>\r\n", generateMsgID(), cfg.Host)
	fmt.Fprintf(&b, "\r\n")
	fmt.Fprintf(&b, "%s\r\n", cfg.Body)
	return b.String()
}

func generateMsgID() string {
	return fmt.Sprintf("%d.%d", time.Now().Unix(), os.Getpid())
}

