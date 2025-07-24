package main

import (
	"log"
	"net"
	"os"
)

type FileTransfer struct {
	file       *os.File
	transferID uint32
	filename   string
	filesize   uint64
	nChunks    uint32
}

func closeWithError(msg string, conn net.Conn) {
	log.Println(msg)
	conn.Close()
	os.Exit(1)
}

func ValidateResponse(err error, expectedOpCode byte,
	actualOpCode byte, conn net.Conn) {

	if err != nil {
		closeWithError("Server error! closing connection.", conn)
	}

	if actualOpCode != expectedOpCode {
		closeWithError("Error! unexpected server response.", conn)
	}

}
