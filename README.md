 # 🤖 WebSocket AI Assistant

Aplikasi Go yang menyediakan interface WebSocket untuk berkomunikasi dengan Google Vertex AI. Aplikasi ini mendukung input teks dan audio real-time dengan Text-to-Speech (TTS) response.

## 🏗️ Arsitektur

Aplikasi menggunakan **Proxy Pattern** dengan elemen **MVC (Model-View-Controller)**:

- **Model**: WebSocket proxy ke Google Vertex AI
- **View**: Interface web dengan HTML/CSS/JavaScript
- **Controller**: WebSocket handlers yang mengatur komunikasi

## ✨ Fitur

- **Real-time WebSocket Communication** dengan Google Vertex AI
- **Audio Input Support** - Recording dan streaming audio ke AI
- **Text-to-Speech Response** - AI response dalam format audio
- **Multi-language Support** - Bahasa Inggris dan Indonesia
- **Stock API Integration** - Integrasi dengan Finnhub API
- **Weather API Integration** - Integrasi dengan OpenWeatherMap API
- **Modern Web Interface** - Responsive dan user-friendly

## 📁 Struktur Proyek

```
websocket-ai/
├── proxy/                   # WebSocket proxy ke Vertex AI
│   └── proxy.go            # Main proxy server (Port 8081)
├── server/                  # Static file server
│   └── server.go              # File server (Port 8080)
├── templates/               # Frontend files
│   ├── index.html          # Main interface
│   ├── assets/             # Images dan assets
│   └── static/             # JavaScript modules
│       ├── audio-recorder.js
│       ├── tools/
│       │   ├── stock-api.js
│       │   └── weather-api.js
│       └── utils.js
├── go.mod
├── go.sum
└── README.md
```

## 🚀 Persyaratan

- **Go 1.24+** (sesuai dengan go.mod)
- **Google Cloud Platform Account** dengan Vertex AI API enabled
- **Service Account** dengan credentials untuk Vertex AI
- **Browser modern** dengan WebSocket dan Web Audio API support

## ⚙️ Instalasi

1. **Clone repository**
   ```bash
   git clone <repository-url>
   cd websocket-ai
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Setup Google Cloud credentials**
   ```bash
   # Set environment variable untuk service account
   export GOOGLE_APPLICATION_CREDENTIALS="path/to/service-account.json"
   ```

4. **Jalankan aplikasi**
   
   **Terminal 1 - Static File Server:**
   ```bash
   cd server
   go run server.go
   ```
   
   **Terminal 2 - WebSocket Proxy:**
   ```bash
   cd proxy
   go run proxy.go
   ```

5. **Akses aplikasi**
   - Buka browser ke `http://localhost:8080`
   - WebSocket akan otomatis connect ke `ws://localhost:8081/ws`

## 🔧 Konfigurasi

### Google Cloud Setup
1. Buat project di Google Cloud Console
2. Enable Vertex AI API
3. Buat service account dengan role "Vertex AI User"
4. Download JSON credentials
5. Set environment variable `GOOGLE_APPLICATION_CREDENTIALS`

### API Keys (Opsional)
Untuk fitur tambahan, set API keys di file JavaScript:

**Stock API (Finnhub):**
```javascript
// templates/static/tools/stock-api.js
const FINNHUB_API_KEY = 'your-finnhub-api-key';
```

**Weather API (OpenWeatherMap):**
```javascript
// templates/static/tools/weather-api.js
const OPENWEATHER_API_KEY = 'your-openweather-api-key';
```

## 🎯 Penggunaan

1. **Text Chat**: Ketik pesan dan tekan Enter atau klik Send
2. **Voice Chat**: Klik Record untuk mulai recording, klik Stop untuk mengirim
3. **Real-time Response**: AI akan merespons dengan teks dan audio
4. **Multi-modal**: Support untuk text, audio, dan image input

## 🔌 API Endpoints

- `GET /` - Main interface
- `WS /ws` - WebSocket endpoint untuk AI communication
- `GET /static/*` - Static files (CSS, JS, images)

## 📝 WebSocket Protocol

### Client → Server
```json
{
  "type": "text|audio|audio_end|image",
  "content": "data atau text"
}
```

### Server → Client
```json
{
  "status": "success|streaming|fail",
  "code": 200,
  "response": "AI response text",
  "audio": "base64-encoded-audio",
  "partial": "streaming text"
}
```

## 🛠️ Development

### Struktur yang Disarankan untuk Refactoring
```
websocket-ai/
├── cmd/
│   ├── ai-gateway/          # WebSocket proxy ke Vertex AI
│   └── static-server/      # Static file server
├── internal/
│   ├── gateway/             # AI gateway logic
│   ├── server/              # Static server logic
│   └── models/              # Data structures
├── web/                     # Frontend files
└── config/                  # Configuration files
```

## 📄 License

Apache License 2.0 - Lihat file LICENSE untuk detail lengkap.

## 🤝 Contributing

1. Fork repository
2. Buat feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push ke branch (`git push origin feature/amazing-feature`)
5. Buat Pull Request
