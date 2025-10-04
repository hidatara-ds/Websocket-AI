package main

import (
	"fmt"
	"log"
	"net/http"

	"websocket-ai/internal/server"
)

const PORT = "8080"

func main() {
	// Path ke folder templates
	templatesPath := server.GetTemplatesPath()
	fs := http.FileServer(http.Dir(templatesPath))
	http.Handle("/", server.AddCorsHeaders(fs))

	fmt.Printf("Starting Static File Server at http://localhost:%s\n", PORT)
	log.Fatal(http.ListenAndServe(":"+PORT, nil))
}
