package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/KD0S-02/KDTransfer/internal/protocol"
)

func closeWithError(msg string, conn net.Conn) {
	log.Println(msg)
	conn.Close()
	os.Exit(1)
}

func validateResponse(err error, expectedOpCode byte,
	actualOpCode byte, conn net.Conn) {

	if err != nil {
		closeWithError("Server error! closing connection.", conn)
	}

	if actualOpCode != expectedOpCode {
		closeWithError("Error! unexpected server response.", conn)
	}

}

func generateTransferID(filename string, senderIP string) uint32 {
	timeStamp := time.Now().UnixNano()
	data := fmt.Sprintf("%s-%s-%d", filename, senderIP, timeStamp)
	hash := sha256.Sum256([]byte(data))
	return binary.BigEndian.Uint32(hash[:4])
}

func processFile(currFilePath string) (filename string, fileSize uint64, chunks [][]byte, err error) {
	data, err := os.ReadFile(currFilePath)

	if err != nil {
		return "", 0, nil, fmt.Errorf("failed to open file: %w", err)
	}

	for i := 0; i < len(data); i += protocol.CHUNK_SIZE {
		end := i + protocol.CHUNK_SIZE
		if end > len(data) {
			end = len(data)
		}
		chunk := data[i:end]
		chunks = append(chunks, chunk)
	}

	filename = filepath.Base(currFilePath)
	fileSize = uint64(len(data))

	return filename, fileSize, chunks, nil
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

	fmt.Println("Peer found:", string(payload))

	peerConn, err := net.Dial("tcp", string(payload))

	if err != nil {
		fmt.Println(err.Error())
		closeWithError("Failed to make direct connection to peer.", conn)
	}

	defer peerConn.Close()

	filename, fileSize, chunks, err := processFile(*file)

	if err != nil {
		closeWithError("Failed to process file: "+err.Error(), conn)
	}

	transferID := generateTransferID(filename, peerConn.LocalAddr().String())

	payload = protocol.CreateFileTransferStartPayload(transferID, filename, fileSize, uint32(len(chunks)))

	request = protocol.MakeMessage(protocol.FILE_TRANSFER_START, payload)

	_, err = peerConn.Write(request)

	if err != nil {
		closeWithError("Failed to send file transfer start message: "+err.Error(), conn)
	}

	log.Printf("Sending file: %s (ID: %d, Size: %d bytes, Chunks: %d)",
		filename, transferID, fileSize, len(chunks))

	for i, chunk := range chunks {
		payload = protocol.CreateFileTransferDataPayload(transferID, uint32(i), chunk)
		request = protocol.MakeMessage(protocol.FILE_TRANSFER_DATA, payload)

		_, err = peerConn.Write(request)
		if err != nil {
			closeWithError("Failed to send file chunk: "+err.Error(), conn)
		}

		log.Printf("Sent chunk %d for transfer ID %d", i, transferID)
	}

	payload = binary.BigEndian.AppendUint32(nil, transferID)
	request = protocol.MakeMessage(protocol.FILE_TRANSFER_END, payload)

	_, err = peerConn.Write(request)

	if err != nil {
		closeWithError("Failed to send file transfer end message: "+err.Error(), conn)
	}

	log.Printf("File transfer completed for ID: %d", transferID)
	fmt.Println("File transfer completed successfully.")
}

func handleInput(conn net.Conn) {

	switch os.Args[1] {

	case "send":
		handleSendCommand(conn)

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

	var (
		file       *os.File
		filename   string
		transferID uint32
		nChunks    uint32
		chunkIndex uint32
		chunkData  []byte
		fileSize   uint64
	)

	close := false

	for !close {

		opCode, payload, err := protocol.ReadMessage(peerConn)

		if err != nil {
			log.Println("error when reading from direct connection: " + peerConn.RemoteAddr().String())
			return
		}

		switch opCode {

		case protocol.FILE_TRANSFER_START:
			transferID, filename, fileSize, nChunks = protocol.ParseFileTransferPayload(payload)
			log.Printf("Receiving file: %s (ID: %d, Size: %d bytes, Chunks: %d)",
				filename, transferID, fileSize, nChunks)
			file, err = os.Create(filename)

			if err != nil {
				log.Println("Error creating file:", err)
				return
			}

		case protocol.FILE_TRANSFER_DATA:
			transferID, chunkIndex, chunkData = protocol.ParseFileTransferDataPayload(payload)
			log.Printf("Received chunk %d for transfer ID %d", chunkIndex, transferID)

			if file == nil {
				log.Println("file not open for writing:", err)
				return
			}

			n, err := file.Write(chunkData)
			if err != nil {
				log.Println("Error writing chunks to file:", err)
				return
			}
			log.Printf("Wrote %d bytes to file: %s", n, filename)

		case protocol.FILE_TRANSFER_END:
			log.Printf("File transfer completed for ID: %d", transferID)
			log.Println("File saved as:", filename)
			close = true
		}

	}

	file.Close()
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
