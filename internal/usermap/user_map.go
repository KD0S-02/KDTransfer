package usermap

import (
	"sync"
)

type Peer struct {
	ID       string
	IP       string
	Port     string
	Outgoing chan []byte
}

var userMap sync.Map

// adds a user to the user map
func AddUser(id string, peer Peer) {
	userMap.Store(id, peer)
}

// removes a user from the user map
func RemoveUser(id string) {
	userMap.Delete(id)
}

// retrieves a user from the user map
func GetUser(id string) (Peer, bool) {
	if user, ok := userMap.Load(id); ok {
		return user.(Peer), true
	}
	return Peer{}, false
}
