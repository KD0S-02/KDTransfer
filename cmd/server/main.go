package main

import (
	"fmt"
	"net"
	"os"

	"github.com/KD0S-02/KDTransfer/internal/connectionhandler"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")

	if err != nil {
		fmt.Println("Error starting server:", err)
		os.Exit(1)
	}

	defer listener.Close()

	fmt.Println("Server is listening on port 8080...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go connectionhandler.HandleConnection(conn)
	}

}
