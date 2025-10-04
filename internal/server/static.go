package server

import (
	"net/http"
	"path/filepath"
)

// AddCorsHeaders middleware untuk menambahkan CORS dan Cache-Control
func AddCorsHeaders(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		// Jika request adalah OPTIONS (preflight), balas langsung
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// GetTemplatesPath returns the path to templates directory
func GetTemplatesPath() string {
	return filepath.Join("..", "..", "web", "templates")
}
