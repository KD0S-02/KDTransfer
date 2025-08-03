package signallingserver

type PeerInfo struct {
	Type       PeerType
	LocalAddr  []string
	PublicAddr []string
}

type Peer struct {
	ID       string
	PeerInfo PeerInfo
	Outgoing chan []byte
}

type PeerLookUp struct {
	PeerID     string
	SenderInfo PeerInfo
}

type PeerType string

const (
	PeerTypeNative  PeerType = "native"
	PeerTypeBrowser PeerType = "browser"
)
