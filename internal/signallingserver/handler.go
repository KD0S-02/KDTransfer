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

func (ss *SignallingServer) handleRegister(conn net.Conn, payload []byte) (*Peer, string,
	error) {
	id := crypto.GenerateID()

	var peerInfo PeerInfo
	if err := json.Unmarshal(payload, &peerInfo); err != nil {
		return nil, "", fmt.Errorf("failed parsing register payload: %w", err)
	}

	user := NewPeer(id, peerInfo)
	ss.AddUser(id, user)

	// Start writer goroutine for this peer
	go func() {
		for msg := range user.Outgoing {
			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if _, err := conn.Write(msg); err != nil {
				log.Printf("Write error for peer %s: %v", id, err)
				return
			}
		}
		log.Printf("Writer goroutine exiting for peer %s", id)
	}()

	// Send registration acknowledgment
	if err := ss.SendToPeer(user, protocol.SERVER_ACK, []byte(id)); err != nil {
		log.Printf("Failed sending SERVER_ACK to %s: %v", id, err)
		ss.RemoveUser(id)
		return nil, "", fmt.Errorf("failed sending register ack: %w", err)
	}

	log.Printf("New connection established with ID: %s", id)
	return user, id, nil
}

func (ss *SignallingServer) handlePeerLookup(user *Peer, payload []byte) error {
	var peerLookUp PeerLookUp
	if err := json.Unmarshal(payload, &peerLookUp); err != nil {
		if sendErr := ss.SendToPeer(user, protocol.ERROR, []byte("invalid request")); sendErr != nil {
			log.Printf("Failed sending error to user %s: %v", user.ID, sendErr)
		}
		return nil
	}

	peer, found := ss.GetUser(peerLookUp.PeerID)
	if !found {
		if sendErr := ss.SendToPeer(user, protocol.ERROR, []byte("peer not found")); sendErr != nil {
			log.Printf("Failed sending error to user %s: %v", user.ID, sendErr)
		}
		return nil
	}

	// Marshal and send peer info to requesting user
	data, err := json.Marshal(peer.PeerInfo)
	if err != nil {
		log.Printf("Failed to marshal peer info for %s: %v", peer.ID, err)
		if sendErr := ss.SendToPeer(user, protocol.ERROR, []byte("server error")); sendErr != nil {
			log.Printf("Failed sending error to user %s: %v", user.ID, sendErr)
		}
		return nil
	}

	if err := ss.SendToPeer(user, protocol.PEER_LOOKUP_ACK, data); err != nil {
		return fmt.Errorf("failed sending lookup response to user %s: %w", user.ID, err)
	}

	// Marshal and forward sender info to target peer (best effort)
	senderData, err := json.Marshal(peerLookUp.SenderInfo)
	if err != nil {
		log.Printf("Failed to marshal sender info: %v", err)
		return nil
	}

	if err := ss.SendToPeer(peer, protocol.PEER_INFO_FORWARD, senderData); err != nil {
		log.Printf("Warning: failed forwarding sender info to peer %s: %v", peer.ID, err)
	}

	log.Printf("Lookup completed: user %s <-> peer %s", user.ID, peer.ID)
	return nil
}

func (ss *SignallingServer) HandleConnection(conn net.Conn) error {
	defer conn.Close()

	var userID string
	var user *Peer
	var registered bool

	defer func() {
		if registered {
			ss.RemoveUser(userID)
			log.Printf("Connection closed for user: %s", userID)
		}
	}()

	for {
		buf := ss.GetBuffer()
		opCode, n, err := protocol.ReadMessage(conn, buf)

		if err != nil {
			ss.PutBuffer(buf)
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("error reading message: %w", err)
		}

		// Copy payload and return buffer
		payload := make([]byte, n)
		copy(payload, buf[:n])
		ss.PutBuffer(buf)

		switch opCode {
		case protocol.SERVER_HELLO:
			if registered {
				return fmt.Errorf("duplicate registration attempt")
			}

			user, userID, err = ss.handleRegister(conn, payload)
			if err != nil {
				return fmt.Errorf("registration failed: %w", err)
			}

			registered = true
			log.Printf("User %s registered successfully", userID)

		case protocol.PEER_INFO_LOOKUP:
			if !registered {
				return fmt.Errorf("peer lookup before registration")
			}

			if err := ss.handlePeerLookup(user, payload); err != nil {
				log.Printf("Peer lookup error for %s: %v", userID, err)
				return fmt.Errorf("peer lookup failed: %w", err)
			}

		default:
			return fmt.Errorf("unknown operation code: %d", opCode)
		}
	}
}
