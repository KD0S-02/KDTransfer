package usermap

import (
	"sync"

	"github.com/KD0S-02/KDTransfer/internal/protocol"
)

type Peer struct {
	ID       string
	Type     PeerType
	AddrInfo protocol.Addressinfo
	Outgoing chan []byte
}

type PeerType string

const (
	PeerTypeNative  PeerType = "native"
	PeerTypeBrowser PeerType = "browser"
)

var userMap sync.Map

func AddUser(id string, peer Peer) {
	userMap.Store(id, peer)
}

func RemoveUser(id string) {
	userMap.Delete(id)
}

func GetUser(id string) (Peer, bool) {
	if user, ok := userMap.Load(id); ok {
		return user.(Peer), true
	}
	return Peer{}, false
}
