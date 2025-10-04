package gateway

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"golang.org/x/oauth2/google"
	"websocket-ai/internal/models"
)

const (
	HOST        = "us-central1-aiplatform.googleapis.com"
	SERVICE_URL = "wss://" + HOST + "/ws/google.cloud.aiplatform.v1beta1.LlmBidiService/BidiGenerateContent"
	TTS_URL     = "https://texttospeech.googleapis.com/v1/text:synthesize"
)

func GetAccessToken() (string, error) {
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

func TextToSpeech(text, language string) (string, error) {
	token, err := GetAccessToken()
	if err != nil {
		return "", fmt.Errorf("error getting access token for TTS: %v", err)
	}

	// Create TTS request
	ttsReq := models.TTSRequest{}
	ttsReq.Input.Text = text
	ttsReq.Voice.LanguageCode = language

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
	var ttsResp models.TTSResponse
	if err := json.Unmarshal(body, &ttsResp); err != nil {
		return "", fmt.Errorf("error unmarshaling TTS response: %v", err)
	}

	return ttsResp.AudioContent, nil
}

func SetupVertexAI() (*websocket.Conn, error) {
	token, err := GetAccessToken()
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

func IsSetupComplete(jsonStr string) bool {
	return strings.Contains(jsonStr, `"setupComplete": {}`)
}
