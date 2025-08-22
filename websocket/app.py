from flask import Flask, send_from_directory
from flask_sock import Sock
import json
import time
import threading
import traceback
import logging
from werkzeug.serving import WSGIRequestHandler

# Disable excessive logging
logging.getLogger('werkzeug').setLevel(logging.WARNING)
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

app = Flask(__name__)
sock = Sock(app)

# Store active connections with metadata
active_connections = {}

class ConnectionManager:
    def __init__(self):
        self.connections = {}
        self.message_count = 0
    
    def add_connection(self, ws, connection_id):
        self.connections[connection_id] = {
            'ws': ws,
            'connected_at': time.time(),
            'last_ping': time.time(),
            'message_count': 0
        }
        logger.info(f"➕ Connection {connection_id} added. Total: {len(self.connections)}")
    
    def remove_connection(self, connection_id):
        if connection_id in self.connections:
            conn_info = self.connections[connection_id]
            duration = time.time() - conn_info['connected_at']
            del self.connections[connection_id]
            logger.info(f"➖ Connection {connection_id} removed (lived {duration:.1f}s). Total: {len(self.connections)}")
    
    def get_connection_info(self):
        return {
            'total_connections': len(self.connections),
            'connections': {
                cid: {
                    'connected_at': info['connected_at'],
                    'duration': time.time() - info['connected_at'],
                    'message_count': info['message_count']
                } for cid, info in self.connections.items()
            }
        }

connection_manager = ConnectionManager()

@app.route("/")
def index():
    return send_from_directory(".", "index.html")

@app.route("/status")
def status():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "server_time": int(time.time()),
        **connection_manager.get_connection_info()
    }

def safe_send(ws, message, connection_id):
    """Safely send a message to WebSocket"""
    try:
        ws.send(json.dumps(message))
        return True
    except Exception as e:
        logger.error(f"Failed to send message to {connection_id}: {e}")
        return False

@sock.route("/ws")
def ws_handler(ws):
    connection_id = f"conn_{int(time.time() * 1000)}_{id(ws) % 10000}"
    logger.info(f"🔌 NEW CONNECTION: {connection_id}")
    
    try:
        # Add to connection manager
        connection_manager.add_connection(ws, connection_id)
        
        # Send immediate welcome message
        welcome_msg = {
            "type": "system_ready",
            "message": "🤖 Test connection successful!",
            "connection_id": connection_id,
            "server_time": int(time.time())
        }
        
        if not safe_send(ws, welcome_msg, connection_id):
            logger.error(f"Failed to send welcome message to {connection_id}")
            return
        
        logger.info(f"✅ Welcome sent to {connection_id}")
        
        # Main message loop - NO TIMEOUTS!
        while True:
            try:
                # Blocking receive - let the client control the flow
                raw_message = ws.receive()
                
                if raw_message is None:
                    logger.info(f"📪 Connection {connection_id} closed by client (clean)")
                    break
                
                # Update connection stats
                if connection_id in connection_manager.connections:
                    connection_manager.connections[connection_id]['message_count'] += 1
                    connection_manager.connections[connection_id]['last_ping'] = time.time()
                
                # Parse and handle message
                try:
                    message = json.loads(raw_message)
                    msg_type = message.get('type', 'unknown')
                    
                    logger.info(f"📨 {connection_id}: {msg_type}")
                    
                    # Handle different message types
                    response = None
                    
                    if msg_type == "ping":
                        response = {
                            "type": "pong",
                            "timestamp": int(time.time()),
                            "original_timestamp": message.get('timestamp'),
                            "server_connection_time": time.time() - connection_manager.connections[connection_id]['connected_at']
                        }
                        
                    elif msg_type == "test":
                        response = {
                            "type": "test_response",
                            "message": "✅ Test successful!",
                            "echo_data": message.get('data', ''),
                            "server_time": int(time.time()),
                            "connection_stats": {
                                "id": connection_id,
                                "messages_received": connection_manager.connections[connection_id]['message_count'],
                                "uptime": time.time() - connection_manager.connections[connection_id]['connected_at']
                            }
                        }
                        
                    elif msg_type == "heartbeat":
                        response = {
                            "type": "heartbeat_ack",
                            "timestamp": int(time.time()),
                            "connection_uptime": time.time() - connection_manager.connections[connection_id]['connected_at']
                        }
                        # Don't log every heartbeat to avoid spam
                        logger.debug(f"💓 Heartbeat from {connection_id}")
                        
                    elif msg_type == "audio_stream":
                        # Handle audio data (for your main app)
                        audio_size = len(message.get('data', ''))
                        response = {
                            "type": "audio_received",
                            "message": f"🎵 Audio chunk received ({audio_size} bytes)",
                            "size": audio_size,
                            "timestamp": int(time.time())
                        }
                        
                    else:
                        # Echo unknown messages
                        response = {
                            "type": "echo",
                            "original_type": msg_type,
                            "message": f"📡 Echo: {msg_type}",
                            "original_message": message,
                            "timestamp": int(time.time())
                        }
                    
                    # Send response
                    if response:
                        if not safe_send(ws, response, connection_id):
                            logger.error(f"Failed to send response to {connection_id}")
                            break
                        
                        if msg_type == "ping":
                            logger.debug(f"🏓 Pong sent to {connection_id}")
                        elif msg_type == "heartbeat":
                            # Don't spam logs with heartbeat responses
                            pass
                        else:
                            logger.info(f"📤 Response sent to {connection_id}: {response['type']}")
                            
                except json.JSONDecodeError as e:
                    logger.warning(f"❌ Invalid JSON from {connection_id}: {e}")
                    error_response = {
                        "type": "error",
                        "message": "Invalid JSON format",
                        "error": str(e),
                        "timestamp": int(time.time())
                    }
                    if not safe_send(ws, error_response, connection_id):
                        break
                        
                except Exception as e:
                    logger.error(f"❌ Message processing error for {connection_id}: {e}")
                    logger.debug(traceback.format_exc())
                    
                    error_response = {
                        "type": "error", 
                        "message": "Server processing error",
                        "timestamp": int(time.time())
                    }
                    if not safe_send(ws, error_response, connection_id):
                        break
                    
            except Exception as e:
                error_str = str(e).lower()
                if any(term in error_str for term in ['closed', 'broken', 'reset', 'aborted']):
                    logger.info(f"📪 Connection {connection_id} closed unexpectedly: {e}")
                else:
                    logger.error(f"❌ Receive error for {connection_id}: {e}")
                    logger.debug(traceback.format_exc())
                break
                    
    except Exception as e:
        logger.error(f"❌ WebSocket handler error for {connection_id}: {e}")
        logger.debug(traceback.format_exc())
        
    finally:
        # Always clean up
        connection_manager.remove_connection(connection_id)
        logger.info(f"🧹 Cleanup completed for {connection_id}")

# Custom request handler to reduce logging noise
class QuietWSGIRequestHandler(WSGIRequestHandler):
    def log_request(self, code='-', size='-'):
        # Only log non-WebSocket requests
        if not self.path.startswith('/ws'):
            super().log_request(code, size)

if __name__ == "__main__":
    print("=" * 60)
    print("🚀 BULLETPROOF WebSocket Server Starting...")
    print("=" * 60)
    print("✨ Features:")
    print("  🔒 Bulletproof error handling")
    print("  📊 Connection management & stats")
    print("  🏓 Ping/pong support")
    print("  🧪 Test message support")
    print("  🎵 Audio stream ready")
    print("  📈 Status endpoint: http://localhost:5000/status")
    print("=" * 60)
    print("💡 No more mysterious disconnections!")
    print("⏹️  Press Ctrl+C to stop")
    print("=" * 60)
    
    try:
        app.run(
            host="0.0.0.0",
            port=5000,
            debug=False,
            threaded=True,
            use_reloader=False,
            request_handler=QuietWSGIRequestHandler
        )
    except KeyboardInterrupt:
        print("\n" + "=" * 60)
        print("👋 Server stopped gracefully!")
        print(f"📊 Final stats: {connection_manager.get_connection_info()}")
        print("=" * 60)
    except Exception as e:
        logger.error(f"❌ Server startup failed: {e}")
        logger.debug(traceback.format_exc())