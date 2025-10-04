package main

import (
	"log"
	"net/http"

	"websocket-ai/internal/gateway"
)

const PORT = "8081"

func main() {
	log.Println("Starting AI Gateway WebSocket server on port", PORT)

	// Serve static files for the web interface
	http.Handle("/", http.FileServer(http.Dir("../../web/templates")))

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := gateway.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Failed to upgrade connection:", err)
			return
		}
		gateway.HandleClient(conn)
	})

	go gateway.CleanupConnections()

	if err := http.ListenAndServe("127.0.0.1:"+PORT, nil); err != nil {
		log.Fatal("AI Gateway failed to start:", err)
	}
}
