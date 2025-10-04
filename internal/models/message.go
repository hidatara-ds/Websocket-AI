package models

// Message represents a WebSocket message from client
type Message struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// Response represents Vertex AI response structure
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

// AuthToken represents Google OAuth token
type AuthToken struct {
	AccessToken string `json:"access_token"`
}

// TTSRequest represents Text-to-Speech request
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

// TTSResponse represents Text-to-Speech response
type TTSResponse struct {
	AudioContent string `json:"audioContent"`
}
