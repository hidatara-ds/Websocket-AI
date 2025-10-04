#!/usr/bin/env python3
"""
Audio Processing Utilities for WebSocket AI Assistant
Handles audio file processing, format conversion, and analysis
"""

import os
import json
import logging
import wave
import numpy as np
from typing import Dict, List, Optional, Tuple
from dataclasses import dataclass
import librosa
import soundfile as sf
from scipy import signal
import matplotlib.pyplot as plt

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

@dataclass
class AudioMetadata:
    """Audio file metadata"""
    filename: str
    duration: float
    sample_rate: int
    channels: int
    format: str
    bit_depth: int
    file_size: int

class AudioProcessor:
    """Main audio processing class"""
    
    def __init__(self):
        self.supported_formats = ['wav', 'mp3', 'flac', 'ogg', 'm4a']
        self.default_sample_rate = 16000
    
    def load_audio(self, filepath: str) -> Tuple[np.ndarray, int]:
        """Load audio file and return data with sample rate"""
        try:
            audio_data, sample_rate = librosa.load(filepath, sr=None)
            logger.info(f"Loaded audio: {filepath}, duration: {len(audio_data)/sample_rate:.2f}s")
            return audio_data, sample_rate
        except Exception as e:
            logger.error(f"Error loading audio {filepath}: {e}")
            raise
    
    def get_metadata(self, filepath: str) -> AudioMetadata:
        """Get audio file metadata"""
        try:
            # Get file info
            file_size = os.path.getsize(filepath)
            
            # Load audio to get properties
            audio_data, sample_rate = self.load_audio(filepath)
            
            # Get format from extension
            format_name = os.path.splitext(filepath)[1][1:].lower()
            
            return AudioMetadata(
                filename=os.path.basename(filepath),
                duration=len(audio_data) / sample_rate,
                sample_rate=sample_rate,
                channels=1 if audio_data.ndim == 1 else audio_data.shape[1],
                format=format_name,
                bit_depth=16,  # Default assumption
                file_size=file_size
            )
        except Exception as e:
            logger.error(f"Error getting metadata for {filepath}: {e}")
            raise
    
    def resample_audio(self, audio_data: np.ndarray, original_sr: int, target_sr: int) -> np.ndarray:
        """Resample audio to target sample rate"""
        try:
            resampled = librosa.resample(audio_data, orig_sr=original_sr, target_sr=target_sr)
            logger.info(f"Resampled from {original_sr}Hz to {target_sr}Hz")
            return resampled
        except Exception as e:
            logger.error(f"Error resampling audio: {e}")
            raise
    
    def normalize_audio(self, audio_data: np.ndarray) -> np.ndarray:
        """Normalize audio to [-1, 1] range"""
        max_val = np.max(np.abs(audio_data))
        if max_val > 0:
            return audio_data / max_val
        return audio_data
    
    def apply_noise_reduction(self, audio_data: np.ndarray, sample_rate: int) -> np.ndarray:
        """Apply basic noise reduction using spectral gating"""
        try:
            # Compute STFT
            stft = librosa.stft(audio_data)
            magnitude = np.abs(stft)
            phase = np.angle(stft)
            
            # Estimate noise floor
            noise_floor = np.percentile(magnitude, 10, axis=1, keepdims=True)
            
            # Apply spectral gating
            gate_threshold = noise_floor * 2
            mask = magnitude > gate_threshold
            magnitude_cleaned = magnitude * mask
            
            # Reconstruct audio
            stft_cleaned = magnitude_cleaned * np.exp(1j * phase)
            audio_cleaned = librosa.istft(stft_cleaned)
            
            logger.info("Applied noise reduction")
            return audio_cleaned
        except Exception as e:
            logger.error(f"Error applying noise reduction: {e}")
            return audio_data
    
    def detect_silence(self, audio_data: np.ndarray, sample_rate: int, 
                      silence_threshold: float = 0.01, min_silence_duration: float = 0.5) -> List[Tuple[float, float]]:
        """Detect silence periods in audio"""
        try:
            # Calculate RMS energy
            frame_length = int(sample_rate * 0.1)  # 100ms frames
            rms = []
            
            for i in range(0, len(audio_data), frame_length):
                frame = audio_data[i:i+frame_length]
                rms.append(np.sqrt(np.mean(frame**2)))
            
            rms = np.array(rms)
            
            # Find silence periods
            silence_frames = rms < silence_threshold
            silence_periods = []
            
            in_silence = False
            silence_start = 0
            
            for i, is_silent in enumerate(silence_frames):
                if is_silent and not in_silence:
                    silence_start = i * 0.1  # Convert frame index to time
                    in_silence = True
                elif not is_silent and in_silence:
                    silence_duration = (i * 0.1) - silence_start
                    if silence_duration >= min_silence_duration:
                        silence_periods.append((silence_start, silence_start + silence_duration))
                    in_silence = False
            
            logger.info(f"Detected {len(silence_periods)} silence periods")
            return silence_periods
        except Exception as e:
            logger.error(f"Error detecting silence: {e}")
            return []
    
    def extract_features(self, audio_data: np.ndarray, sample_rate: int) -> Dict[str, float]:
        """Extract audio features for analysis"""
        try:
            features = {}
            
            # Basic features
            features['duration'] = len(audio_data) / sample_rate
            features['rms_energy'] = np.sqrt(np.mean(audio_data**2))
            features['zero_crossing_rate'] = np.mean(librosa.feature.zero_crossing_rate(audio_data))
            
            # Spectral features
            spectral_centroids = librosa.feature.spectral_centroid(y=audio_data, sr=sample_rate)[0]
            features['spectral_centroid_mean'] = np.mean(spectral_centroids)
            features['spectral_centroid_std'] = np.std(spectral_centroids)
            
            # MFCC features
            mfccs = librosa.feature.mfcc(y=audio_data, sr=sample_rate, n_mfcc=13)
            features['mfcc_mean'] = np.mean(mfccs)
            features['mfcc_std'] = np.std(mfccs)
            
            # Tempo
            tempo, _ = librosa.beat.beat_track(y=audio_data, sr=sample_rate)
            features['tempo'] = tempo
            
            logger.info("Extracted audio features")
            return features
        except Exception as e:
            logger.error(f"Error extracting features: {e}")
            return {}
    
    def save_audio(self, audio_data: np.ndarray, sample_rate: int, filepath: str) -> None:
        """Save audio data to file"""
        try:
            sf.write(filepath, audio_data, sample_rate)
            logger.info(f"Saved audio to {filepath}")
        except Exception as e:
            logger.error(f"Error saving audio to {filepath}: {e}")
            raise
    
    def create_spectrogram(self, audio_data: np.ndarray, sample_rate: int, 
                          filepath: str) -> None:
        """Create and save spectrogram visualization"""
        try:
            plt.figure(figsize=(12, 8))
            
            # Create spectrogram
            D = librosa.amplitude_to_db(np.abs(librosa.stft(audio_data)), ref=np.max)
            librosa.display.specshow(D, y_axis='hz', x_axis='time', sr=sample_rate)
            plt.colorbar(format='%+2.0f dB')
            plt.title('Spectrogram')
            plt.xlabel('Time')
            plt.ylabel('Frequency (Hz)')
            
            plt.tight_layout()
            plt.savefig(filepath, dpi=150, bbox_inches='tight')
            plt.close()
            
            logger.info(f"Saved spectrogram to {filepath}")
        except Exception as e:
            logger.error(f"Error creating spectrogram: {e}")
    
    def process_audio_file(self, input_path: str, output_path: str, 
                          target_sr: int = 16000, apply_processing: bool = True) -> Dict[str, any]:
        """Complete audio processing pipeline"""
        try:
            # Load audio
            audio_data, original_sr = self.load_audio(input_path)
            
            # Get metadata
            metadata = self.get_metadata(input_path)
            
            # Resample if needed
            if original_sr != target_sr:
                audio_data = self.resample_audio(audio_data, original_sr, target_sr)
                sample_rate = target_sr
            else:
                sample_rate = original_sr
            
            # Apply processing
            if apply_processing:
                audio_data = self.normalize_audio(audio_data)
                audio_data = self.apply_noise_reduction(audio_data, sample_rate)
            
            # Save processed audio
            self.save_audio(audio_data, sample_rate, output_path)
            
            # Extract features
            features = self.extract_features(audio_data, sample_rate)
            
            # Create spectrogram
            spectrogram_path = output_path.replace('.wav', '_spectrogram.png')
            self.create_spectrogram(audio_data, sample_rate, spectrogram_path)
            
            # Detect silence
            silence_periods = self.detect_silence(audio_data, sample_rate)
            
            result = {
                'input_file': input_path,
                'output_file': output_path,
                'metadata': {
                    'original_duration': metadata.duration,
                    'processed_duration': len(audio_data) / sample_rate,
                    'sample_rate': sample_rate,
                    'channels': metadata.channels
                },
                'features': features,
                'silence_periods': silence_periods,
                'processing_applied': apply_processing
            }
            
            logger.info(f"Successfully processed {input_path}")
            return result
            
        except Exception as e:
            logger.error(f"Error processing audio file {input_path}: {e}")
            raise

def main():
    """Main function for testing"""
    processor = AudioProcessor()
    
    # Example usage
    print("Audio Processor initialized")
    print(f"Supported formats: {processor.supported_formats}")
    print(f"Default sample rate: {processor.default_sample_rate}Hz")
    
    # You would typically process actual audio files here
    # result = processor.process_audio_file("input.wav", "output.wav")

if __name__ == "__main__":
    main()
