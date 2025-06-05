package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/oauth2/google"
)

const (
	HOST        = "us-central1-aiplatform.googleapis.com"
	SERVICE_URL = "wss://" + HOST + "/ws/google.cloud.aiplatform.v1beta1.LlmBidiService/BidiGenerateContent"
	PORT        = "8081"
	// Google Text-to-Speech API endpoint
	TTS_URL = "https://texttospeech.googleapis.com/v1/text:synthesize"
)

var (
	activeConnections sync.Map
	upgrader          = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins (adjust as needed for security)
		},
	}
)

type AuthToken struct {
	AccessToken string `json:"access_token"`
}

// Response struct for parsing Vertex AI responses
type Response struct {
	ServerContent struct {
		TurnComplete bool `json:"turn_complete"`
		ModelTurn    struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"model_turn"`
	} `json:"server_content"`
	SetupComplete struct{} `json:"setupComplete"`
}

// Text-to-Speech request struct
type TTSRequest struct {
	Input struct {
		Text string `json:"text"`
	} `json:"input"`
	Voice struct {
		LanguageCode string `json:"languageCode"`
		Name         string `json:"name"`
	} `json:"voice"`
	AudioConfig struct {
		AudioEncoding string `json:"audioEncoding"`
	} `json:"audioConfig"`
}

// Text-to-Speech response struct
type TTSResponse struct {
	AudioContent string `json:"audioContent"`
}

func getAccessToken() (string, error) {
	scopes := []string{
		"https://www.googleapis.com/auth/cloud-platform",
	}

	ctx := context.Background()
	creds, err := google.FindDefaultCredentials(ctx, scopes...)
	if err != nil {
		return "", fmt.Errorf("error getting credentials: %v", err)
	}
	tokenSource := creds.TokenSource
	token, err := tokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("error retrieving access token: %v", err)
	}
	return token.AccessToken, nil
}

func textToSpeech(text, language string) (string, error) {
	token, err := getAccessToken()
	if err != nil {
		return "", fmt.Errorf("error getting access token for TTS: %v", err)
	}

	// Create TTS request
	ttsReq := TTSRequest{}
	ttsReq.Input.Text = text
	ttsReq.Voice.LanguageCode = language // "en-US" for English

	// Choose appropriate voice based on language
	if language == "en-US" {
		ttsReq.Voice.Name = "en-US-Wavenet-D" // Male voice
	} else if language == "id-ID" {
		ttsReq.Voice.Name = "id-ID-Wavenet-A" // Indonesian voice
	}

	ttsReq.AudioConfig.AudioEncoding = "MP3"

	// Convert to JSON
	jsonData, err := json.Marshal(ttsReq)
	if err != nil {
		return "", fmt.Errorf("error marshaling TTS request: %v", err)
	}

	// Create HTTP request
	client := &http.Client{}
	req, err := http.NewRequest("POST", TTS_URL, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", fmt.Errorf("error creating TTS request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending TTS request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading TTS response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("TTS API error: %s", string(body))
	}

	// Parse response
	var ttsResp TTSResponse
	if err := json.Unmarshal(body, &ttsResp); err != nil {
		return "", fmt.Errorf("error unmarshaling TTS response: %v", err)
	}

	return ttsResp.AudioContent, nil
}

func proxyMessagesClient(src, dest *websocket.Conn, name string, wg *sync.WaitGroup) {
	defer wg.Done()
	defer src.Close()

	for {
		messageType, message, err := src.ReadMessage()
		if err != nil {
			log.Printf("%s connection closed: %v", name, err)
			responseMessage := fmt.Sprintf(`{"status": "fail connect to websocket", "code": 500, "message": "%v"}`, err)
			if err := dest.WriteMessage(websocket.TextMessage, []byte(responseMessage)); err != nil {
				log.Printf("%s error sending message: %v", name, err)
			}
			return
		}

		// Handle binary message (for audio input)
		if messageType == websocket.BinaryMessage {
			// Convert binary data to base64 string
			base64Data := base64.StdEncoding.EncodeToString(message)

			// Format for real-time audio streaming to Vertex AI
			responseMessage := fmt.Sprintf(`{
				"realtimeInput": {
					"mediaChunks": [{
						"mime_type": "audio/webm",
						"data": "%s"
					}]
				}
			}`, base64Data)

			if err := dest.WriteMessage(websocket.TextMessage, []byte(responseMessage)); err != nil {
				log.Printf("%s error sending audio message: %v", name, err)
				return
			}

			// Send confirmation back to client
			audioConfirmation := `{"status": "audio_received", "code": 200}`
			if err := src.WriteMessage(websocket.TextMessage, []byte(audioConfirmation)); err != nil {
				log.Printf("%s error sending audio confirmation: %v", name, err)
			}

			continue
		}

		log.Printf("Raw client message: %s", string(message))

		// Try to parse as JSON object first
		var jsonData map[string]interface{}
		if err := json.Unmarshal(message, &jsonData); err == nil {
			// Successfully parsed as JSON object
			if textContent, ok := jsonData["text"].(string); ok {
				// Handle text message directly from JSON
				log.Printf("Parsed text from JSON: %s", textContent)

				responseMessage := fmt.Sprintf(`{
					"client_content": {
						"turns": [{
							"role": "user",
							"parts": [{ "text": %s }]
						}],
						"turn_complete": true
					}
				}`, string(json.RawMessage(fmt.Sprintf(`"%s"`, textContent))))

				log.Printf("Sending to Vertex AI: %s", responseMessage)
				if err := dest.WriteMessage(websocket.TextMessage, []byte(responseMessage)); err != nil {
					log.Printf("%s error sending message: %v", name, err)
					return
				}
				continue
			}
		}

		// If not a direct JSON object, try the Message struct
		var requestMessage Message
		if err := json.Unmarshal(message, &requestMessage); err != nil {
			log.Printf("Error parsing JSON as Message: %v", err)
			continue // Skip this message but keep connection alive
		}

		if requestMessage.Type == "" || requestMessage.Content == "" {
			log.Printf("Invalid message format: %+v", requestMessage)
			continue
		}

		log.Printf("Parsed message: %+v", requestMessage)

		responseMessage := ""

		if requestMessage.Type == "text" {
			// Fix the JSON formatting and ensure proper escaping of user content
			responseMessage = fmt.Sprintf(`{
				"client_content": {
					"turns": [{
						"role": "user",
						"parts": [{ "text": %s }]
					}],
					"turn_complete": true
				}
			}`, string(json.RawMessage(fmt.Sprintf(`"%s"`, requestMessage.Content))))
		} else if requestMessage.Type == "audio" {
			// Format for real-time audio streaming
			responseMessage = fmt.Sprintf(`{
				"realtimeInput": {
                  "mediaChunks": [{
                    "mime_type": "audio/webm",
                    "data": "%s"
                  }]
                }
			}`, requestMessage.Content)

			// Send confirmation back to client
			audioConfirmation := `{"status": "audio_received", "code": 200}`
			if err := src.WriteMessage(websocket.TextMessage, []byte(audioConfirmation)); err != nil {
				log.Printf("%s error sending audio confirmation: %v", name, err)
			}
		} else if requestMessage.Type == "audio_end" {
			// Signal end of audio stream
			responseMessage = `{
				"realtimeInput": {
					"endOfStream": true
				}
			}`
		} else if requestMessage.Type == "image" {
			responseMessage = fmt.Sprintf(`{
				"realtimeInput": {
                  "mediaChunks": [{
                    "mime_type": "image/jpeg",
                    "data": "%s"
                  }]
                }
			}`, requestMessage.Content)
		}

		log.Printf("Sending to Vertex AI: %s", responseMessage)

		if responseMessage != "" {
			if err := dest.WriteMessage(websocket.TextMessage, []byte(responseMessage)); err != nil {
				log.Printf("%s error sending message: %v", name, err)
				return
			}
		}
	}
}

type Message struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

func proxyMessagesServer(src, dest *websocket.Conn, name string, wg *sync.WaitGroup) {
	defer wg.Done()
	defer src.Close()

	partMessage := ""

	for {
		_, message, err := src.ReadMessage()
		if err != nil {
			log.Printf("%s connection closed: %v", name, err)
			responseMessage := fmt.Sprintf(`{"status": "fail", "code": 500, "message": "Connection to AI service lost: %v"}`, err)
			if err := dest.WriteMessage(websocket.TextMessage, []byte(responseMessage)); err != nil {
				log.Printf("%s error sending message: %v", name, err)
			}
			return
		}

		log.Printf("Received from Vertex AI: %s", string(message))

		var response Response
		err = json.Unmarshal(message, &response)
		if err != nil {
			log.Printf("Failed to decode JSON: %v", err)
			log.Printf("Raw message: %s", string(message))
			// Send error to client
			errorMsg := fmt.Sprintf(`{"status": "fail", "code": 500, "message": "Error processing AI response"}`)
			dest.WriteMessage(websocket.TextMessage, []byte(errorMsg))
			continue // Continue instead of returning to keep the connection alive
		}

		responseMessage := ""
		if isSetupComplete(string(message)) {
			responseMessage = `{"status": "connected to Vertex AI", "code": 200, "message": "AI Assistant Ready"}`
		} else if response.ServerContent.TurnComplete {
			// Generate speech from AI response
			audioContent, err := textToSpeech(partMessage, "en-US")

			if err != nil {
				log.Printf("Error generating speech: %v", err)
				// Send just the text response if TTS fails
				// Create a proper JSON object instead of string formatting
				responseObj := map[string]interface{}{
					"status":   "success",
					"code":     200,
					"response": partMessage,
				}
				jsonData, err := json.Marshal(responseObj)
				if err != nil {
					log.Printf("Error marshaling JSON: %v", err)
					responseMessage = fmt.Sprintf(`{"status": "fail", "code": 500, "message": "Error formatting response"}`)
				} else {
					responseMessage = string(jsonData)
				}
			} else {
				// Send both text and audio
				// Create a proper JSON object instead of string formatting
				responseObj := map[string]interface{}{
					"status":   "success",
					"code":     200,
					"response": partMessage,
					"audio":    audioContent,
				}
				jsonData, err := json.Marshal(responseObj)
				if err != nil {
					log.Printf("Error marshaling JSON: %v", err)
					responseMessage = fmt.Sprintf(`{"status": "fail", "code": 500, "message": "Error formatting response with audio"}`)
				} else {
					responseMessage = string(jsonData)
				}
			}

			partMessage = ""
		} else if len(response.ServerContent.ModelTurn.Parts) > 0 {
			// Process all parts in the response
			for _, part := range response.ServerContent.ModelTurn.Parts {
				if part.Text != "" {
					partMessage += part.Text
					log.Printf("Received text part: %s", part.Text)
				}
			}

			// Always send streaming updates when we receive parts
			// Create a proper JSON object instead of string formatting
			streamingObj := map[string]interface{}{
				"status":  "streaming",
				"code":    200,
				"partial": partMessage,
			}
			jsonData, err := json.Marshal(streamingObj)
			if err != nil {
				log.Printf("Error marshaling streaming JSON: %v", err)
				continue
			}

			streamingResponse := string(jsonData)
			log.Printf("Sending streaming update: %s", streamingResponse)
			if err := dest.WriteMessage(websocket.TextMessage, []byte(streamingResponse)); err != nil {
				log.Printf("%s error sending streaming message: %v", name, err)
				return
			}
		} else if strings.Contains(string(message), `"generationComplete": true`) {
			// Handle generationComplete message - just log it
			log.Printf("Generation complete received")
			// No need to send anything to client for this message type
		} else {
			// For any other message types, just pass through as-is
			responseMessage = string(message)
			log.Printf("Passing through message: %s", responseMessage)
		}

		if responseMessage != "" {
			log.Printf("Sending to client: %s", responseMessage)
			if err := dest.WriteMessage(websocket.TextMessage, []byte(responseMessage)); err != nil {
				log.Printf("%s error sending message: %v", name, err)
				return
			}
		}
	}
}

func setupVertexAI() (*websocket.Conn, error) {
	token, err := getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("Error getting access token: %v", err)
	}
	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)

	serverConn, _, err := dialer.Dial(SERVICE_URL, headers)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to Vertex AI WebSocket: %v", err)
	}

	// Updated setup payload for a conversational AI chatbot
	setupPayloadVertex := `{
		"setup": {
			"model": "projects/our-service-454404-j3/locations/us-central1/publishers/google/models/gemini-2.0-flash-exp", 
			"generationConfig": {
				"responseModalities": ["TEXT"],
				"temperature": 0.7,
				"topP": 0.95,
				"topK": 40
			},
			"system_instruction": {
				"role": "system",
				"parts": [{
					"text": "You are a helpful, friendly AI assistant. You can understand both text and voice inputs in multiple languages including English and Indonesian. Respond in a conversational, helpful manner. Keep your responses concise and engaging."
				}]
			}
		}
	}`

	serverConn.WriteMessage(websocket.TextMessage, []byte(setupPayloadVertex))

	return serverConn, nil
}

func isSetupComplete(jsonStr string) bool {
	return strings.Contains(jsonStr, `"setupComplete": {}`)
}

func handleClient(clientConn *websocket.Conn) {
	log.Println("New client connected")

	serverConn, err := setupVertexAI()
	if err != nil {
		log.Println("Failed to setup Vertex AI connection:", err)
		clientConn.Close()
		return
	}

	defer serverConn.Close()

	activeConnections.Store(clientConn, true)

	var count int
	activeConnections.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	log.Println("active connection:", count)

	var wg sync.WaitGroup
	wg.Add(2)
	go proxyMessagesClient(clientConn, serverConn, "Client->Server", &wg)
	go proxyMessagesServer(serverConn, clientConn, "Server->Client", &wg)
	wg.Wait()

	activeConnections.Delete(clientConn)
}

func cleanupConnections() {
	for {
		time.Sleep(30 * time.Second)

		var count int
		activeConnections.Range(func(key, value interface{}) bool {
			count++
			return true
		})
		log.Println("Checking active connections:", count)

		activeConnections.Range(func(key, value interface{}) bool {
			conn := key.(*websocket.Conn)
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("Found stale connection, closing...")
				conn.Close()
				activeConnections.Delete(conn)
			}
			return true
		})
	}
}

func main() {
	log.Println("Starting WebSocket proxy server on port", PORT)

	// Serve static files for the web interface
	http.Handle("/", http.FileServer(http.Dir("templates")))

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Failed to upgrade connection:", err)
			return
		}
		handleClient(conn)
	})

	go cleanupConnections()

	if err := http.ListenAndServe("127.0.0.1:"+PORT, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
