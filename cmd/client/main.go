package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/KD0S-02/KDTransfer/internal/protocol"
)

func closeWithError(msg string, conn net.Conn) {
	log.Println(msg)
	conn.Close()
	os.Exit(1)
}

func validateResponse(err error, expectedOpCode byte, actualOpCode byte, conn net.Conn) {

	if err != nil {
		closeWithError("Server error! closing connection.", conn)
	}

	if actualOpCode != expectedOpCode {
		closeWithError("Error! unexpected server response.", conn)
	}

}

func handleSendCommand(conn net.Conn) {
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	file := sendCmd.String("file", "", "Path of file to be transferred.")
	peer := sendCmd.String("peer", "", "ID of receiving peer.")

	sendCmd.Parse(os.Args[2:])

	if *file == "" || *peer == "" {
		closeWithError("Usage: send --file <filepath> --peer <peerID>", conn)
	}

	request := protocol.MakeMessage(protocol.PEER_INFO_LOOKUP, []byte(*peer))
	conn.Write(request)

	opCode, payload, err := protocol.ReadMessage(conn)

	validateResponse(err, protocol.PEER_LOOKUP_ACK, opCode, conn)

	peerConn, err := net.Dial("tcp", string(payload))

	log.Println(string(payload))

	if err != nil {
		fmt.Println(err.Error())
		closeWithError("Failed to make direct connection to peer.", conn)
	}

	peerConn.Write([]byte("sending file: " + *file))
}

func handleInput(conn net.Conn) {

	switch os.Args[1] {

	case "send":
		handleSendCommand(conn)

	case "recv":

		listener, err := net.Listen("tcp", "localhost:0")

		addr := listener.Addr().String()

		fmt.Println("listening on " + addr)

		if err != nil {
			log.Fatal("Error when listening for direct connections.")
			os.Exit(1)
		}

		request := protocol.MakeMessage(protocol.SERVER_HELLO, []byte(addr))

		conn.Write(request)

		defer listener.Close()

		opCode, payload, err := protocol.ReadMessage(conn)

		validateResponse(err, protocol.SERVER_ACK, opCode, conn)

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

	for {

		message := make([]byte, 1024)
		bytesReceived, err := peerConn.Read(message)

		if err != nil {
			log.Println("error when reading from direct connection: " + peerConn.RemoteAddr().String())
			return
		}

		if bytesReceived == 0 {
			continue
		}

		log.Println(string(message))

	}

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
		log.Fatal("Missing Commnad: 'send' or 'recv'")
	}

	conn, err := net.Dial("tcp", ":8080")

	if err != nil {
		fmt.Println("Error Connecting to Communication Server:", err)
		os.Exit(1)
	}

	defer conn.Close()

	handleInput(conn)

}
