package network

import (
	"fmt"
	"testing"
)

func TestPublicIp(t *testing.T) {
	ip, err := PublicAddr()
	if err != nil {
		t.Fatalf("Failed to get public IP: %v", err)
	}
	fmt.Println(ip)
}
