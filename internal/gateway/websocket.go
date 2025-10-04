package gateway

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"websocket-ai/internal/models"
)

var (
	ActiveConnections sync.Map
	Upgrader          = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins (adjust as needed for security)
		},
	}
)

func ProxyMessagesClient(src, dest *websocket.Conn, name string, wg *sync.WaitGroup) {
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
		var requestMessage models.Message
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

func ProxyMessagesServer(src, dest *websocket.Conn, name string, wg *sync.WaitGroup) {
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

		var response models.Response
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
		if IsSetupComplete(string(message)) {
			responseMessage = `{"status": "connected to Vertex AI", "code": 200, "message": "AI Assistant Ready"}`
		} else if response.ServerContent.TurnComplete {
			// Generate speech from AI response
			audioContent, err := TextToSpeech(partMessage, "en-US")

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

func HandleClient(clientConn *websocket.Conn) {
	log.Println("New client connected")

	serverConn, err := SetupVertexAI()
	if err != nil {
		log.Println("Failed to setup Vertex AI connection:", err)
		clientConn.Close()
		return
	}

	defer serverConn.Close()

	ActiveConnections.Store(clientConn, true)

	var count int
	ActiveConnections.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	log.Println("active connection:", count)

	var wg sync.WaitGroup
	wg.Add(2)
	go ProxyMessagesClient(clientConn, serverConn, "Client->Server", &wg)
	go ProxyMessagesServer(serverConn, clientConn, "Server->Client", &wg)
	wg.Wait()

	ActiveConnections.Delete(clientConn)
}

func CleanupConnections() {
	for {
		time.Sleep(30 * time.Second)

		var count int
		ActiveConnections.Range(func(key, value interface{}) bool {
			count++
			return true
		})
		log.Println("Checking active connections:", count)

		ActiveConnections.Range(func(key, value interface{}) bool {
			conn := key.(*websocket.Conn)
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("Found stale connection, closing...")
				conn.Close()
				ActiveConnections.Delete(conn)
			}
			return true
		})
	}
}
