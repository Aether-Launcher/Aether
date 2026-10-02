package theme

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/Aether-Launcher/Aether/pkg/logger"
)

// Server serves the themes directory statically so <img> tags in the
// frontend can load theme-provided PNGs by URL.
type Server struct {
	port     int
	listener net.Listener
	mu       sync.Mutex
}

// NewServer creates a new local theme asset server.
func NewServer() *Server {
	return &Server{}
}

// Start launches the HTTP server on a random available port and returns the
// base URL. If already running, it returns the existing URL.
func (s *Server) Start() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener != nil {
		return fmt.Sprintf("http://127.0.0.1:%d", s.port), nil
	}

	fsHandler := http.FileServer(http.Dir(getThemesDir()))
	mux := http.NewServeMux()
	mux.Handle("/", enableCORS(fsHandler))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to start theme asset server: %w", err)
	}

	s.listener = listener
	s.port = listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d", s.port)

	go func() {
		logger.Info("Theme", fmt.Sprintf("Asset server listening at %s", url))
		if err := http.Serve(listener, mux); err != nil && err != http.ErrServerClosed {
			logger.Error("Theme", fmt.Sprintf("Server error: %v", err))
		}
	}()

	return url, nil
}

// Stop gracefully shuts down the server and releases the port.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return nil
	}
	err := s.listener.Close()
	s.listener = nil
	s.port = 0
	return err
}

// URL returns the server's current base URL, or "" if not running.
func (s *Server) URL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return ""
	}
	return fmt.Sprintf("http://127.0.0.1:%d", s.port)
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
