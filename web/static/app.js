// AI Assistant Application Logic
import { AudioRecorder } from "./audio-recorder.js";

let ws;
let audioRecorder;
let isRecording = false;
let currentResponseElement = null;

// Make functions global
window.sendMessage = sendMessage;
window.updateSendInterval = updateSendInterval;

// Logging function
function clientLog(message, type = 'info') {
    const timestamp = new Date().toLocaleTimeString();
    const formattedMessage = `[${timestamp}] [${type}] ${message}`;
    
    // Log to browser console
    if (type === 'error') {
        console.error(formattedMessage);
    } else {
        console.log(formattedMessage);
    }
    
    // Log to UI
    const logElement = document.getElementById('log');
    const logEntry = document.createElement('div');
    logEntry.className = `log-entry ${type}`;
    logEntry.textContent = formattedMessage;
    logElement.appendChild(logEntry);
    logElement.scrollTop = logElement.scrollHeight;
}

// Connect to WebSocket server
function connect() {
    clientLog('Connecting to WebSocket server...');
    
    if (ws) {
        try {
            ws.close();
        } catch (e) {
            // Ignore errors on close
        }
    }
    
    ws = new WebSocket("ws://localhost:8081/ws");
    
    ws.onopen = () => {
        clientLog('WebSocket connection established');
        document.getElementById('status').textContent = "Connected";
        addMessage("Hello! I'm your AI assistant. How can I help you today?", 'ai');
    };

    ws.onmessage = (event) => {
        try {
            clientLog(`Received: ${event.data.substring(0, 100)}${event.data.length > 100 ? '...' : ''}`);
            
            const data = JSON.parse(event.data);
            
            // Handle messages already in your custom format
            if (data.status === "connected to Vertex AI") {
                document.getElementById('vertexStatus').textContent = "Vertex AI: connected";
                document.getElementById('status').textContent = "Ready";
                clientLog('Connected to Vertex AI');
            } 
            else if (data.status === "audio_received") {
                clientLog('Audio chunk received by server');
            } 
            else if (data.status === "streaming" && data.hasOwnProperty('partial')) {
                document.getElementById('status').textContent = "AI is responding...";
                
                if (!currentResponseElement) {
                    currentResponseElement = addMessage(data.partial, 'ai');
                } else {
                    currentResponseElement.textContent = data.partial;
                    const chat = document.getElementById('chat');
                    chat.scrollTop = chat.scrollHeight;
                }
            } 
            else if (data.status === "success" && data.hasOwnProperty('response')) {
                document.getElementById('status').textContent = "Ready";
                
                if (currentResponseElement) {
                    currentResponseElement.textContent = data.response;
                    currentResponseElement = null;
                } else {
                    addMessage(data.response, 'ai');
                }
                
                if (data.audio) {
                    playAudio('data:audio/mp3;base64,' + data.audio);
                }
            } 
            else if (data.status === "fail") {
                clientLog(`Error from server: ${data.message}`, 'error');
                document.getElementById('status').textContent = "Error: " + data.message;
                addMessage(`Error: ${data.message}`, 'ai');
            }
            // Add handling for native Vertex AI message formats:
            else if (data.serverContent && data.serverContent.modelTurn && 
                     data.serverContent.modelTurn.parts && 
                     data.serverContent.modelTurn.parts.length > 0) {
                
                // Handle streaming response from Vertex AI
                document.getElementById('status').textContent = "AI is responding...";
                
                // Extract the text from the response
                const text = data.serverContent.modelTurn.parts[0].text;
                
                if (!currentResponseElement) {
                    currentResponseElement = addMessage(text, 'ai');
                } else {
                    // Append to the existing message for streaming responses
                    currentResponseElement.textContent += text;
                    const chat = document.getElementById('chat');
                    chat.scrollTop = chat.scrollHeight;
                }
            }
            else if (data.serverContent && data.serverContent.turnComplete) {
                // Handle completion message from Vertex AI
                document.getElementById('status').textContent = "Ready";
                currentResponseElement = null;
            }
            else if (data.hasOwnProperty('setupComplete')) {
                // Setup complete message, can be ignored or logged
                clientLog('Vertex AI setup complete');
            }
            else if (data.hasOwnProperty('message') && data.message !== "AI Assistant Ready") {
                addMessage(data.message, 'ai');
            }
            else {
                clientLog(`Unhandled message type: ${JSON.stringify(data)}`);
            }
        } catch (e) {
            clientLog(`Error processing server message: ${e.message}`, 'error');
            document.getElementById('status').textContent = "Error processing response";
        }
    };

    ws.onclose = (event) => {
        clientLog(`WebSocket connection closed: ${event.code} ${event.reason}`, 'error');
        document.getElementById('status').textContent = "Disconnected. Reconnecting...";
        document.getElementById('vertexStatus').textContent = "Vertex AI: disconnected";
        setTimeout(connect, 3000);
    };
    
    ws.onerror = (error) => {
        clientLog(`WebSocket error: ${error}`, 'error');
        document.getElementById('vertexStatus').textContent = "Vertex AI: error";
    };
}

function sendMessage() {
    const input = document.getElementById('textInput');
    const message = input.value.trim();
    
    if (message && ws && ws.readyState === WebSocket.OPEN) {
        clientLog(`Sending text message: ${message}`);
        addMessage(message, 'user');
        input.value = '';
        
        document.getElementById('status').textContent = "Processing...";
        currentResponseElement = null;
        
        try {
            ws.send(JSON.stringify({
                type: "text",
                content: message
            }));
        } catch (e) {
            clientLog(`Error sending message: ${e.message}`, 'error');
            document.getElementById('status').textContent = "Error sending message";
        }
    } else {
        if (!message) {
            clientLog('Cannot send empty message', 'error');
        } else if (!ws || ws.readyState !== WebSocket.OPEN) {
            clientLog('WebSocket not connected', 'error');
            document.getElementById('status').textContent = "Not connected";
            connect();
        }
    }
}

function addMessage(text, sender) {
    const chat = document.getElementById('chat');
    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${sender}`;
    messageDiv.textContent = text;
    
    // Only add timestamp for user messages, not for AI messages
    if (sender === 'user') {
        const timeDiv = document.createElement('div');
        timeDiv.className = 'message-time';
        timeDiv.textContent = new Date().toLocaleTimeString();
        messageDiv.appendChild(timeDiv);
    }
    
    chat.appendChild(messageDiv);
    chat.scrollTop = chat.scrollHeight;
    return messageDiv;
}

function playAudio(audioSrc) {
    try {
        const audio = new Audio(audioSrc);
        audio.play().catch(error => {
            clientLog(`Failed to play audio: ${error}`, 'error');
        });
    } catch (e) {
        clientLog(`Error creating audio element: ${e.message}`, 'error');
    }
}

async function startRecording() {
    if (isRecording) return;
    
    try {
        clientLog('Initializing audio recorder...');
        
        if (!audioRecorder) {
            audioRecorder = new AudioRecorder();
            
            audioRecorder.on("log", (message) => {
                clientLog(`AudioRecorder: ${message}`);
            });
            
            audioRecorder.on("error", (message) => {
                clientLog(`AudioRecorder error: ${message}`, 'error');
            });
            
            audioRecorder.on("data", (base64data) => {
                if (ws && ws.readyState === WebSocket.OPEN) {
                    clientLog(`Sending audio chunk (${base64data.length} chars in base64)`);
                    
                    try {
                        ws.send(JSON.stringify({
                            type: "audio",
                            content: base64data
                        }));
                    } catch (e) {
                        clientLog(`Error sending audio data: ${e.message}`, 'error');
                    }
                } else {
                    clientLog('WebSocket not connected, cannot send audio', 'error');
                }
            });
            
            audioRecorder.on("started", () => {
                document.getElementById('micStatus').textContent = "Mic: active";
            });
            
            audioRecorder.on("stopped", () => {
                document.getElementById('micStatus').textContent = "Mic: inactive";
            });
        }
        
        await audioRecorder.start();
        isRecording = true;
        
        document.getElementById('recordButton').textContent = 'Stop';
        document.getElementById('recordButton').classList.add('recording');
        document.getElementById('status').textContent = "Recording...";
        
        addMessage("🎤 Voice recording started...", 'user');
        currentResponseElement = null;
        
    } catch (error) {
        clientLog(`Error starting recording: ${error}`, 'error');
        document.getElementById('status').textContent = "Error accessing microphone";
    }
}

function stopRecording() {
    if (!isRecording || !audioRecorder) {
        clientLog('Not recording, nothing to stop', 'error');
        return;
    }
    
    clientLog('Stopping recording');
    audioRecorder.stop();
    isRecording = false;
    
    if (ws && ws.readyState === WebSocket.OPEN) {
        clientLog('Sending end-of-stream signal');
        try {
            ws.send(JSON.stringify({
                type: "audio_end",
                content: ""
            }));
        } catch (e) {
            clientLog(`Error sending end-of-stream signal: ${e.message}`, 'error');
        }
    } else {
        clientLog('WebSocket not connected, cannot send end-of-stream', 'error');
    }
    
    document.getElementById('recordButton').textContent = 'Record';
    document.getElementById('recordButton').classList.remove('recording');
    document.getElementById('status').textContent = "Processing voice...";
}

function updateSendInterval(value) {
    if (audioRecorder) {
        audioRecorder.setSendInterval(parseInt(value));
        clientLog(`Audio send interval updated to ${value}ms`);
    }
}

// Event listeners
document.getElementById('textInput').addEventListener('keydown', function(event) {
    if (event.key === 'Enter') {
        event.preventDefault();
        sendMessage();
    }
});

document.getElementById('recordButton').addEventListener('click', function() {
    if (isRecording) {
        stopRecording();
    } else {
        startRecording();
    }
});

document.getElementById('sendButton').addEventListener('click', sendMessage);

// Start connection when page loads
window.addEventListener('load', function() {
    clientLog('Page loaded, connecting to server...');
    connect();
});
