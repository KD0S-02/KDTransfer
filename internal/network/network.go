package network

import (
	"fmt"
	"net"
	"time"
)

type ConnType string

const (
	TCPConn    ConnType = "tcp"
	WEBRTCConn ConnType = "webrtc"
)

func RaceConnections(localAddrs []string) (peerConn net.Conn, connType ConnType, err error) {
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
		return peerConn, "", fmt.Errorf("failed to connect to peer")
	}

	return peerConn, TCPConn, nil
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
