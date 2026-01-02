package network

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/pion/stun"
)

type ConnType string

const (
	TCPConn     ConnType = "tcp"
	WEBRTCConn  ConnType = "webrtc"
	dialTimeout          = 2 * time.Second
	raceTimeout          = 3 * time.Second
)

func PublicAddr() (string, error) {
	u, err := stun.ParseURI("stun:stun.l.google.com:19302")
	if err != nil {
		return "", err
	}

	c, err := stun.DialURI(u, &stun.DialConfig{})
	if err != nil {
		return "", err
	}

	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	ip := ""
	if err := c.Do(message, func(res stun.Event) {
		if res.Error != nil {
			panic(res.Error)
		}

		var xorAddr stun.XORMappedAddress
		if err := xorAddr.GetFrom(res.Message); err != nil {
			panic(err)
		}
		ip = net.JoinHostPort(xorAddr.IP.String(), strconv.Itoa(xorAddr.Port))

	}); err != nil {
		return ip, err
	}

	return ip, nil
}

func RaceConnections(localAddrs []string) (peerConn net.Conn, connType ConnType, err error) {

	if len(localAddrs) == 0 {
		return nil, "", fmt.Errorf("no addresses to try")
	}

	connChan := make(chan net.Conn, 1)

	for _, addr := range localAddrs {
		go func(address string) {
			conn, err := net.DialTimeout("tcp", address,
				dialTimeout)
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
	case <-time.After(raceTimeout):
		return nil, "", fmt.Errorf("connection timeout: peer not reachable")
	}

	return peerConn, TCPConn, nil
}

func LocalAddresses(port string) (localAddrs []string,
	err error) {

	interfaces, err := net.Interfaces()

	if err != nil {
		err = fmt.Errorf("failed to get network interfaces: %s", err)
		return nil, err
	}

	for _, iface := range interfaces {

		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		if isVirtualInterface(iface.Name) {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP

			if ip.IsLoopback() || ip.IsMulticast() ||
				ip.IsLinkLocalUnicast() {
				continue
			}

			localAddrs = append(localAddrs, net.JoinHostPort(ip.String(),
				port))
		}
	}

	return localAddrs, nil
}

func isVirtualInterface(name string) bool {

	virtualPrefixes := []string{
		"docker",  // Docker bridge (docker0)
		"br-",     // Docker custom bridges
		"veth",    // Docker virtual ethernet
		"virbr",   // Libvirt/KVM bridges
		"vmnet",   // VMware interfaces
		"vboxnet", // VirtualBox host-only
		"utun",    // macOS VPN tunnels
		"tun",     // Generic tunnels
		"tap",     // TAP devices
	}

	lowerName := strings.ToLower(name)

	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(lowerName, prefix) {
			return true
		}
	}

	return false
}
