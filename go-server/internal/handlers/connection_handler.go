package handlers

import (
	"fmt"
	"net"
)

func HandleConnection(conn net.Conn) {
	defer conn.Close()

	// Handle the connection (e.g., read/write data)
	fmt.Println("Handling new connection from", conn.RemoteAddr())
	// Add your connection handling logic here
}
