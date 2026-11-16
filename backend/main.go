package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unicode/utf16"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type Config struct {
	AgentURL   string
	Token      string
	StaticRoot string
	SchemaPath string
	TrustProxy bool
}

func errorJSON(c *gin.Context, status int, detail string) {
	c.AbortWithStatusJSON(status, gin.H{"detail": detail})
}

func validatePrompt(c *gin.Context, cfg Config) (string, bool) {
	origin := c.GetHeader("Origin")
	scheme := "http"
	if c.Request.TLS != nil || (cfg.TrustProxy && c.GetHeader("X-Forwarded-Proto") == "https") {
		scheme = "https"
	}
	if origin != "" && origin != scheme+"://"+c.Request.Host {
		errorJSON(c, 403, "Origin is not allowed.")
		return "", false
	}
	media, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || media != "application/json" {
		errorJSON(c, 415, "Use Content-Type: application/json.")
		return "", false
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32768)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var large *http.MaxBytesError
		if errors.As(err, &large) {
			errorJSON(c, 413, "Request body is too large.")
		} else {
			errorJSON(c, 400, "Request must contain valid JSON.")
		}
		return "", false
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(body, &fields); err != nil {
		errorJSON(c, 400, "Request must contain valid JSON.")
		return "", false
	}
	var prompt string
	if fields == nil || json.Unmarshal(fields["prompt"], &prompt) != nil || len(fields) != 1 {
		errorJSON(c, 422, "Prompt must contain 1–4000 characters after trimming.")
		return "", false
	}
	prompt = strings.TrimFunc(prompt, func(r rune) bool { return unicode.IsSpace(r) || r == '\uFEFF' })
	count := len(utf16.Encode([]rune(prompt)))
	if count < 1 || count > 4000 {
		errorJSON(c, 422, "Prompt must contain 1–4000 characters after trimming.")
		return "", false
	}
	return prompt, true
}

func NewGateway(cfg Config) (*gin.Engine, error) {
	upstream, err := url.Parse(cfg.AgentURL)
	if err != nil || upstream.Host == "" || (upstream.Scheme != "http" && upstream.Scheme != "https") || upstream.User != nil || upstream.RawQuery != "" || upstream.Fragment != "" || (upstream.Path != "" && upstream.Path != "/") {
		return nil, fmt.Errorf("invalid agent service URL")
	}
	if len(cfg.Token) < 32 || strings.HasPrefix(cfg.Token, "replace-") {
		return nil, fmt.Errorf("configure a matching private agent-service token; run npm run setup")
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.Use(gin.CustomRecovery(func(c *gin.Context, _ any) { errorJSON(c, 500, "An unexpected server error occurred.") }))
	client := &http.Client{Timeout: 190 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	proxy := func(c *gin.Context, path string, prompt string) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 185*time.Second)
		defer cancel()
		method := "GET"
		var body io.Reader
		if prompt != "" {
			method = "POST"
			encoded, _ := json.Marshal(gin.H{"prompt": prompt})
			body = strings.NewReader(string(encoded))
		}
		request, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(cfg.AgentURL, "/")+path, body)
		if err != nil {
			errorJSON(c, 503, "The Node agent service is unavailable.")
			return
		}
		request.Header.Set("X-Agent-Service-Token", cfg.Token)
		if method == "POST" {
			request.Header.Set("Content-Type", "application/json")
		}
		response, err := client.Do(request)
		if err != nil {
			if c.Request.Context().Err() == nil {
				errorJSON(c, 503, "The Node agent service is unavailable. Check that both servers are running.")
			}
			return
		}
		defer response.Body.Close()
		if response.StatusCode == 401 || response.StatusCode >= 300 && response.StatusCode < 400 {
			errorJSON(c, 503, "Agent-service configuration could not be verified.")
			return
		}
		for _, name := range []string{"Content-Type", "Cache-Control", "X-Accel-Buffering"} {
			if value := response.Header.Get(name); value != "" {
				c.Header(name, value)
			}
		}
		c.Status(response.StatusCode)
		streaming := strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream")
		if streaming {
			c.Writer.Flush()
		}
		buffer := make([]byte, 32768)
		for {
			n, readErr := response.Body.Read(buffer)
			if n > 0 {
				if _, err = c.Writer.Write(buffer[:n]); err != nil {
					return
				}
				if streaming {
					c.Writer.Flush()
				}
			}
			if readErr != nil {
				return
			}
		}
	}
	router.GET("/api/health", func(c *gin.Context) { proxy(c, "/api/health", "") })
	for _, mode := range []string{"query", "stream"} {
		path := "/api/agent/" + mode
		router.POST(path, func(c *gin.Context) {
			if prompt, ok := validatePrompt(c, cfg); ok {
				proxy(c, path, prompt)
			}
		})
	}
	router.GET("/api/docs", func(c *gin.Context) { c.Data(200, "text/html; charset=utf-8", []byte(docsHTML)) })
	router.GET("/api/openapi.json", func(c *gin.Context) { c.File(cfg.SchemaPath) })
	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") || c.Request.Method != "GET" && c.Request.Method != "HEAD" {
			errorJSON(c, 404, "Not found.")
			return
		}
		for _, part := range strings.Split(strings.ReplaceAll(path, "\\", "/"), "/") {
			if strings.HasPrefix(part, ".") {
				errorJSON(c, 404, "Not found.")
				return
			}
		}
		name := strings.TrimPrefix(path, "/")
		if name == "" {
			name = "index.html"
		}
		root, err := filepath.EvalSymlinks(cfg.StaticRoot)
		if err != nil {
			errorJSON(c, 503, "React build not found. Run npm run build or use the development frontend.")
			return
		}
		target, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			errorJSON(c, 404, "Not found.")
			return
		}
		rel, err := filepath.Rel(root, target)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			errorJSON(c, 404, "Not found.")
			return
		}
		file, err := os.Open(target)
		if err != nil {
			errorJSON(c, 404, "Not found.")
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil || !info.Mode().IsRegular() {
			errorJSON(c, 404, "Not found.")
			return
		}
		if name == "index.html" {
			c.Header("Cache-Control", "no-cache")
		} else if strings.HasPrefix(name, "assets/") {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
		http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), file)
	})
	return router, nil
}

const docsHTML = `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Smallest Agent API</title><style>body{font:21px/1.7 system-ui;background:#f4f8ff;color:#142547;max-width:1000px;margin:40px auto;padding:24px}article{background:#eaf7f0;padding:24px;border-radius:20px}code{overflow-wrap:anywhere}</style><h1>React + Go/Gin + Node Agent Service</h1><article><p>GET /api/health checks the authenticated Node service and reports model key presence.</p><p>POST /api/agent/query returns a normalized result; POST /api/agent/stream relays real UTF-8 SSE.</p><p>Use application/json and a trimmed 1–4000 character prompt. JSON/body/origin limits apply before forwarding. Missing key returns query 503 or stream error; done requires a final result and preserves result errors.</p><p>Only the server attaches the internal service token. No model key or service credential belongs in React.</p></article><p><a href="/api/openapi.json">OpenAPI schema</a> · <a href="/#setup">Setup</a></p></html>`

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
func main() {
	envFile := flag.String("env", ".env", "gateway environment file")
	static := flag.String("static", "../frontend/dist", "React build directory")
	schema := flag.String("schema", "openapi.json", "OpenAPI schema path")
	flag.Parse()
	if err := godotenv.Load(*envFile); err != nil && !os.IsNotExist(err) {
		log.Fatal("Cannot load gateway configuration")
	}
	router, err := NewGateway(Config{AgentURL: env("AGENT_SERVICE_URL", "http://127.0.0.1:9401"), Token: env("AGENT_SERVICE_TOKEN", ""), StaticRoot: *static, SchemaPath: *schema, TrustProxy: env("TRUST_PROXY", "0") == "1"})
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Addr: env("API_HOST", "127.0.0.1") + ":" + env("API_PORT", "9400"), Handler: router, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stopped
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()
	log.Printf("Gin gateway ready at %s", server.Addr)
	if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("Gateway could not start")
	}
}
