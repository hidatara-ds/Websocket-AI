package gateway

import (
	"sync"
	"time"
)

// Metrics holds application metrics
type Metrics struct {
	mu                  sync.RWMutex
	TotalConnections    int64
	ActiveConnections   int64
	TotalMessages       int64
	TotalErrors         int64
	AverageResponseTime time.Duration
	StartTime           time.Time
	LastActivity        time.Time
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		StartTime: time.Now(),
	}
}

// IncrementConnections increments total connections
func (m *Metrics) IncrementConnections() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalConnections++
	m.ActiveConnections++
	m.LastActivity = time.Now()
}

// DecrementConnections decrements active connections
func (m *Metrics) DecrementConnections() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ActiveConnections > 0 {
		m.ActiveConnections--
	}
}

// IncrementMessages increments total messages
func (m *Metrics) IncrementMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalMessages++
	m.LastActivity = time.Now()
}

// IncrementErrors increments total errors
func (m *Metrics) IncrementErrors() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalErrors++
}

// UpdateResponseTime updates average response time
func (m *Metrics) UpdateResponseTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.AverageResponseTime == 0 {
		m.AverageResponseTime = duration
	} else {
		// Simple moving average
		m.AverageResponseTime = (m.AverageResponseTime + duration) / 2
	}
}

// GetStats returns current metrics
func (m *Metrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uptime := time.Since(m.StartTime)

	return map[string]interface{}{
		"uptime":                uptime.String(),
		"total_connections":     m.TotalConnections,
		"active_connections":    m.ActiveConnections,
		"total_messages":        m.TotalMessages,
		"total_errors":          m.TotalErrors,
		"average_response_time": m.AverageResponseTime.String(),
		"last_activity":         m.LastActivity.Format(time.RFC3339),
		"uptime_seconds":        int64(uptime.Seconds()),
	}
}

// Reset resets all metrics
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalConnections = 0
	m.ActiveConnections = 0
	m.TotalMessages = 0
	m.TotalErrors = 0
	m.AverageResponseTime = 0
	m.StartTime = time.Now()
	m.LastActivity = time.Now()
}

// HealthCheck represents application health status
type HealthCheck struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Uptime    string                 `json:"uptime"`
	Metrics   map[string]interface{} `json:"metrics"`
	Services  map[string]string      `json:"services"`
}

// GetHealthCheck returns current health status
func (m *Metrics) GetHealthCheck() *HealthCheck {
	stats := m.GetStats()

	return &HealthCheck{
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    stats["uptime"].(string),
		Metrics:   stats,
		Services: map[string]string{
			"websocket": "operational",
			"vertex_ai": "operational",
			"tts":       "operational",
		},
	}
}
