package main

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testToken = "test-service-token-0123456789-abcdefghijklmnopqrstuvwxyz"

func gatewayFor(t *testing.T, upstream string) http.Handler {
	t.Helper()
	handler, err := NewGateway(Config{AgentURL: upstream, Token: testToken, StaticRoot: t.TempDir(), SchemaPath: "openapi.json"})
	if err != nil {
		t.Fatal(err)
	}
	return handler
}
func post(t *testing.T, handler http.Handler, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest("POST", "http://localhost:9400/api/agent/query", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestValidateBeforeUpstream(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(200) }))
	defer upstream.Close()
	handler := gatewayFor(t, upstream.URL)
	for _, entry := range []struct {
		body   string
		status int
	}{{`{}`, 422}, {`{"prompt":"  "}`, 422}, {`{"prompt":42}`, 422}, {`{"prompt":null}`, 422}, {`{"prompt":"x","extra":true}`, 422}, {`{broken`, 400}, {`{"prompt":"` + strings.Repeat("x", 4001) + `"}`, 422}, {`{"prompt":"` + strings.Repeat("x", 40000) + `"}`, 413}} {
		if response := post(t, handler, entry.body, nil); response.Code != entry.status {
			t.Fatalf("expected %d, got %d: %s", entry.status, response.Code, response.Body.String())
		}
	}
	if response := post(t, handler, `{"prompt":"hello"}`, map[string]string{"Content-Type": "application/json-bogus"}); response.Code != 415 {
		t.Fatal(response.Code)
	}
	if response := post(t, handler, `{"prompt":"hello"}`, map[string]string{"Origin": "https://foreign.invalid", "X-Forwarded-Host": "foreign.invalid"}); response.Code != 403 {
		t.Fatal(response.Code)
	}
	if calls.Load() != 0 {
		t.Fatal("invalid requests reached the private service")
	}
}

func TestNormalizeAndAttachServerCredential(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Agent-Service-Token") != testToken {
			t.Error("server token missing")
		}
		if r.Header.Get("Origin") != "" || r.Header.Get("Cookie") != "" {
			t.Error("browser credentials forwarded")
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"prompt":"hello"}` {
			t.Error(string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"result":"real service result","is_error":false}`)
	}))
	defer upstream.Close()
	handler := gatewayFor(t, upstream.URL)
	response := post(t, handler, `{"prompt":" hello "}`, map[string]string{"Origin": "http://localhost:9400", "Cookie": "browser=private", "X-Agent-Service-Token": "spoofed"})
	if response.Code != 200 || !strings.Contains(response.Body.String(), "real service result") {
		t.Fatal(response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), testToken) {
		t.Fatal("token exposed")
	}
}

func TestStreamFlushesBeforeUpstreamFinishes(t *testing.T) {
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: message\ndata: {\"text\":\"héllo\"}\n\n")
		w.(http.Flusher).Flush()
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		_, _ = io.WriteString(w, "event: done\ndata: {}\n\n")
	}))
	defer upstream.Close()
	gateway := httptest.NewServer(gatewayFor(t, upstream.URL))
	defer gateway.Close()
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Post(gateway.URL+"/api/agent/stream", "application/json", strings.NewReader(`{"prompt":"hello"}`))
	if err != nil {
		close(release)
		t.Fatal(err)
	}
	defer response.Body.Close()
	reader := bufio.NewReader(response.Body)
	line, err := reader.ReadString('\n')
	if err != nil || line != "event: message\n" {
		close(release)
		t.Fatal(line, err)
	}
	close(release)
	rest, err := io.ReadAll(reader)
	if err != nil || !strings.Contains(string(rest), "héllo") || !strings.Contains(string(rest), "event: done") {
		t.Fatal(string(rest), err)
	}
}

func TestCancellationReachesPrivateService(t *testing.T) {
	cancelled := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: message\ndata: {}\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(cancelled)
	}))
	defer upstream.Close()
	gateway := httptest.NewServer(gatewayFor(t, upstream.URL))
	defer gateway.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, "POST", gateway.URL+"/api/agent/stream", strings.NewReader(`{"prompt":"hello"}`))
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = bufio.NewReader(response.Body).ReadString('\n')
	cancel()
	_ = response.Body.Close()
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("browser cancellation did not reach Node boundary")
	}
}

func TestStaticFilesAndAPIPrecedence(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "index.html"), []byte("<html>React</html>"), 0600)
	_ = os.WriteFile(filepath.Join(root, ".env"), []byte("never-public"), 0600)
	_ = os.Mkdir(filepath.Join(root, "assets"), 0700)
	_ = os.WriteFile(filepath.Join(root, "assets", "main.js"), []byte("const answer = 42;"), 0600)
	router, err := NewGateway(Config{AgentURL: "http://127.0.0.1:1", Token: testToken, StaticRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct {
		path   string
		status int
	}{{"/", 200}, {"/assets/main.js", 200}, {"/.env", 404}, {"/%2eenv", 404}, {"/api/unknown", 404}, {"/assets/", 404}} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest("GET", entry.path, nil))
		if recorder.Code != entry.status {
			t.Fatalf("%s: %d", entry.path, recorder.Code)
		}
	}
}

func TestMissingOrMismatchedService(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "private unauthorized diagnostic", 401) }))
	defer upstream.Close()
	response := post(t, gatewayFor(t, upstream.URL), `{"prompt":"hello"}`, nil)
	if response.Code != 503 || strings.Contains(response.Body.String(), "diagnostic") {
		t.Fatal(response.Code, response.Body.String())
	}
	if _, err := NewGateway(Config{AgentURL: upstream.URL, Token: ""}); err == nil {
		t.Fatal("missing token accepted")
	}
	if _, err := NewGateway(Config{AgentURL: upstream.URL + "/unexpected?path=x", Token: testToken}); err == nil {
		t.Fatal("ambiguous upstream URL accepted")
	}
}

func TestTrustedHTTPSConfiguration(t *testing.T) {
	cfg := Config{AgentURL: "http://127.0.0.1:1", Token: testToken, TrustProxy: true}
	router, _ := NewGateway(cfg)
	// Empty prompt validates origins without contacting the upstream.
	response := post(t, router, `{"prompt":""}`, map[string]string{"Origin": "https://localhost:9400", "X-Forwarded-Proto": "https"})
	if response.Code != 422 {
		t.Fatal(response.Code)
	}
	cfg.TrustProxy = false
	router, _ = NewGateway(cfg)
	response = post(t, router, `{"prompt":""}`, map[string]string{"Origin": "https://localhost:9400", "X-Forwarded-Proto": "https"})
	if response.Code != 403 {
		t.Fatal(response.Code)
	}
}
