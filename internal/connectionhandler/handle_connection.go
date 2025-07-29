package connectionhandler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/KD0S-02/KDTransfer/internal/protocol"
	"github.com/KD0S-02/KDTransfer/internal/usermap"
)

func startWriter(conn net.Conn, peer usermap.Peer) {
	go func() {
		for msg := range peer.Outgoing {
			_, err := conn.Write(msg)
			if err != nil {
				log.Println("Error writing to connection:", err)
				conn.Close()
				usermap.RemoveUser(peer.ID)
				break
			}
		}
	}()
}

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

func handleRegister(conn net.Conn, payload []byte) usermap.Peer {
	id := generateID()

	var addrInfo protocol.Addressinfo

	err := json.Unmarshal(payload, &addrInfo)

	if err != nil {
		log.Printf("Failed parsing register payload from conn: %s",
			conn.RemoteAddr().String())
	}

	user := usermap.Peer{
		ID:       id,
		AddrInfo: addrInfo,
		Outgoing: make(chan []byte, 32),
	}

	usermap.AddUser(id, user)
	startWriter(conn, user)

	response := protocol.MakeMessage(protocol.SERVER_ACK, []byte(id))
	user.Outgoing <- response

	log.Printf("New connection established with ID: %s", id)

	return user
}

func handlePeerLookup(conn net.Conn, payload []byte, user usermap.Peer) {
	peerID := string(payload)
	peer, found := usermap.GetUser(peerID)

	if !found {
		sendErrorResponse(conn, "User not found")
		return
	}

	// Send peer info to requesting user
	if err := sendPeerInfo(conn, peer.AddrInfo); err != nil {
		log.Printf("Error sending peer lookup response: %v", err)
		return
	}

	// Forward requesting user's info to the found peer
	if err := forwardUserInfo(peer, user.AddrInfo); err != nil {
		log.Printf("Error forwarding user info to peer %s: %v", peerID, err)
		return
	}

	log.Printf("Successfully completed peer lookup: %s found %s", user.ID, peerID)
}

func sendErrorResponse(conn net.Conn, message string) {
	response := protocol.MakeMessage(protocol.ERROR, []byte(message))
	if _, err := conn.Write(response); err != nil {
		log.Printf("Error sending error response: %v", err)
	}
}

func sendPeerInfo(conn net.Conn, addrInfo interface{}) error {
	data, err := json.Marshal(addrInfo)
	if err != nil {
		sendErrorResponse(conn, "Error encoding address info")
		return fmt.Errorf("failed to marshal address info: %w", err)
	}

	response := protocol.MakeMessage(protocol.PEER_LOOKUP_ACK, data)
	if _, err := conn.Write(response); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}

func forwardUserInfo(peer usermap.Peer, addrInfo interface{}) error {
	data, err := json.Marshal(addrInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal user address info: %w", err)
	}

	forwardResponse := protocol.MakeMessage(protocol.PEER_INFO_FORWARD, data)

	select {
	case peer.Outgoing <- forwardResponse:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout sending to peer channel")
	default:
		return fmt.Errorf("peer channel is full")
	}
}

func HandleConnection(conn net.Conn) {
	defer conn.Close()

	var user usermap.Peer

	for {

		opCode, payload, err := protocol.ReadMessage(conn)

		if err != nil {

			if err == io.EOF {
				log.Println("Connection closed by client")
				return
			}

			log.Println("Error reading message:", err)
			return
		}

		switch opCode {

		case protocol.SERVER_HELLO:
			user = handleRegister(conn, payload)

		case protocol.PEER_INFO_LOOKUP:
			handlePeerLookup(conn, payload, user)

		default:
			log.Println("Unknown operation code:", opCode)
		}

	}

}
