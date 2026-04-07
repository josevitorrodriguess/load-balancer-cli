package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/josevitorrodriguess/load-balancer-cli/internal/balancer"
)

const maxAttempts = 3

type Config struct {
	DialTimeout           time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
}

func DefaultConfig() Config {
	return Config{
		DialTimeout:           3 * time.Second,
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}
}

func StartProxy(mux *http.ServeMux, lb balancer.Balancer) error {
	return StartProxyWithConfig(mux, lb, DefaultConfig())
}

func StartProxyWithConfig(mux *http.ServeMux, lb balancer.Balancer, cfg Config) error {
	transport := newTransport(cfg)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{ResponseWriter: w}
		body, err := readRequestBody(r)
		if err != nil {
			slog.Error("proxy error", "method", r.Method, "path", r.URL.Path, "error", err)
			http.Error(rw, "failed to read request body", http.StatusBadRequest)
			return
		}

		tried := make(map[string]struct{})
		var lastErr error

		for attempt := 0; attempt < maxAttempts; attempt++ {
			backend, err := nextBackend(lb, tried)
			if err != nil {
				if lastErr != nil {
					ErrorHandler(rw, r, lastErr)
					return
				}

				http.Error(rw, "no backend available", http.StatusBadGateway)
				return
			}

			tried[backend.URL] = struct{}{}
			if err := lb.IncrementActiveConnections(backend.URL); err != nil {
				slog.Error("proxy error", "method", r.Method, "path", r.URL.Path, "backend_url", backend.URL, "error", err)
				http.Error(rw, "no backend available", http.StatusBadGateway)
				return
			}

			serverParsed, err := url.Parse(backend.URL)
			if err != nil {
				_ = lb.DecrementActiveConnections(backend.URL)
				http.Error(rw, "invalid backend url", http.StatusInternalServerError)
				return
			}

			prox := httputil.NewSingleHostReverseProxy(serverParsed)
			prox.Transport = transport
			originalDirector := prox.Director

			prox.Director = func(req *http.Request) {
				originalDirector(req)

				req.URL.Scheme = serverParsed.Scheme
				req.URL.Host = serverParsed.Host
				req.Host = serverParsed.Host

				ip, _, err := net.SplitHostPort(req.RemoteAddr)
				if err != nil {
					ip = req.RemoteAddr
				}

				req.Header.Set("X-Forwarded-For", ip)
				req.Header.Set("X-Real-IP", ip)

				if req.TLS != nil {
					req.Header.Set("X-Forwarded-Proto", "https")
				} else {
					req.Header.Set("X-Forwarded-Proto", "http")
				}
			}

			attemptFailed := false
			prox.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
				attemptFailed = true
				lastErr = err
				_ = lb.ReportFailure(backend.URL)
			}

			prox.ServeHTTP(rw, cloneRequest(r, body))
			_ = lb.DecrementActiveConnections(backend.URL)

			if !attemptFailed {
				return
			}

			if headersWritten(rw) {
				return
			}
		}

		if lastErr != nil {
			ErrorHandler(rw, r, lastErr)
		}
	})

	return nil
}

func newTransport(cfg Config) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: cfg.DialTimeout,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
		ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
	}
}

func ErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := classifyProxyError(err)

	slog.Error("proxy error",
		"method", r.Method,
		"path", r.URL.Path,
		"status", status,
		"error", err,
	)

	if headersWritten(w) {
		return
	}

	writeProxyError(w, status, code, message)
}

func classifyProxyError(err error) (status int, code, message string) {
	switch {
	case errors.Is(err, context.Canceled):
		return 499, "client_closed_request", "Client canceled the request"
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout, "backend_timeout", "Backend took too long to respond"
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return http.StatusGatewayTimeout, "backend_timeout", "Backend took too long to respond"
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return http.StatusBadGateway, "backend_connection_failed", "Failed to connect to backend"
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return http.StatusBadGateway, "backend_request_failed", "Backend request failed"
	}

	if errors.Is(err, io.EOF) || strings.Contains(strings.ToLower(err.Error()), "connection reset by peer") {
		return http.StatusBadGateway, "backend_connection_closed", "Backend closed the connection unexpectedly"
	}

	return http.StatusBadGateway, "proxy_error", "Failed to proxy request to backend"
}

func writeProxyError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	payload := map[string]string{
		"error":   http.StatusText(status),
		"code":    code,
		"message": message,
	}

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("proxy error response write failed", "error", err)
	}
}

func headersWritten(w http.ResponseWriter) bool {
	tracker, ok := w.(*responseWriter)
	return ok && tracker.written
}

func readRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	return body, nil
}

func cloneRequest(r *http.Request, body []byte) *http.Request {
	req := r.Clone(r.Context())
	req.Header = r.Header.Clone()
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}

	return req
}

func nextBackend(lb balancer.Balancer, tried map[string]struct{}) (*balancer.Backend, error) {
	backends := lb.Backends()
	if len(backends) == 0 {
		return nil, balancer.ErrNoBackendAvailable
	}

	for i := 0; i < len(backends); i++ {
		backend, err := lb.NextBackend()
		if err != nil {
			return nil, err
		}

		if _, ok := tried[backend.URL]; ok {
			continue
		}

		return backend, nil
	}

	return nil, balancer.ErrNoBackendAvailable
}

type responseWriter struct {
	http.ResponseWriter
	written bool
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.written = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Write(data []byte) (int, error) {
	w.written = true
	return w.ResponseWriter.Write(data)
}
