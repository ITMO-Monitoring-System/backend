package middleware

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/gorilla/mux"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (w *responseRecorder) Unwrap() http.ResponseWriter { return w.ResponseWriter }

var safeRequestID = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

func (w *responseRecorder) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseRecorder) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(data)
}

func (w *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	w.status = http.StatusSwitchingProtocols
	return hijacker.Hijack()
}

func (w *responseRecorder) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *responseRecorder) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

func (w *responseRecorder) ReadFrom(reader io.Reader) (int64, error) {
	if rf, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		if w.status == 0 {
			w.status = http.StatusOK
		}
		return rf.ReadFrom(reader)
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return io.Copy(w.ResponseWriter, reader)
}

func NewLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		begin := time.Now()
		id := r.Header.Get("X-Request-ID")
		if !safeRequestID.MatchString(id) {
			bytes := make([]byte, 16)
			_, _ = rand.Read(bytes)
			id = hex.EncodeToString(bytes)
		}
		w.Header().Set("X-Request-ID", id)
		recorder := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		if recorder.status == 0 {
			recorder.status = http.StatusOK
		}
		path := "[unmatched]"
		if route := mux.CurrentRoute(r); route != nil {
			if template, err := route.GetPathTemplate(); err == nil {
				path = template
			}
		}
		slog.Info("http_request", "service", "backend", "event", "http_request", "method", r.Method,
			"path", path, "status", recorder.status, "duration_ms", time.Since(begin).Milliseconds(), "request_id", id)
	})
}
