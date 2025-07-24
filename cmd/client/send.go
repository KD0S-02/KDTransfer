package main

import (
	"crypto/sha256"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/KD0S-02/KDTransfer/internal/protocol"
)

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

func generateTransferID(filename string, senderIP string) uint32 {
	timeStamp := time.Now().UnixNano()
	data := fmt.Sprintf("%s-%s-%d", filename, senderIP, timeStamp)
	hash := sha256.Sum256([]byte(data))
	return binary.BigEndian.Uint32(hash[:4])
}

func HandleSendCommand(conn net.Conn) {
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

	ValidateResponse(err, protocol.PEER_LOOKUP_ACK, opCode, conn)

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
