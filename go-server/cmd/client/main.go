package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"os"

	protocol "github.com/KD0S-02/KDTransfer/go-server/internal/protocol"
)

func main() {
	conn, err := net.Dial("tcp", ":8080")

	if err != nil {
		fmt.Println("Error Connecting to Communication Server:", err)
		os.Exit(1)
	}

	defer conn.Close()

	// sending hello message to comm server
	var message bytes.Buffer
	message.WriteByte(protocol.SERVER_HELLO)
	message.Write([]byte{0, 0, 0, 0})

	_, err = conn.Write(message.Bytes())
	if err != nil {
		fmt.Println("Error sending HELLO: ", err)
		os.Exit(1)
	}

	header := make([]byte, 5)

	conn.Read(header)

	var id string

	if header[0] == protocol.ACK {
		payloadLength := protocol.GetPayloadLength(header)
		payload := make([]byte, payloadLength)
		conn.Read(payload)
		id = string(payload)
	}

	log.Printf("Starting with ID: %s", string(id))

	reader := bufio.NewScanner(os.Stdin)

	for {
		fmt.Println(">")
		for reader.Scan() {
			fmt.Println(reader.Text())
		}
	}

}
