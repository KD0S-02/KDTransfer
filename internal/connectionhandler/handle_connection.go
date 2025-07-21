package connectionhandler

import (
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
	ip, port, err := net.SplitHostPort(string(payload))

	if err != nil {
		log.Println("Error parsing remote address:", err)
		response := protocol.MakeMessage(protocol.ERROR, []byte("Invalid remote address"))
		conn.Write(response)
		user := usermap.Peer{}
		return user
	}

	user := usermap.Peer{
		ID:       id,
		IP:       ip,
		Port:     port,
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

	peer, found := usermap.GetUser(string(payload))

	if !found {
		response := protocol.MakeMessage(protocol.ERROR, []byte("User not found"))
		conn.Write(response)
	} else {

		peerInfo := protocol.MakeMessage(protocol.PEER_LOOKUP_ACK,
			[]byte(peer.IP+":"+peer.Port))
		_, err := conn.Write(peerInfo)

		userInfo := protocol.MakeMessage(protocol.PEER_INFO_FORWARD, []byte(user.IP+":"+user.Port))
		peer.Outgoing <- userInfo

		if err != nil {
			log.Println("Error sending peer lookup response:", err)
			return
		}

	}
}

// handles the incoming connection from a client
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
