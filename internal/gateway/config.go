package gateway

import (
	"os"
	"strconv"
	"time"
)

// Config holds application configuration
type Config struct {
	Port               string
	Host               string
	ServiceURL         string
	TTSURL             string
	MaxConnections     int
	ConnectionTimeout  time.Duration
	CleanupInterval    time.Duration
	EnableCORS         bool
	EnableLogging      bool
	LogLevel           string
	ModelName          string
	Temperature        float64
	TopP               float64
	TopK               int
	MaxTokens          int
	ResponseModalities []string
	SystemInstruction  string
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	return &Config{
		Port:               getEnv("PORT", "8081"),
		Host:               getEnv("HOST", "127.0.0.1"),
		ServiceURL:         getEnv("SERVICE_URL", "wss://us-central1-aiplatform.googleapis.com/ws/google.cloud.aiplatform.v1beta1.LlmBidiService/BidiGenerateContent"),
		TTSURL:             getEnv("TTS_URL", "https://texttospeech.googleapis.com/v1/text:synthesize"),
		MaxConnections:     getEnvInt("MAX_CONNECTIONS", 100),
		ConnectionTimeout:  getEnvDuration("CONNECTION_TIMEOUT", 30*time.Second),
		CleanupInterval:    getEnvDuration("CLEANUP_INTERVAL", 30*time.Second),
		EnableCORS:         getEnvBool("ENABLE_CORS", true),
		EnableLogging:      getEnvBool("ENABLE_LOGGING", true),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		ModelName:          getEnv("MODEL_NAME", "projects/our-service-454404-j3/locations/us-central1/publishers/google/models/gemini-2.0-flash-exp"),
		Temperature:        getEnvFloat("TEMPERATURE", 0.7),
		TopP:               getEnvFloat("TOP_P", 0.95),
		TopK:               getEnvInt("TOP_K", 40),
		MaxTokens:          getEnvInt("MAX_TOKENS", 2048),
		ResponseModalities: []string{"TEXT"},
		SystemInstruction:  getEnv("SYSTEM_INSTRUCTION", "You are a helpful, friendly AI assistant. You can understand both text and voice inputs in multiple languages including English and Indonesian. Respond in a conversational, helpful manner. Keep your responses concise and engaging."),
	}
}

// Helper functions for environment variable parsing
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
