@echo off
echo Starting WebSocket AI Assistant...

echo.
echo Starting Static File Server (Port 8080)...
start "Static Server" cmd /k "cd cmd\static-server && go run main.go"

timeout /t 2 /nobreak >nul

echo.
echo Starting AI Gateway (Port 8081)...
start "AI Gateway" cmd /k "cd cmd\ai-gateway && go run main.go"

echo.
echo Both servers are starting...
echo Static File Server: http://localhost:8080
echo AI Gateway WebSocket: ws://localhost:8081/ws
echo.
echo Press any key to stop all servers...
pause >nul

echo Stopping servers...
taskkill /f /im go.exe >nul 2>&1
echo Servers stopped.
