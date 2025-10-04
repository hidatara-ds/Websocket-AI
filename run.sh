#!/bin/bash

echo "Starting WebSocket AI Assistant..."

echo ""
echo "Starting Static File Server (Port 8080)..."
cd cmd/static-server && go run main.go &
STATIC_PID=$!

sleep 2

echo ""
echo "Starting AI Gateway (Port 8081)..."
cd ../ai-gateway && go run main.go &
GATEWAY_PID=$!

echo ""
echo "Both servers are starting..."
echo "Static File Server: http://localhost:8080"
echo "AI Gateway WebSocket: ws://localhost:8081/ws"
echo ""
echo "Press Ctrl+C to stop all servers..."

# Function to cleanup on exit
cleanup() {
    echo ""
    echo "Stopping servers..."
    kill $STATIC_PID $GATEWAY_PID 2>/dev/null
    echo "Servers stopped."
    exit 0
}

# Set trap to cleanup on script exit
trap cleanup SIGINT SIGTERM

# Wait for user to stop
wait
