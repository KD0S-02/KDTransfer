package main

import (
	"fmt"
	"os"

	"github.com/KD0S-02/KDTransfer/internal/config"
	"github.com/KD0S-02/KDTransfer/internal/signallingserver"
)

func main() {
	cfg := config.LoadConfig()
	ss, err := signallingserver.NewSignallingServer(cfg)
	if err != nil {
		fmt.Printf("Failed to create signalling server: %v\n", err)
		os.Exit(1)
	}
	err = ss.Start()
	if err != nil {
		fmt.Printf("Failed to start signalling server: %v\n", err)
		os.Exit(1)
	}
}
