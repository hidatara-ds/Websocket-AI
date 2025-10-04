# 📁 Struktur Proyek Baru

## 🎯 Perubahan yang Dilakukan

### ✅ **Sebelum (Struktur Lama)**
```
websocket-ai/
├── proxy/                   # ❌ Nama membingungkan
│   └── proxy.go            # ❌ Tidak jelas fungsinya
├── server/                  # ❌ Nama membingungkan  
│   └── server.go           # ❌ Tidak jelas fungsinya
├── templates/               # ❌ Struktur tidak terorganisir
└── README.md
```

### ✅ **Sesudah (Struktur Baru)**
```
websocket-ai/
├── cmd/                     # ✅ Entry points yang jelas
│   ├── ai-gateway/         # ✅ WebSocket proxy ke Vertex AI
│   │   └── main.go
│   └── static-server/      # ✅ Static file server
│       └── main.go
├── internal/               # ✅ Internal packages
│   ├── gateway/            # ✅ AI gateway logic
│   │   ├── websocket.go
│   │   └── vertex_ai.go
│   ├── server/             # ✅ Static server logic
│   │   └── static.go
│   └── models/             # ✅ Data structures
│       └── message.go
├── web/                    # ✅ Frontend files
│   ├── templates/
│   ├── static/
│   └── assets/
├── run.bat                 # ✅ Windows runner
├── run.sh                  # ✅ Linux/Mac runner
├── go.mod
├── go.sum
└── README.md
```

## 🚀 Cara Menjalankan

### **Windows:**
```bash
# Double-click run.bat atau:
run.bat
```

### **Linux/Mac:**
```bash
./run.sh
```

### **Manual:**
```bash
# Terminal 1 - Static File Server
cd cmd/static-server
go run main.go

# Terminal 2 - AI Gateway  
cd cmd/ai-gateway
go run main.go
```

## 📋 Keuntungan Struktur Baru

1. **✅ Penamaan Jelas**: 
   - `ai-gateway` vs `proxy` - lebih jelas fungsinya
   - `static-server` vs `server` - lebih jelas fungsinya

2. **✅ Organisasi Lebih Baik**:
   - `cmd/` untuk entry points
   - `internal/` untuk business logic
   - `web/` untuk frontend files

3. **✅ Separation of Concerns**:
   - Gateway logic terpisah dari main
   - Models terpisah dari business logic
   - Static server logic terpisah

4. **✅ Mudah Dikembangkan**:
   - Struktur mengikuti Go best practices
   - Import paths yang jelas
   - Package dependencies yang terorganisir

## 🔧 File yang Diubah

| File Lama | File Baru | Alasan |
|-----------|-----------|---------|
| `proxy/proxy.go` | `cmd/ai-gateway/main.go` + `internal/gateway/` | Pisahkan main dari business logic |
| `server/server.go` | `cmd/static-server/main.go` + `internal/server/` | Pisahkan main dari business logic |
| `templates/` | `web/templates/` | Lebih jelas sebagai web assets |

## 📝 Import Paths

**Sebelum:**
```go
import "../models"
import "../../internal/gateway"
```

**Sesudah:**
```go
import "websocket-ai/internal/models"
import "websocket-ai/internal/gateway"
```
