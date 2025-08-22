package signallingserver

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/KD0S-02/KDTransfer/internal/crypto"
	"github.com/KD0S-02/KDTransfer/internal/protocol"
)

func (ss *SignallingServer) startWriter(conn net.Conn, peer Peer) {
	go func() {
		for msg := range peer.Outgoing {
			_, err := conn.Write(msg)
			if err != nil {
				log.Println("Error writing to connection:", err)
				conn.Close()
				ss.RemoveUser(peer.ID)
				break
			}
		}
	}()
}

func (ss *SignallingServer) handleRegister(conn net.Conn, payload []byte) {
	id := crypto.GenerateID()

	var peerInfo PeerInfo

	err := json.Unmarshal(payload, &peerInfo)

	if err != nil {
		log.Printf("Failed parsing register payload from conn: %s",
			conn.RemoteAddr().String())
	}

	user := Peer{
		ID:       id,
		PeerInfo: peerInfo,
		Outgoing: make(chan []byte, 32),
	}

	ss.AddUser(id, user)
	ss.startWriter(conn, user)

	response := protocol.MakeMessage(protocol.SERVER_ACK, []byte(id))
	user.Outgoing <- response

	log.Printf("New connection established with ID: %s", id)
}

func (ss *SignallingServer) handlePeerLookup(conn net.Conn, payload []byte) {
	var peerLookUp PeerLookUp
	err := json.Unmarshal(payload, &peerLookUp)
	if err != nil {
		ss.sendErrorResponse(conn, "failed to decode lookup message")
		return
	}

	peer, found := ss.GetUser(peerLookUp.PeerID)

	if !found {
		ss.sendErrorResponse(conn, "User not found")
		return
	}

	// Send peer info to requesting user
	if err := ss.sendPeerInfo(conn, peer.PeerInfo); err != nil {
		log.Printf("Error sending peer lookup response: %v", err)
		return
	}

	// Forward requesting user's info to the found peer
	if err := ss.forwardUserInfo(peer, peerLookUp.SenderInfo); err != nil {
		log.Printf("Error forwarding user info to peer %s: %v", peerLookUp.PeerID, err)
		return
	}
}

func (ss *SignallingServer) sendErrorResponse(conn net.Conn, message string) {
	response := protocol.MakeMessage(protocol.ERROR, []byte(message))
	if _, err := conn.Write(response); err != nil {
		log.Printf("Error sending error response: %v", err)
	}
}

func (ss *SignallingServer) sendPeerInfo(conn net.Conn, addrInfo interface{}) error {
	data, err := json.Marshal(addrInfo)
	if err != nil {
		ss.sendErrorResponse(conn, "Error encoding address info")
		return fmt.Errorf("failed to marshal address info: %w", err)
	}

	response := protocol.MakeMessage(protocol.PEER_LOOKUP_ACK, data)
	if _, err := conn.Write(response); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}

func (ss *SignallingServer) forwardUserInfo(peer Peer, userInfo PeerInfo) error {
	data, err := json.Marshal(userInfo)
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

func (ss *SignallingServer) HandleConnection(conn net.Conn) {
	defer conn.Close()

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
			ss.handleRegister(conn, payload)

		case protocol.PEER_INFO_LOOKUP:
			ss.handlePeerLookup(conn, payload)

		default:
			log.Println("Unknown operation code:", opCode)
		}
	}
}
