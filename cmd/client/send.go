package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/KD0S-02/KDTransfer/internal/protocol"
)

func processFile(currFilePath string, netp string) (filename string, fileSize uint64,
	chunks [][]byte, err error) {

	data, err := os.ReadFile(currFilePath)

	var chunkSize int

	if netp == "tcp" {
		chunkSize = protocol.TCP_CHUNK_SIZE
	} else {
		chunkSize = protocol.WEBRTC_CHUNK_SIZE
	}

	if err != nil {
		return "", 0, nil, fmt.Errorf("failed to open file: %w", err)
	}

	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
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

func raceConnections(addrinfo protocol.Addressinfo) (peerConn net.Conn, err error) {
	localAddrs := addrinfo.LocalAddr
	connChan := make(chan net.Conn, 1)

	for _, addr := range localAddrs {
		go func(address string) {
			conn, err := net.DialTimeout("tcp", address, 3*time.Second)
			if err != nil {
				return
			}
			select {
			case connChan <- conn:
			default:
				conn.Close()
			}
		}(addr)
	}

	select {
	case peerConn = <-connChan:
	case <-time.After(5 * time.Second):
		return peerConn, fmt.Errorf("failed to connect to peer")
	}

	return peerConn, nil
}

func HandleSendCommand(conn net.Conn) error {
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	file := sendCmd.String("file", "", "Path of file to be transferred.")
	peer := sendCmd.String("peer", "", "ID of receiving peer.")

	sendCmd.Parse(os.Args[2:])

	if *file == "" || *peer == "" {
		return fmt.Errorf("usage: send --file <filepath> --peer <peerID>")
	}

	request := protocol.MakeMessage(protocol.PEER_INFO_LOOKUP, []byte(*peer))
	conn.Write(request)

	opCode, payload, err := protocol.ReadMessage(conn)

	ValidateResponse(err, protocol.PEER_LOOKUP_ACK, opCode, conn)

	fmt.Println("Peer found:", string(payload))

	var addrinfo protocol.Addressinfo

	err = json.Unmarshal(payload, &addrinfo)

	if err != nil {
		return fmt.Errorf("failed to decode peer address: %s", err.Error())
	}

	peerConn, err := raceConnections(addrinfo)

	if err != nil {
		return err
	}

	defer peerConn.Close()

	filename, fileSize, chunks, err := processFile(*file, "tcp")

	if err != nil {
		return fmt.Errorf("failed to process file: %s", err.Error())
	}

	transferID := generateTransferID(filename, peerConn.LocalAddr().String())

	payload = protocol.CreateFileTransferStartPayload(transferID,
		filename, fileSize, uint32(len(chunks)))

	request = protocol.MakeMessage(protocol.FILE_TRANSFER_START, payload)

	_, err = peerConn.Write(request)

	if err != nil {
		return fmt.Errorf("failed to send file transfer start message: %s", err.Error())
	}

	log.Printf("Sending file: %s (ID: %d, Size: %d bytes, Chunks: %d)",
		filename, transferID, fileSize, len(chunks))

	ft := &FileTransfer{
		filename:   filename,
		transferID: transferID,
		filesize:   fileSize,
		startTime:  time.Now(),
	}

	Transfers.Store(transferID, ft)

	for i, chunk := range chunks {
		payload = protocol.CreateFileTransferDataPayload(transferID, uint32(i), chunk)
		request = protocol.MakeMessage(protocol.FILE_TRANSFER_DATA, payload)

		_, err = peerConn.Write(request)
		if err != nil {
			return fmt.Errorf("failed to send file chunk: %s", err.Error())
		}
	}

	payload = binary.BigEndian.AppendUint32(nil, transferID)
	request = protocol.MakeMessage(protocol.FILE_TRANSFER_END, payload)

	_, err = peerConn.Write(request)

	if err != nil {
		return fmt.Errorf("failed to send file transfer end message: %s", err.Error())
	}

	CompleteTransfer(transferID, "uploaded")

	return err
}
