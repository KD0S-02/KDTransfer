package config

import (
	"os"
)

type Config struct {
	SignalingServerAddress string
	TCPPort                string
	UDPPort                string
}

func LoadConfig() *Config {

	config := &Config{
		SignalingServerAddress: "localhost:8080",
		TCPPort:                "2502",
		UDPPort:                "2503",
	}

	if addr := os.Getenv("SIGNALING_SERVER_ADDRESS"); addr != "" {
		config.SignalingServerAddress = addr
	} else {
		host := getEnvOrDefault("SIGNALING_SERVER_HOST", "localhost")
		port := getEnvOrDefault("SIGNALING_SERVER_PORT", "8080")
		config.SignalingServerAddress = host + ":" + port
	}

	if port := os.Getenv("TCP_PORT"); port != "" {
		config.TCPPort = port
	}

	if port := os.Getenv("UDP_PORT"); port != "" {
		config.UDPPort = port
	}

	return config
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
