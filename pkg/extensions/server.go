package extensions

import (
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"Aether/pkg/fs"
)

// Server handles serving extension UI files locally for iframes
type Server struct {
	port     int
	listener net.Listener
	mu       sync.Mutex
}

// NewServer creates a new local extension server
func NewServer() *Server {
	return &Server{}
}

// Start launches the HTTP server on a random available port and returns the base URL.
// If the server is already running, it returns the existing URL.
func (s *Server) Start() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener != nil {
		return fmt.Sprintf("http://127.0.0.1:%d", s.port), nil
	}

	extDir := filepath.Join(fs.GetDataDir(), "extensions")
	fsHandler := http.FileServer(http.Dir(extDir))

	mux := http.NewServeMux()
	mux.Handle("/", enableCORS(fsHandler))
	mux.HandleFunc("/_screenshots/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		rel := strings.TrimPrefix(r.URL.Path, "/_screenshots/")
		parts := strings.SplitN(rel, "/", 2)
		if len(parts) < 2 {
			http.Error(w, "Invalid screenshot request", http.StatusBadRequest)
			return
		}
		instanceID := parts[0]
		fileName := filepath.Base(parts[1])

		instanceDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), instanceID)
		if err != nil {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		imgPath, err := fs.ContainedPath(filepath.Join(instanceDir, "screenshots"), fileName)
		if err != nil {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		http.ServeFile(w, r, imgPath)
	})

	// Bind to port 0 to let the OS assign a random available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to start local server: %w", err)
	}

	s.listener = listener
	s.port = listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d", s.port)

	go func() {
		fmt.Printf("[Extensions] UI Server listening at %s\n", url)
		if err := http.Serve(listener, mux); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[Extensions] Server error: %v\n", err)
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

// GetPort returns the currently bound port
func (s *Server) GetPort() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.port
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow the Wails frontend (usually wails:// or http://localhost:...) to load these assets
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
