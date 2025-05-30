package usermap

import (
	"sync"
)

type UserInfo struct {
	ID string
	IP string
}

var userMap sync.Map

// AddUser adds a user to the user map
func AddUser(id string, ip string) {
	userMap.Store(id, UserInfo{ID: id, IP: ip})
}

// GetUser retrieves a user from the user map
func getUser(id string) (UserInfo, bool) {
	if user, ok := userMap.Load(id); ok {
		return user.(UserInfo), true
	}
	return UserInfo{}, false
}
