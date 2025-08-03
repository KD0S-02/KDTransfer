package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	SignallingServerHost string
	SignallingServerPort string
	TCPPort              string
	UDPPort              string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, using defaults")
	}

	config := &Config{
		SignallingServerHost: getEnvOrDefault("SIGNALLING_SERVER_HOST", "localhost"),
		SignallingServerPort: getEnvOrDefault("SIGNALLING_SERVER_PORT", "8080"),
		TCPPort:              getEnvOrDefault("TCP_PORT", "2502"),
		UDPPort:              getEnvOrDefault("UDP_PORT", "2503"),
	}

	return config
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
