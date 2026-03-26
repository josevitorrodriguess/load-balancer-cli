package proxy

import (
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

	"github.com/josevitorrodriguess/load-balancer-cli/internal/balancer"
)

func StartProxy(mux *http.ServeMux, lb balancer.Balancer) error {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		backend, err := lb.NextBackend()
		if err != nil {
			http.Error(w, "no backend available", http.StatusBadGateway)
			return
		}

		serverParsed, err := url.Parse(backend.URL)
		if err != nil {
			http.Error(w, "invalid backend url", http.StatusInternalServerError)
			return
		}

		prox := httputil.NewSingleHostReverseProxy(serverParsed)

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

		prox.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			_ = lb.ReportFailure(backend.URL)
			ErrorHandler(w, r, err)
		}

		prox.ServeHTTP(w, r)
	})

	return nil
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
	type writeHeaderTracker interface {
		Written() bool
	}

	tracker, ok := w.(writeHeaderTracker)
	return ok && tracker.Written()
}
