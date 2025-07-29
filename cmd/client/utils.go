package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

type FileTransfer struct {
	file          *os.File
	transferID    uint32
	filename      string
	filesize      uint64
	nChunks       uint32
	bytesReceived uint64
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

func GetAllLocalAddresses(port string) (localAddrs []string, err error) {

	addrs, err := net.InterfaceAddrs()

	if err != nil {
		err = fmt.Errorf("failed to get network interfaces: %s", err)
		return nil, err
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		ip := ipNet.IP

		if ip.IsLoopback() || ip.IsLinkLocalUnicast() ||
			ip.IsMulticast() || ip.To4() == nil {
			continue
		}

		localAddrs = append(localAddrs, net.JoinHostPort(ip.String(), port))
	}

	return localAddrs, nil
}
