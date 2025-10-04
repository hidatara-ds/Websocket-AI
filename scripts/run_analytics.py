#!/usr/bin/env python3
"""
Analytics Runner for WebSocket AI Assistant
Runs data processing and analytics on the system
"""

import sys
import os
import json
import asyncio
from datetime import datetime
from data_processor import DataProcessor, MessageData, APIClient, PerformanceMonitor

async def main():
    """Main analytics function"""
    print("🔍 WebSocket AI Assistant Analytics")
    print("=" * 50)
    
    # Initialize components
    processor = DataProcessor()
    monitor = PerformanceMonitor()
    
    # Simulate some data collection
    print("📊 Collecting data...")
    
    # Add sample messages
    sample_messages = [
        MessageData(
            id=f"msg_{i}",
            timestamp=datetime.now(),
            user_id=f"user_{i % 3}",
            message_type=["text", "audio", "image"][i % 3],
            content=f"Sample message {i}",
            response_time=1.0 + (i * 0.1),
            success=True
        )
        for i in range(10)
    ]
    
    for msg in sample_messages:
        processor.add_message(msg)
        monitor.record_metric("response_time", msg.response_time)
    
    # Generate analytics
    print("📈 Generating analytics...")
    analytics = processor.get_analytics()
    
    print("\n📊 Analytics Results:")
    print(f"Total Messages: {analytics['total_messages']}")
    print(f"Success Rate: {analytics['success_rate']}%")
    print(f"Average Response Time: {analytics['average_response_time']}s")
    print(f"Uptime: {analytics['uptime_hours']:.2f} hours")
    
    print("\n📋 Message Type Distribution:")
    for msg_type, count in analytics['message_type_distribution'].items():
        print(f"  {msg_type}: {count}")
    
    # Performance metrics
    print("\n⚡ Performance Metrics:")
    performance = monitor.get_performance_summary()
    for metric, stats in performance.items():
        print(f"  {metric}: avg={stats['avg']:.2f}, min={stats['min']:.2f}, max={stats['max']:.2f}")
    
    # Export data
    print("\n💾 Exporting data...")
    processor.export_data("analytics_export.json")
    print("✅ Data exported to analytics_export.json")
    
    # Try to connect to API (if running)
    print("\n🔌 Checking API connection...")
    try:
        async with APIClient() as client:
            health = await client.get_health()
            print(f"✅ API Health: {health.get('status', 'unknown')}")
    except Exception as e:
        print(f"⚠️  API not available: {e}")
    
    print("\n✅ Analytics complete!")

if __name__ == "__main__":
    asyncio.run(main())
