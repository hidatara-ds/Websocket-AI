#!/usr/bin/env python3
"""
Data Processor for WebSocket AI Assistant
Handles data processing, analytics, and utility functions
"""

import json
import logging
import time
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any
from dataclasses import dataclass
import asyncio
import aiohttp
import pandas as pd
import numpy as np

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

@dataclass
class MessageData:
    """Represents a message in the system"""
    id: str
    timestamp: datetime
    user_id: str
    message_type: str  # 'text', 'audio', 'image'
    content: str
    response_time: Optional[float] = None
    success: bool = True
    error_message: Optional[str] = None

class DataProcessor:
    """Main data processor class"""
    
    def __init__(self):
        self.messages: List[MessageData] = []
        self.analytics_data = {}
        self.start_time = datetime.now()
    
    def add_message(self, message: MessageData) -> None:
        """Add a new message to the processor"""
        self.messages.append(message)
        logger.info(f"Added message {message.id} of type {message.message_type}")
    
    def get_analytics(self) -> Dict[str, Any]:
        """Generate analytics data from messages"""
        if not self.messages:
            return {"error": "No messages to analyze"}
        
        # Convert to DataFrame for easier analysis
        df = pd.DataFrame([
            {
                'timestamp': msg.timestamp,
                'message_type': msg.message_type,
                'response_time': msg.response_time,
                'success': msg.success
            }
            for msg in self.messages
        ])
        
        # Calculate metrics
        total_messages = len(df)
        success_rate = df['success'].mean() * 100
        avg_response_time = df['response_time'].dropna().mean()
        
        # Message type distribution
        type_distribution = df['message_type'].value_counts().to_dict()
        
        # Time-based analysis
        df['hour'] = df['timestamp'].dt.hour
        hourly_distribution = df.groupby('hour').size().to_dict()
        
        return {
            "total_messages": total_messages,
            "success_rate": round(success_rate, 2),
            "average_response_time": round(avg_response_time, 2) if not pd.isna(avg_response_time) else 0,
            "message_type_distribution": type_distribution,
            "hourly_distribution": hourly_distribution,
            "uptime_hours": (datetime.now() - self.start_time).total_seconds() / 3600
        }
    
    def export_data(self, filename: str) -> None:
        """Export processed data to JSON file"""
        analytics = self.get_analytics()
        analytics['messages'] = [
            {
                'id': msg.id,
                'timestamp': msg.timestamp.isoformat(),
                'user_id': msg.user_id,
                'message_type': msg.message_type,
                'content_length': len(msg.content),
                'response_time': msg.response_time,
                'success': msg.success,
                'error_message': msg.error_message
            }
            for msg in self.messages
        ]
        
        with open(filename, 'w') as f:
            json.dump(analytics, f, indent=2)
        
        logger.info(f"Data exported to {filename}")
    
    def cleanup_old_data(self, days: int = 7) -> None:
        """Remove messages older than specified days"""
        cutoff_date = datetime.now() - timedelta(days=days)
        original_count = len(self.messages)
        
        self.messages = [
            msg for msg in self.messages 
            if msg.timestamp > cutoff_date
        ]
        
        removed_count = original_count - len(self.messages)
        logger.info(f"Cleaned up {removed_count} old messages")

class APIClient:
    """Client for external API calls"""
    
    def __init__(self, base_url: str = "http://localhost:8081"):
        self.base_url = base_url
        self.session: Optional[aiohttp.ClientSession] = None
    
    async def __aenter__(self):
        self.session = aiohttp.ClientSession()
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        if self.session:
            await self.session.close()
    
    async def get_health(self) -> Dict[str, Any]:
        """Get health status from the API"""
        if not self.session:
            raise RuntimeError("Session not initialized")
        
        async with self.session.get(f"{self.base_url}/health") as response:
            return await response.json()
    
    async def get_metrics(self) -> Dict[str, Any]:
        """Get metrics from the API"""
        if not self.session:
            raise RuntimeError("Session not initialized")
        
        async with self.session.get(f"{self.base_url}/metrics") as response:
            return await response.json()

class PerformanceMonitor:
    """Monitor system performance"""
    
    def __init__(self):
        self.metrics_history: List[Dict[str, Any]] = []
        self.start_time = time.time()
    
    def record_metric(self, metric_name: str, value: float, tags: Dict[str, str] = None) -> None:
        """Record a performance metric"""
        metric = {
            'timestamp': datetime.now().isoformat(),
            'metric_name': metric_name,
            'value': value,
            'tags': tags or {}
        }
        self.metrics_history.append(metric)
        logger.debug(f"Recorded metric {metric_name}: {value}")
    
    def get_performance_summary(self) -> Dict[str, Any]:
        """Get performance summary"""
        if not self.metrics_history:
            return {"error": "No metrics recorded"}
        
        # Group by metric name
        metric_groups = {}
        for metric in self.metrics_history:
            name = metric['metric_name']
            if name not in metric_groups:
                metric_groups[name] = []
            metric_groups[name].append(metric['value'])
        
        # Calculate statistics
        summary = {}
        for name, values in metric_groups.items():
            summary[name] = {
                'count': len(values),
                'min': min(values),
                'max': max(values),
                'avg': sum(values) / len(values),
                'latest': values[-1]
            }
        
        return summary

def main():
    """Main function for testing"""
    processor = DataProcessor()
    
    # Add some sample data
    sample_messages = [
        MessageData(
            id="1",
            timestamp=datetime.now(),
            user_id="user1",
            message_type="text",
            content="Hello, how are you?",
            response_time=1.2,
            success=True
        ),
        MessageData(
            id="2",
            timestamp=datetime.now(),
            user_id="user2",
            message_type="audio",
            content="base64_audio_data",
            response_time=2.5,
            success=True
        ),
        MessageData(
            id="3",
            timestamp=datetime.now(),
            user_id="user1",
            message_type="text",
            content="What's the weather like?",
            response_time=0.8,
            success=True
        )
    ]
    
    for msg in sample_messages:
        processor.add_message(msg)
    
    # Generate analytics
    analytics = processor.get_analytics()
    print("Analytics:", json.dumps(analytics, indent=2))
    
    # Export data
    processor.export_data("analytics.json")
    
    # Performance monitoring
    monitor = PerformanceMonitor()
    monitor.record_metric("response_time", 1.2)
    monitor.record_metric("response_time", 2.5)
    monitor.record_metric("response_time", 0.8)
    
    performance = monitor.get_performance_summary()
    print("Performance:", json.dumps(performance, indent=2))

if __name__ == "__main__":
    main()
