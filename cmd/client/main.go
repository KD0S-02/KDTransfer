package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/KD0S-02/KDTransfer/internal/config"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Usage: program [send|recv]")
		os.Exit(1)
	}

	if err := run(); err != nil {
		log.Fatal(err)
	}

}

func run() error {
	config := config.LoadConfig()
	conn, err := net.Dial("tcp", config.SignalingServerAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to signaling server: %w", err)
	}
	defer conn.Close()

	return handleCommand(conn)
}

func handleCommand(conn net.Conn) error {
	switch os.Args[1] {
	case "send":
		return HandleSendCommand(conn)
	case "recv":
		return HandleReceiveCommand(conn)
	default:
		return fmt.Errorf("unknown command: %s", os.Args[1])
	}
}
