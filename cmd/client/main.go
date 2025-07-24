package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/KD0S-02/KDTransfer/internal/protocol"
)

func handleInput(conn net.Conn) {

	switch os.Args[1] {

	case "send":
		HandleSendCommand(conn)

	case "recv":

		listener, err := net.Listen("tcp", ":0")

		addr := listener.Addr().String()
		port := addr[strings.LastIndex(addr, ":")+1:]

		if err != nil {
			log.Fatal("Error when listening for direct connections.")
			os.Exit(1)
		}

		request := protocol.MakeMessage(protocol.SERVER_HELLO, []byte(port))

		conn.Write(request)

		defer listener.Close()

		opCode, payload, err := protocol.ReadMessage(conn)

		ValidateResponse(err, protocol.SERVER_ACK, opCode, conn)

		log.Println("Current ID: " + string(payload))

		go listenForDirectConnections(listener)

		r := bufio.NewScanner(os.Stdin)

		for r.Scan() {
			line := r.Text()
			if line == "disconnect" {
				os.Exit(1)
			} else {
				continue
			}
		}
	}
}

func handleDirectConnections(peerConn net.Conn) {

	defer peerConn.Close()

	close := false

	for !close {
		close = HandleMessages(peerConn)
	}

	log.Println("Closing direct connection with peer:", peerConn.RemoteAddr().String())
}

func listenForDirectConnections(listener net.Listener) {

	for {

		peerConn, err := listener.Accept()

		if err != nil {
			log.Println("Error accepting connection:", err)
			return
		}

		go handleDirectConnections(peerConn)
	}

}

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Missing Command: 'send' or 'recv'")
		return
	}

	conn, err := net.Dial("tcp", ":8080")

	if err != nil {
		fmt.Println("Error Connecting to Communication Server:", err)
		os.Exit(1)
	}

	defer conn.Close()

	handleInput(conn)

}
