package middleware

import (
	"bufio"
	"bytes"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLoggerRecordsStatusAndDuration(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	handler := RequestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/logged", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("handler status changed: got %d", w.Code)
	}
	logLine := buf.String()
	for _, want := range []string{"method=POST", "path=/logged", "status=201", "duration=", "remote_addr=192.0.2.1:1234", "user_agent=test-agent"} {
		if !strings.Contains(logLine, want) {
			t.Fatalf("expected log to contain %q, got %q", want, logLine)
		}
	}
}

func TestStatusRecorderDelegatesHijacker(t *testing.T) {
	rw := &hijackableResponseWriter{}
	recorder := &statusRecorder{ResponseWriter: rw}

	conn, _, err := recorder.Hijack()
	if err != nil {
		t.Fatalf("expected hijack to be delegated: %v", err)
	}
	if conn == nil {
		t.Fatal("expected delegated connection")
	}
	if !rw.hijacked {
		t.Fatal("underlying writer was not hijacked")
	}
}

type hijackableResponseWriter struct {
	http.ResponseWriter
	hijacked bool
}

func (w *hijackableResponseWriter) Header() http.Header {
	return http.Header{}
}

func (w *hijackableResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (w *hijackableResponseWriter) WriteHeader(statusCode int) {}

func (w *hijackableResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.hijacked = true
	server, client := net.Pipe()
	client.Close()
	return server, bufio.NewReadWriter(bufio.NewReader(server), bufio.NewWriter(server)), nil
}
