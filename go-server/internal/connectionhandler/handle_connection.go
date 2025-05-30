package connectionhandler

import (
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/KD0S-02/KDTransfer/go-server/internal/protocol"
	"github.com/KD0S-02/KDTransfer/go-server/internal/usermap"
)

func generateID() string {
	letters := []rune("abcdefghijlmnopqrstuvwxyz1234567890")

	randID := make([]rune, 8)

	source := rand.NewSource(time.Now().UnixNano())
	random := rand.New(source)

	for i := 0; i < 8; i++ {
		randID[i] = letters[random.Intn(len(letters))]
	}

	return string(randID)
}

// HandleConnection handles the incoming connection from a client
func HandleConnection(conn net.Conn) {
	defer conn.Close()

	// Buffer for reading data from the client
	var msg []byte

	// Read the message header first (1 byte for command + 4 bytes for payload length)
	header := make([]byte, 5)
	_, err := conn.Read(header)
	if err != nil {
		log.Println("Error reading message header:", err)
		return
	}

	// Extract command and payload length from the header
	command := header[0]
	payloadLength := protocol.GetPayloadLength(header)

	// Read the payload
	if payloadLength > 0 {
		msg = make([]byte, payloadLength)
		_, err = conn.Read(msg)
		if err != nil {
			log.Println("Error reading message payload:", err)
			return
		}
	}

	commStart := false

	if command == protocol.SERVER_HELLO {
		log.Printf("KDTPv1: HELLO %s\n", conn.RemoteAddr().String())

		// Generate an ID and add user to the map
		id := generateID()
		usermap.AddUser(id, conn.RemoteAddr().String())
		idLength := len(id)

		// // Send an ACK response back to the client with the generated ID
		ackMessage := append([]byte{protocol.SERVER_ACK}, byte(idLength>>24), byte(idLength>>16),
			byte(idLength>>8), byte(idLength))

		ackMessage = append(ackMessage, []byte(id)...)

		commStart = true

		_, err = conn.Write(ackMessage)
		if err != nil {
			log.Println("Error sending ACK to client:", err)
			return
		}
		log.Printf("%s Connected\n", conn.RemoteAddr().String())
	}

	for commStart {

		header := make([]byte, 5)
		_, err := conn.Read(header)
		if err != nil {
			log.Println("Error reading message header during communication:", err)
			return
		}

		command := header[0]
		payloadLength := protocol.GetPayloadLength(header)

		var msg []byte
		if payloadLength > 0 {
			msg = make([]byte, payloadLength)
			_, err = conn.Read(msg)
			if err != nil {
				log.Println("Error reading message payload during communication:", err)
				return
			}
		}

		switch command {

		case protocol.PEER_INFO:
			break
		case protocol.BYE:
			commStart = false

		default:
			log.Printf("Invalid Command from %s", conn.RemoteAddr().String())
		}
	}

}
