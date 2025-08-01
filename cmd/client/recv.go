package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/KD0S-02/KDTransfer/internal/config"
	"github.com/KD0S-02/KDTransfer/internal/protocol"
)

func HandleReceiveCommand(conn net.Conn) error {

	config := config.LoadConfig()

	listener, err := net.Listen("tcp", "0.0.0.0:"+config.TCPPort)

	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	defer listener.Close()

	localAddrs, err := GetAllLocalAddresses(config.TCPPort)

	if err != nil {
		return fmt.Errorf("error while reading network address: %s",
			err.Error())
	}

	addrInfo := &protocol.Addressinfo{
		LocalAddr: localAddrs,
	}

	addrs, err := json.Marshal(addrInfo)

	if err != nil {
		return fmt.Errorf("error while encoding network addrs as json: %s",
			err.Error())
	}

	request := protocol.MakeMessage(protocol.SERVER_HELLO, []byte(addrs))
	if _, err := conn.Write(request); err != nil {
		return fmt.Errorf("failed to send port to server: %w", err)
	}

	_, payload, err := protocol.ReadMessage(conn)
	if err != nil {
		return fmt.Errorf("failed to read server response: %w", err)
	}

	log.Println("Current ID:", string(payload))

	go func() {
		if err := listenForDirectConnections(listener); err != nil {
			log.Printf("error in peer listener: %v", err)
		}
	}()

	return waitForUserInput()
}

func waitForUserInput() error {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Type 'disconnect' to exit")
	for scanner.Scan() {
		if scanner.Text() == "disconnect" {
			return nil
		}
	}
	return scanner.Err()
}

func listenForDirectConnections(listener net.Listener) error {
	for {
		peerConn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("failed to accept connection: %w", err)
		}
		go handlePeerConnection(peerConn)
	}
}

func handlePeerConnection(conn net.Conn) {
	defer conn.Close()

	for {
		shouldClose, err := handleMessages(conn)

		if err != nil {
			_ = fmt.Errorf("%s", err.Error())
			return
		}

		if shouldClose {
			return
		}
	}
}

func handleMessages(peerConn net.Conn) (close bool, err error) {

	opCode, payload, err := protocol.ReadMessage(peerConn)

	if err != nil {
		return true, err
	}

	switch opCode {

	case protocol.FILE_TRANSFER_START:
		transferID, filename, filesize, nChunks := protocol.ParseFileTransferPayload(payload)

		ft := &FileTransfer{
			transferID: transferID,
			filename:   filename,
			filesize:   filesize,
			startTime:  time.Now(),
		}

		log.Printf("Sending file: %s (ID: %d, Size: %d bytes, Chunks: %d)",
			filename, transferID, filesize, nChunks)

		file, err := os.Create(filename)
		ft.file = file

		Transfers.Store(transferID, ft)

		if err != nil {
			return true, err
		}

	case protocol.FILE_TRANSFER_DATA:
		transferID, _, chunkData := protocol.ParseFileTransferDataPayload(payload)

		ft, ok := GetTransfer(transferID)

		if !ok {
			log.Println("File transfer not found for ID:", transferID)
			return true, err
		}

		file := ft.file

		if file == nil {
			log.Println("file not open for writing:", err)
			return true, err
		}

		_, err := file.Write(chunkData)
		if err != nil {
			closeWithError("Failed to write chunk to file: "+err.Error(), peerConn)
		}

	case protocol.FILE_TRANSFER_END:
		transferID := binary.BigEndian.Uint32(payload)
		ft, ok := GetTransfer(transferID)

		if !ok {
			return true, err
		}

		CompleteTransfer(ft.transferID, "received")

		return true, err
	}

	return false, err
}
